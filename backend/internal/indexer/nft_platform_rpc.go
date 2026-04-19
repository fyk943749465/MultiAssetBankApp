package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 事件 topic0（与 subgraph/nft-platform/abis 中 ABI 一致）。
var (
	topicERC721Transfer = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

	topicCollectionCreated  = crypto.Keccak256Hash([]byte("CollectionCreated(address,address,uint256,bytes32)"))
	topicCreationFeeUpdated = crypto.Keccak256Hash([]byte("CreationFeeUpdated(uint256,uint256)"))
	topicEthReceived        = crypto.Keccak256Hash([]byte("EthReceived(address,uint256)"))
	topicFactoryOwnershipTr = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))
	topicPaused             = crypto.Keccak256Hash([]byte("Paused(address)"))
	topicUnpaused           = crypto.Keccak256Hash([]byte("Unpaused(address)"))
	topicRefundSent         = crypto.Keccak256Hash([]byte("RefundSent(address,uint256)"))
	topicWithdrawal         = crypto.Keccak256Hash([]byte("Withdrawal(address,uint256)"))

	topicItemListed            = crypto.Keccak256Hash([]byte("ItemListed(address,uint256,address,uint256)"))
	topicItemSold              = crypto.Keccak256Hash([]byte("ItemSold(address,uint256,address,address,uint256,uint256,uint256,uint256)"))
	topicListingCanceled       = crypto.Keccak256Hash([]byte("ListingCanceled(address,uint256,address)"))
	topicListingPriceUpdated   = crypto.Keccak256Hash([]byte("ListingPriceUpdated(address,uint256,address,uint256,uint256)"))
	topicPlatformFeeUpdated    = crypto.Keccak256Hash([]byte("PlatformFeeUpdated(uint256,uint256)"))
	topicMaxRoyaltyBpsUpdated  = crypto.Keccak256Hash([]byte("MaxRoyaltyBpsUpdated(uint256)"))
	topicPlatformFeesWithdrawn = crypto.Keccak256Hash([]byte("PlatformFeesWithdrawn(address,uint256)"))
	topicUntrackedEthWithdrawn = crypto.Keccak256Hash([]byte("UntrackedEthWithdrawn(address,uint256)"))

	topicApproval          = crypto.Keccak256Hash([]byte("Approval(address,address,uint256)"))
	topicApprovalForAll    = crypto.Keccak256Hash([]byte("ApprovalForAll(address,address,bool)"))
	topicBaseURIUpdated    = crypto.Keccak256Hash([]byte("BaseURIUpdated(string)"))
	topicDefaultRoyaltyUpd = crypto.Keccak256Hash([]byte("DefaultRoyaltyUpdated(address,uint96)"))
	topicInitialized       = crypto.Keccak256Hash([]byte("Initialized(uint64)"))
)

const nftCollectionAddrBatch = 180

// NFTPlatformRPC 扫 NFTFactory、NFTMarketPlace、以及库中已登记的全部 nft_collection 合约日志，写入 005_nft_platform.sql 定义的表。
type NFTPlatformRPC struct {
	mu        sync.Mutex
	DB        *gorm.DB
	Eth       *ethclient.Client
	ChainID   uint64
	ChainID64 int64

	FactoryAddr common.Address
	FactoryID   uint64
	MarketAddr  common.Address
	MarketID    uint64

	cursorName string
}

// NewNFTPlatformRPC 从 nft_contracts 读取本链的工厂与市场地址；要求迁移已种子工厂/市场行。
func NewNFTPlatformRPC(db *gorm.DB, eth *ethclient.Client, chainID uint64) (*NFTPlatformRPC, error) {
	if db == nil || eth == nil {
		return nil, errors.New("nft platform rpc indexer: nil db or eth")
	}
	cid := int64(chainID)

	var factory models.NFTContract
	if err := db.Where("chain_id = ? AND contract_kind = ?", cid, "nft_factory").First(&factory).Error; err != nil {
		return nil, fmt.Errorf("nft platform rpc indexer: nft_factory row for chain_id=%d: %w", chainID, err)
	}
	var market models.NFTContract
	if err := db.Where("chain_id = ? AND contract_kind = ?", cid, "nft_marketplace").First(&market).Error; err != nil {
		return nil, fmt.Errorf("nft platform rpc indexer: nft_marketplace row for chain_id=%d: %w", chainID, err)
	}

	fa := common.HexToAddress(factory.Address)
	ma := common.HexToAddress(market.Address)

	return &NFTPlatformRPC{
		DB:          db,
		Eth:         eth,
		ChainID:     chainID,
		ChainID64:   cid,
		FactoryAddr: fa,
		FactoryID:   factory.ID,
		MarketAddr:  ma,
		MarketID:    market.ID,
		cursorName:  fmt.Sprintf("nft_platform_rpc_%d", chainID),
	}, nil
}

// Run 定时扫块（阻塞，请在 goroutine 中调用）。
func (n *NFTPlatformRPC) Run(ctx context.Context) {
	t := time.NewTicker(PollInterval())
	defer t.Stop()
	for {
		if err := n.SyncOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("nft platform rpc indexer: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (n *NFTPlatformRPC) factoryAndMarketTopics() []common.Hash {
	// 同一 topic0 可能出现在工厂与市场（如 Paused）；按 lg.Address 再分流。
	return []common.Hash{
		topicCollectionCreated, topicCreationFeeUpdated, topicEthReceived, topicFactoryOwnershipTr,
		topicPaused, topicUnpaused, topicRefundSent, topicWithdrawal,
		topicItemListed, topicItemSold, topicListingCanceled, topicListingPriceUpdated,
		topicPlatformFeeUpdated, topicMaxRoyaltyBpsUpdated,
		topicPlatformFeesWithdrawn, topicUntrackedEthWithdrawn,
	}
}

func (n *NFTPlatformRPC) collectionTopics() []common.Hash {
	return []common.Hash{
		topicERC721Transfer,
		topicApproval, topicApprovalForAll,
		topicBaseURIUpdated, topicDefaultRoyaltyUpd, topicFactoryOwnershipTr,
		topicInitialized,
	}
}

// SyncOnce 处理已确认区块内的平台日志。
func (n *NFTPlatformRPC) SyncOnce(ctx context.Context) error {
	if n == nil || n.DB == nil || n.Eth == nil {
		return errors.New("nft platform rpc indexer: not configured")
	}
	n.mu.Lock()
	defer n.mu.Unlock()

	head, err := ethHeaderByNumber(ctx, n.Eth, nil)
	if err != nil {
		log.Printf("nft platform rpc indexer: 拉取链头失败: %v", err)
		return err
	}
	safe, err := ethConfirmedTip(ctx, n.Eth, head)
	if err != nil {
		return err
	}
	last, err := n.getOrInitCursor(ctx, safe)
	if err != nil {
		return err
	}
	if last >= safe {
		return nil
	}
	from := last + 1
	to := safe
	if from > to {
		return nil
	}

	var collStr []string
	if err := n.DB.WithContext(ctx).Model(&models.NFTContract{}).
		Where("chain_id = ? AND contract_kind = ?", n.ChainID64, "nft_collection").
		Pluck("address", &collStr).Error; err != nil {
		return err
	}
	collAddrs := make([]common.Address, 0, len(collStr))
	for _, s := range collStr {
		if !common.IsHexAddress(s) {
			continue
		}
		collAddrs = append(collAddrs, common.HexToAddress(s))
	}

	fmTopics := n.factoryAndMarketTopics()
	logsFM, err := n.filterLogsChunked(ctx, []common.Address{n.FactoryAddr, n.MarketAddr}, [][]common.Hash{fmTopics}, from, to)
	if err != nil {
		return err
	}
	colTopics := n.collectionTopics()
	var logsColl []types.Log
	for i := 0; i < len(collAddrs); i += nftCollectionAddrBatch {
		end := i + nftCollectionAddrBatch
		if end > len(collAddrs) {
			end = len(collAddrs)
		}
		chunk, err := n.filterLogsChunked(ctx, collAddrs[i:end], [][]common.Hash{colTopics}, from, to)
		if err != nil {
			return err
		}
		logsColl = append(logsColl, chunk...)
	}

	logs := append(logsFM, logsColl...)
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		if logs[i].TxIndex != logs[j].TxIndex {
			return logs[i].TxIndex < logs[j].TxIndex
		}
		return logs[i].Index < logs[j].Index
	})

	blockTimeCache := make(map[uint64]time.Time)
	getTime := func(num uint64) (time.Time, error) {
		if t0, ok := blockTimeCache[num]; ok {
			return t0, nil
		}
		hdr, err := ethHeaderByNumber(ctx, n.Eth, new(big.Int).SetUint64(num))
		if err != nil {
			return time.Time{}, err
		}
		ts := time.Unix(int64(hdr.Time), 0).UTC()
		blockTimeCache[num] = ts
		return ts, nil
	}

	return n.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, lg := range logs {
			ts, err := getTime(lg.BlockNumber)
			if err != nil {
				return err
			}
			if err := n.ingestLog(tx, lg, ts); err != nil {
				return err
			}
		}
		return tx.Model(&models.ChainIndexerCursor{}).
			Where("name = ?", n.cursorName).
			Updates(map[string]any{
				"last_scanned_block": to,
				"updated_at":         time.Now().UTC(),
			}).Error
	})
}

func (n *NFTPlatformRPC) filterLogsChunked(ctx context.Context, addrs []common.Address, topics [][]common.Hash, from, to uint64) ([]types.Log, error) {
	if len(addrs) == 0 {
		return nil, nil
	}
	base := ethereum.FilterQuery{
		Addresses: addrs,
		Topics:    topics,
	}
	var out []types.Log
	for chunkFrom := from; chunkFrom <= to; {
		chunkTo := chunkFrom + maxBlockSpan() - 1
		if chunkTo > to {
			chunkTo = to
		}
		q := base
		q.FromBlock = new(big.Int).SetUint64(chunkFrom)
		q.ToBlock = new(big.Int).SetUint64(chunkTo)
		var chunk []types.Log
		err := ethWithRPCRetry(ctx, func() error {
			var e error
			chunk, e = n.Eth.FilterLogs(ctx, q)
			return e
		})
		if err != nil {
			return nil, err
		}
		out = append(out, chunk...)
		chunkFrom = chunkTo + 1
		if chunkFrom <= to {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(filterChunkPause()):
			}
		}
	}
	return out, nil
}

func (n *NFTPlatformRPC) getOrInitCursor(ctx context.Context, safe uint64) (uint64, error) {
	var cur models.ChainIndexerCursor
	err := n.DB.WithContext(ctx).Where("name = ?", n.cursorName).First(&cur).Error
	if err == nil {
		return cur.LastScannedBlock, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	// 与 Bank / Code Pulse 一致：无游标行时从「已确认头往前 lookbackBlocks」起扫；补历史则删游标后重启。
	var start uint64
	if safe > lookbackBlocks {
		start = safe - lookbackBlocks
	} else {
		start = 1
	}
	last := start - 1
	if safe > 0 && last >= safe {
		last = safe - 1
	}
	cur = models.ChainIndexerCursor{
		Name:             n.cursorName,
		LastScannedBlock: last,
		UpdatedAt:        time.Now().UTC(),
	}
	if err := n.DB.WithContext(ctx).Create(&cur).Error; err != nil {
		return 0, err
	}
	return last, nil
}

func (n *NFTPlatformRPC) ingestLog(tx *gorm.DB, lg types.Log, blockTime time.Time) error {
	if len(lg.Topics) == 0 {
		return nil
	}
	addr := strings.ToLower(lg.Address.Hex())
	t0 := lg.Topics[0]

	switch {
	case strings.EqualFold(addr, n.FactoryAddr.Hex()):
		return n.ingestFactoryLog(tx, lg, blockTime, t0)
	case strings.EqualFold(addr, n.MarketAddr.Hex()):
		return n.ingestMarketplaceLog(tx, lg, blockTime, t0)
	default:
		return n.ingestCollectionLog(tx, lg, blockTime, t0, addr)
	}
}

func (n *NFTPlatformRPC) ingestFactoryLog(tx *gorm.DB, lg types.Log, blockTime time.Time, t0 common.Hash) error {
	txHash := lg.TxHash.Hex()
	logIndex := int(lg.Index)
	bn := int64(lg.BlockNumber)

	switch t0 {
	case topicCollectionCreated:
		if len(lg.Topics) < 3 || len(lg.Data) < 64 {
			return nil
		}
		collectionAddr := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
		creatorAddr := common.BytesToAddress(lg.Topics[2].Bytes()[12:])
		feePaid := new(big.Int).SetBytes(lg.Data[:32])
		salt := lg.Data[32:64]
		saltHex := ("0x" + common.Bytes2Hex(salt))
		var saltPtr *string
		if !isZeroBytes32(salt) {
			saltPtr = &saltHex
		}

		creatorID, err := n.ensureAccount(tx, creatorAddr)
		if err != nil {
			return err
		}

		collAddrLower := strings.ToLower(collectionAddr.Hex())
		var contractRow models.NFTContract
		err = tx.Where("chain_id = ? AND lower(address) = ?", n.ChainID64, collAddrLower).First(&contractRow).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contractRow = models.NFTContract{
				ChainID:        n.ChainID64,
				Address:        collAddrLower,
				ContractKind:   "nft_collection",
				DeployedBlock:  &bn,
				DeployedTxHash: &txHash,
				CreatedAt:      time.Now().UTC(),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "chain_id"}, {Name: "address"}},
				DoNothing: true,
			}).Create(&contractRow).Error; err != nil {
				return err
			}
			if err := tx.Where("chain_id = ? AND lower(address) = ?", n.ChainID64, collAddrLower).First(&contractRow).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		feeStr := feePaid.String()
		col := models.NFTCollection{
			ChainID:            n.ChainID64,
			ContractID:         contractRow.ID,
			CreatorAccountID:   creatorID,
			FeePaidWei:         &feeStr,
			DeploySaltHex:      saltPtr,
			CreatedBlockNumber: bn,
			CreatedTxHash:      txHash,
			CreatedLogIndex:    logIndex,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "created_tx_hash"}, {Name: "created_log_index"}},
			DoNothing: true,
		}).Create(&col).Error; err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"collection": collAddrLower,
			"creator":    strings.ToLower(creatorAddr.Hex()),
			"feePaid":    feeStr,
			"salt":       saltHex,
		})
		fev := models.NFTFactoryEvent{
			ChainID:           n.ChainID64,
			FactoryContractID: n.FactoryID,
			EventType:         "CollectionCreated",
			BlockNumber:       bn,
			BlockTime:         blockTime,
			TxHash:            txHash,
			LogIndex:          logIndex,
			PayloadJSON:       payload,
			CreatedAt:         time.Now().UTC(),
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
			DoNothing: true,
		}).Create(&fev).Error

	case topicCreationFeeUpdated:
		if len(lg.Data) < 64 {
			return nil
		}
		oldF := new(big.Int).SetBytes(lg.Data[:32]).String()
		newF := new(big.Int).SetBytes(lg.Data[32:64]).String()
		payload, _ := json.Marshal(map[string]any{"oldFee": oldF, "newFee": newF})
		return n.insertFactoryEvent(tx, "CreationFeeUpdated", bn, blockTime, txHash, logIndex, payload)

	case topicEthReceived:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		from := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		amt := new(big.Int).SetBytes(lg.Data[:32]).String()
		payload, _ := json.Marshal(map[string]any{"from": strings.ToLower(from), "amount": amt})
		return n.insertFactoryEvent(tx, "EthReceived", bn, blockTime, txHash, logIndex, payload)

	case topicRefundSent:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		to := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		amt := new(big.Int).SetBytes(lg.Data[:32]).String()
		payload, _ := json.Marshal(map[string]any{"to": strings.ToLower(to), "amount": amt})
		return n.insertFactoryEvent(tx, "RefundSent", bn, blockTime, txHash, logIndex, payload)

	case topicWithdrawal:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		to := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		amt := new(big.Int).SetBytes(lg.Data[:32]).String()
		payload, _ := json.Marshal(map[string]any{"to": strings.ToLower(to), "amount": amt})
		return n.insertFactoryEvent(tx, "Withdrawal", bn, blockTime, txHash, logIndex, payload)

	case topicPaused, topicUnpaused:
		acct, err := n.decodeAddressFromData(lg.Data)
		if err != nil {
			return nil
		}
		evName := "Paused"
		if t0 == topicUnpaused {
			evName = "Unpaused"
		}
		payload, _ := json.Marshal(map[string]any{"account": acct})
		return n.insertFactoryEvent(tx, evName, bn, blockTime, txHash, logIndex, payload)

	case topicFactoryOwnershipTr:
		if len(lg.Topics) < 3 {
			return nil
		}
		prev := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		newO := common.BytesToAddress(lg.Topics[2].Bytes()[12:]).Hex()
		payload, _ := json.Marshal(map[string]any{"previousOwner": strings.ToLower(prev), "newOwner": strings.ToLower(newO)})
		return n.insertFactoryEvent(tx, "OwnershipTransferred", bn, blockTime, txHash, logIndex, payload)

	default:
		return nil
	}
}

func (n *NFTPlatformRPC) insertFactoryEvent(tx *gorm.DB, typ string, bn int64, blockTime time.Time, txHash string, logIndex int, payload []byte) error {
	fev := models.NFTFactoryEvent{
		ChainID:           n.ChainID64,
		FactoryContractID: n.FactoryID,
		EventType:         typ,
		BlockNumber:       bn,
		BlockTime:         blockTime,
		TxHash:            txHash,
		LogIndex:          logIndex,
		PayloadJSON:       payload,
		CreatedAt:         time.Now().UTC(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
		DoNothing: true,
	}).Create(&fev).Error
}

func (n *NFTPlatformRPC) ingestMarketplaceLog(tx *gorm.DB, lg types.Log, blockTime time.Time, t0 common.Hash) error {
	txHash := lg.TxHash.Hex()
	logIndex := int(lg.Index)
	bn := int64(lg.BlockNumber)

	switch t0 {
	case topicItemListed:
		if len(lg.Topics) < 4 || len(lg.Data) < 32 {
			return nil
		}
		collection := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
		tokenID := new(big.Int).SetBytes(lg.Topics[2].Bytes()).String()
		seller := common.BytesToAddress(lg.Topics[3].Bytes()[12:])
		price := new(big.Int).SetBytes(lg.Data[:32]).String()
		collLower := strings.ToLower(collection.Hex())

		sellerID, err := n.ensureAccount(tx, seller)
		if err != nil {
			return err
		}

		tr := models.NFTMarketTradeEvent{
			ChainID:               n.ChainID64,
			MarketplaceContractID: n.MarketID,
			EventType:             "ItemListed",
			CollectionAddress:     collLower,
			TokenID:               tokenID,
			SellerAccountID:       &sellerID,
			PriceWei:              &price,
			BlockNumber:           bn,
			BlockTime:             blockTime,
			TxHash:                txHash,
			LogIndex:              logIndex,
			CreatedAt:             time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
			DoNothing: true,
		}).Create(&tr).Error; err != nil {
			return err
		}

		listing := models.NFTActiveListing{
			ChainID:               n.ChainID64,
			MarketplaceContractID: n.MarketID,
			CollectionAddress:     collLower,
			TokenID:               tokenID,
			SellerAccountID:       sellerID,
			PriceWei:              price,
			ListedBlockNumber:     bn,
			ListedTxHash:          txHash,
			ListingStatus:         "active",
			CreatedAt:             time.Now().UTC(),
			UpdatedAt:             time.Now().UTC(),
		}
		if err := tx.Create(&listing).Error; err != nil {
			if isPostgresDuplicateKey(err) {
				return nil
			}
			return err
		}
		return nil

	case topicListingPriceUpdated:
		if len(lg.Topics) < 4 || len(lg.Data) < 64 {
			return nil
		}
		collection := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
		tokenID := new(big.Int).SetBytes(lg.Topics[2].Bytes()).String()
		seller := common.BytesToAddress(lg.Topics[3].Bytes()[12:])
		oldP := new(big.Int).SetBytes(lg.Data[:32]).String()
		newP := new(big.Int).SetBytes(lg.Data[32:64]).String()
		collLower := strings.ToLower(collection.Hex())
		sellerID, err := n.ensureAccount(tx, seller)
		if err != nil {
			return err
		}
		tr := models.NFTMarketTradeEvent{
			ChainID:               n.ChainID64,
			MarketplaceContractID: n.MarketID,
			EventType:             "ListingPriceUpdated",
			CollectionAddress:     collLower,
			TokenID:               tokenID,
			SellerAccountID:       &sellerID,
			OldPriceWei:           &oldP,
			NewPriceWei:           &newP,
			BlockNumber:           bn,
			BlockTime:             blockTime,
			TxHash:                txHash,
			LogIndex:              logIndex,
			CreatedAt:             time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
			DoNothing: true,
		}).Create(&tr).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&models.NFTActiveListing{}).
			Where("chain_id = ? AND collection_address = ? AND token_id = ? AND seller_account_id = ? AND listing_status = ?",
				n.ChainID64, collLower, tokenID, sellerID, "active").
			Updates(map[string]any{
				"price_wei":           newP,
				"listed_block_number": bn,
				"listed_tx_hash":      txHash,
				"updated_at":          now,
			}).Error

	case topicListingCanceled:
		if len(lg.Topics) < 4 {
			return nil
		}
		collection := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
		tokenID := new(big.Int).SetBytes(lg.Topics[2].Bytes()).String()
		seller := common.BytesToAddress(lg.Topics[3].Bytes()[12:])
		collLower := strings.ToLower(collection.Hex())
		sellerID, err := n.ensureAccount(tx, seller)
		if err != nil {
			return err
		}
		tr := models.NFTMarketTradeEvent{
			ChainID:               n.ChainID64,
			MarketplaceContractID: n.MarketID,
			EventType:             "ListingCanceled",
			CollectionAddress:     collLower,
			TokenID:               tokenID,
			SellerAccountID:       &sellerID,
			BlockNumber:           bn,
			BlockTime:             blockTime,
			TxHash:                txHash,
			LogIndex:              logIndex,
			CreatedAt:             time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
			DoNothing: true,
		}).Create(&tr).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		return n.closeActiveListing(tx, collLower, tokenID, sellerID, "cancelled", now, txHash, logIndex)

	case topicItemSold:
		if len(lg.Topics) < 4 || len(lg.Data) < 160 {
			return nil
		}
		collection := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
		tokenID := new(big.Int).SetBytes(lg.Topics[2].Bytes()).String()
		seller := common.BytesToAddress(lg.Topics[3].Bytes()[12:])
		buyer := common.BytesToAddress(lg.Data[12:32])
		price := new(big.Int).SetBytes(lg.Data[32:64]).String()
		pf := new(big.Int).SetBytes(lg.Data[64:96]).String()
		roy := new(big.Int).SetBytes(lg.Data[96:128]).String()
		feeSnap := new(big.Int).SetBytes(lg.Data[128:160]).String()
		collLower := strings.ToLower(collection.Hex())
		sellerID, err := n.ensureAccount(tx, seller)
		if err != nil {
			return err
		}
		buyerID, err := n.ensureAccount(tx, buyer)
		if err != nil {
			return err
		}
		tr := models.NFTMarketTradeEvent{
			ChainID:               n.ChainID64,
			MarketplaceContractID: n.MarketID,
			EventType:             "ItemSold",
			CollectionAddress:     collLower,
			TokenID:               tokenID,
			SellerAccountID:       &sellerID,
			BuyerAccountID:        &buyerID,
			PriceWei:              &price,
			PlatformFeeWei:        &pf,
			RoyaltyAmountWei:      &roy,
			FeeBpsSnapshot:        &feeSnap,
			BlockNumber:           bn,
			BlockTime:             blockTime,
			TxHash:                txHash,
			LogIndex:              logIndex,
			CreatedAt:             time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
			DoNothing: true,
		}).Create(&tr).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		return n.closeActiveListing(tx, collLower, tokenID, sellerID, "sold", now, txHash, logIndex)

	case topicPlatformFeeUpdated:
		if len(lg.Data) < 64 {
			return nil
		}
		oldB := new(big.Int).SetBytes(lg.Data[:32]).String()
		newB := new(big.Int).SetBytes(lg.Data[32:64]).String()
		payload, _ := json.Marshal(map[string]any{"oldBps": oldB, "newBps": newB})
		return n.insertMarketAdmin(tx, "PlatformFeeUpdated", bn, blockTime, txHash, logIndex, payload)

	case topicMaxRoyaltyBpsUpdated:
		if len(lg.Data) < 32 {
			return nil
		}
		newB := new(big.Int).SetBytes(lg.Data[:32]).String()
		payload, _ := json.Marshal(map[string]any{"newBps": newB})
		return n.insertMarketAdmin(tx, "MaxRoyaltyBpsUpdated", bn, blockTime, txHash, logIndex, payload)

	case topicPaused, topicUnpaused:
		acct, err := n.decodeAddressFromData(lg.Data)
		if err != nil {
			return nil
		}
		evName := "Paused"
		if t0 == topicUnpaused {
			evName = "Unpaused"
		}
		payload, _ := json.Marshal(map[string]any{"account": acct})
		return n.insertMarketAdmin(tx, evName, bn, blockTime, txHash, logIndex, payload)

	case topicFactoryOwnershipTr:
		if len(lg.Topics) < 3 {
			return nil
		}
		prev := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		newO := common.BytesToAddress(lg.Topics[2].Bytes()[12:]).Hex()
		payload, _ := json.Marshal(map[string]any{"previousOwner": strings.ToLower(prev), "newOwner": strings.ToLower(newO)})
		return n.insertMarketAdmin(tx, "OwnershipTransferred", bn, blockTime, txHash, logIndex, payload)

	case topicPlatformFeesWithdrawn:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		to := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		amt := new(big.Int).SetBytes(lg.Data[:32]).String()
		payload, _ := json.Marshal(map[string]any{"to": strings.ToLower(to), "amount": amt})
		return n.insertMarketAdmin(tx, "PlatformFeesWithdrawn", bn, blockTime, txHash, logIndex, payload)

	case topicUntrackedEthWithdrawn:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		to := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		amt := new(big.Int).SetBytes(lg.Data[:32]).String()
		payload, _ := json.Marshal(map[string]any{"to": strings.ToLower(to), "amount": amt})
		return n.insertMarketAdmin(tx, "UntrackedEthWithdrawn", bn, blockTime, txHash, logIndex, payload)

	default:
		return nil
	}
}

func (n *NFTPlatformRPC) insertMarketAdmin(tx *gorm.DB, typ string, bn int64, blockTime time.Time, txHash string, logIndex int, payload []byte) error {
	ev := models.NFTMarketplaceAdminEvent{
		ChainID:               n.ChainID64,
		MarketplaceContractID: n.MarketID,
		EventType:             typ,
		BlockNumber:           bn,
		BlockTime:             blockTime,
		TxHash:                txHash,
		LogIndex:              logIndex,
		PayloadJSON:           payload,
		CreatedAt:             time.Now().UTC(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
		DoNothing: true,
	}).Create(&ev).Error
}

func (n *NFTPlatformRPC) closeActiveListing(tx *gorm.DB, collLower, tokenID string, sellerID uint64, status string, closedAt time.Time, closeTx string, closeLog int) error {
	updates := map[string]any{
		"listing_status":  status,
		"closed_at":       closedAt,
		"close_tx_hash":   closeTx,
		"close_log_index": closeLog,
		"updated_at":      closedAt,
	}
	return tx.Model(&models.NFTActiveListing{}).
		Where("chain_id = ? AND collection_address = ? AND token_id = ? AND seller_account_id = ? AND listing_status = ?",
			n.ChainID64, collLower, tokenID, sellerID, "active").
		Updates(updates).Error
}

func (n *NFTPlatformRPC) ingestCollectionLog(tx *gorm.DB, lg types.Log, blockTime time.Time, t0 common.Hash, addrLower string) error {
	collectionID, ok := n.lookupCollectionID(tx, addrLower)
	if !ok {
		return nil
	}
	txHash := lg.TxHash.Hex()
	logIndex := int(lg.Index)
	bn := int64(lg.BlockNumber)

	switch t0 {
	case topicERC721Transfer:
		if len(lg.Topics) < 4 {
			return nil
		}
		from := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
		to := common.BytesToAddress(lg.Topics[2].Bytes()[12:])
		tokenID := new(big.Int).SetBytes(lg.Topics[3].Bytes()).String()

		toID, err := n.ensureAccount(tx, to)
		if err != nil {
			return err
		}
		var fromID *uint64
		if from != (common.Address{}) {
			fid, err := n.ensureAccount(tx, from)
			if err != nil {
				return err
			}
			fromID = &fid
		}

		tr := models.NFTTransfer{
			ChainID:       n.ChainID64,
			CollectionID:  collectionID,
			TokenID:       tokenID,
			FromAccountID: fromID,
			ToAccountID:   toID,
			BlockNumber:   bn,
			BlockTime:     blockTime,
			TxHash:        txHash,
			LogIndex:      logIndex,
			CreatedAt:     time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
			DoNothing: true,
		}).Create(&tr).Error; err != nil {
			return err
		}

		isMint := from == (common.Address{})
		var mintTx *string
		var mintBn *int64
		if isMint {
			mintTx = &txHash
			mintBn = &bn
		}

		var tok models.NFTToken
		err = tx.Where("collection_id = ? AND token_id = ?", collectionID, tokenID).First(&tok).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row := models.NFTToken{
				ChainID:            n.ChainID64,
				CollectionID:       collectionID,
				TokenID:            tokenID,
				OwnerAccountID:     toID,
				MintTxHash:         mintTx,
				MintBlockNumber:    mintBn,
				LastTransferTxHash: &txHash,
				LastTransferBlock:  &bn,
				UpdatedAt:          time.Now().UTC(),
			}
			return tx.Create(&row).Error
		}
		if err != nil {
			return err
		}
		updates := map[string]any{
			"owner_account_id":      toID,
			"last_transfer_tx_hash": txHash,
			"last_transfer_block":   bn,
			"updated_at":            time.Now().UTC(),
		}
		if isMint && tok.MintTxHash == nil {
			updates["mint_tx_hash"] = txHash
			updates["mint_block_number"] = bn
		}
		return tx.Model(&models.NFTToken{}).Where("id = ?", tok.ID).Updates(updates).Error

	case topicBaseURIUpdated:
		s, err := unpackABIString(lg.Data)
		if err != nil {
			log.Printf("nft platform rpc indexer: BaseURIUpdated unpack: %v", err)
			return nil
		}
		if err := tx.Model(&models.NFTCollection{}).Where("id = ?", collectionID).Update("base_uri", s).Error; err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"newBaseURI": s})
		return n.insertCollectionEvent(tx, collectionID, "BaseURIUpdated", bn, blockTime, txHash, logIndex, payload)

	case topicDefaultRoyaltyUpd:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		recv := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		fee := new(big.Int).SetBytes(lg.Data[:32]).String()
		payload, _ := json.Marshal(map[string]any{"receiver": strings.ToLower(recv), "feeNumerator": fee})
		return n.insertCollectionEvent(tx, collectionID, "DefaultRoyaltyUpdated", bn, blockTime, txHash, logIndex, payload)

	case topicFactoryOwnershipTr:
		if len(lg.Topics) < 3 {
			return nil
		}
		prev := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		newO := common.BytesToAddress(lg.Topics[2].Bytes()[12:]).Hex()
		payload, _ := json.Marshal(map[string]any{"previousOwner": strings.ToLower(prev), "newOwner": strings.ToLower(newO)})
		return n.insertCollectionEvent(tx, collectionID, "OwnershipTransferred", bn, blockTime, txHash, logIndex, payload)

	case topicInitialized:
		if len(lg.Data) < 32 {
			return nil
		}
		ver := new(big.Int).SetBytes(lg.Data[:32]).Uint64()
		payload, _ := json.Marshal(map[string]any{"version": ver})
		return n.insertCollectionEvent(tx, collectionID, "Initialized", bn, blockTime, txHash, logIndex, payload)

	case topicApproval:
		if len(lg.Topics) < 4 {
			return nil
		}
		o := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		a := common.BytesToAddress(lg.Topics[2].Bytes()[12:]).Hex()
		tid := new(big.Int).SetBytes(lg.Topics[3].Bytes()).String()
		payload, _ := json.Marshal(map[string]any{"owner": strings.ToLower(o), "approved": strings.ToLower(a), "tokenId": tid})
		return n.insertCollectionEvent(tx, collectionID, "Approval", bn, blockTime, txHash, logIndex, payload)

	case topicApprovalForAll:
		if len(lg.Topics) < 3 || len(lg.Data) < 32 {
			return nil
		}
		o := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
		op := common.BytesToAddress(lg.Topics[2].Bytes()[12:]).Hex()
		ap := lg.Data[31] == 1
		payload, _ := json.Marshal(map[string]any{"owner": strings.ToLower(o), "operator": strings.ToLower(op), "approved": ap})
		return n.insertCollectionEvent(tx, collectionID, "ApprovalForAll", bn, blockTime, txHash, logIndex, payload)

	default:
		return nil
	}
}

func (n *NFTPlatformRPC) insertCollectionEvent(tx *gorm.DB, collectionID uint64, typ string, bn int64, blockTime time.Time, txHash string, logIndex int, payload []byte) error {
	ev := models.NFTCollectionEvent{
		ChainID:      n.ChainID64,
		CollectionID: collectionID,
		EventType:    typ,
		BlockNumber:  bn,
		BlockTime:    blockTime,
		TxHash:       txHash,
		LogIndex:     logIndex,
		PayloadJSON:  payload,
		CreatedAt:    time.Now().UTC(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chain_id"}, {Name: "tx_hash"}, {Name: "log_index"}},
		DoNothing: true,
	}).Create(&ev).Error
}

func (n *NFTPlatformRPC) lookupCollectionID(tx *gorm.DB, contractLower string) (uint64, bool) {
	var id uint64
	err := tx.Table("nft_collections AS col").
		Joins("JOIN nft_contracts c ON c.id = col.contract_id").
		Where("col.chain_id = ? AND lower(c.address) = ?", n.ChainID64, contractLower).
		Select("col.id").
		Scan(&id).Error
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func (n *NFTPlatformRPC) ensureAccount(tx *gorm.DB, addr common.Address) (uint64, error) {
	hex := strings.ToLower(addr.Hex())
	now := time.Now().UTC()
	row := models.NFTAccount{
		ChainID:   n.ChainID64,
		Address:   hex,
		CreatedAt: now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chain_id"}, {Name: "address"}},
		DoNothing: true,
	}).Create(&row).Error; err != nil {
		return 0, err
	}
	var out models.NFTAccount
	if err := tx.Where("chain_id = ? AND address = ?", n.ChainID64, hex).First(&out).Error; err != nil {
		return 0, err
	}
	return out.ID, nil
}

func isPostgresDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}

func (n *NFTPlatformRPC) decodeAddressFromData(data []byte) (string, error) {
	if len(data) < 32 {
		return "", errors.New("short data")
	}
	return strings.ToLower(common.BytesToAddress(data[:32]).Hex()), nil
}

func isZeroBytes32(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func unpackABIString(data []byte) (string, error) {
	t, err := abi.NewType("string", "", nil)
	if err != nil {
		return "", err
	}
	args := abi.Arguments{{Type: t}}
	unpacked, err := args.Unpack(data)
	if err != nil {
		return "", err
	}
	if len(unpacked) == 0 {
		return "", errors.New("empty unpack")
	}
	s, ok := unpacked[0].(string)
	if !ok {
		return "", fmt.Errorf("want string got %T", unpacked[0])
	}
	return s, nil
}
