package indexer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	depositedTopic = crypto.Keccak256Hash([]byte("Deposited(address,address,uint256)"))
	withdrawnTopic = crypto.Keccak256Hash([]byte("Withdrawn(address,address,uint256)"))
)

const (
	confirmations  = uint64(8)
	defaultPoll    = 15 * time.Second
	lookbackBlocks = uint64(2000)
)

// Bank 索引 MultiAssetBank 的 Deposited / Withdrawn 日志并写入 PostgreSQL。
type Bank struct {
	mu         sync.Mutex
	DB         *gorm.DB
	Eth        *ethclient.Client
	BankAddr   common.Address
	ChainID    uint64
	StartBlock uint64
	cursorName string
}

func NewBank(db *gorm.DB, eth *ethclient.Client, bank common.Address, chainID uint64, startBlock uint64) *Bank {
	return &Bank{
		DB:         db,
		Eth:        eth,
		BankAddr:   bank,
		ChainID:    chainID,
		StartBlock: startBlock,
		cursorName: fmt.Sprintf("multi_asset_bank_%d_%s", chainID, bank.Hex()),
	}
}

// Run 定时拉取新区块中的事件（阻塞，请在 goroutine 中调用）。
func (b *Bank) Run(ctx context.Context) {
	t := time.NewTicker(defaultPoll)
	defer t.Stop()
	for {
		if err := b.SyncOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("bank indexer: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// SyncOnce 处理一段已确认区块（可单独测试或手动触发）。
func (b *Bank) SyncOnce(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	head, err := b.Eth.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Printf("bank indexer: 拉取链头失败，无法初始化/更新游标（请检查 ETH_RPC_URL）: %v", err)
		return err
	}
	safe := head.Number.Uint64()
	if safe > confirmations {
		safe -= confirmations
	}
	last, err := b.getOrInitCursor(ctx, safe)
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

	q := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(to),
		Addresses: []common.Address{b.BankAddr},
		Topics: [][]common.Hash{
			{depositedTopic, withdrawnTopic},
		},
	}
	logs, err := b.Eth.FilterLogs(ctx, q)
	if err != nil {
		return err
	}

	blockTimeCache := make(map[uint64]time.Time)
	getTime := func(num uint64) (time.Time, error) {
		if t0, ok := blockTimeCache[num]; ok {
			return t0, nil
		}
		hdr, err := b.Eth.HeaderByNumber(ctx, new(big.Int).SetUint64(num))
		if err != nil {
			return time.Time{}, err
		}
		ts := time.Unix(int64(hdr.Time), 0).UTC()
		blockTimeCache[num] = ts
		return ts, nil
	}

	return b.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, lg := range logs {
			ts, err := getTime(lg.BlockNumber)
			if err != nil {
				return err
			}
			if err := b.ingestLog(tx, lg, ts); err != nil {
				return err
			}
		}
		return tx.Model(&models.ChainIndexerCursor{}).
			Where("name = ?", b.cursorName).
			Updates(map[string]any{
				"last_scanned_block": to,
				"updated_at":         time.Now().UTC(),
			}).Error
	})
}

func (b *Bank) getOrInitCursor(ctx context.Context, safe uint64) (uint64, error) {
	var cur models.ChainIndexerCursor
	err := b.DB.WithContext(ctx).Where("name = ?", b.cursorName).First(&cur).Error
	if err == nil {
		return cur.LastScannedBlock, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	start := b.StartBlock
	if start == 0 {
		if safe > lookbackBlocks {
			start = safe - lookbackBlocks
		} else {
			start = 1
		}
	}
	last := start - 1
	if safe > 0 && last >= safe {
		last = safe - 1
	}

	cur = models.ChainIndexerCursor{
		Name:               b.cursorName,
		LastScannedBlock:   last,
		UpdatedAt:          time.Now().UTC(),
	}
	if err := b.DB.WithContext(ctx).Create(&cur).Error; err != nil {
		return 0, err
	}
	return last, nil
}

func (b *Bank) ingestLog(tx *gorm.DB, lg types.Log, blockTime time.Time) error {
	if len(lg.Topics) != 3 || len(lg.Data) < 32 {
		return nil
	}
	topic0 := lg.Topics[0]
	token := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
	user := common.BytesToAddress(lg.Topics[2].Bytes()[12:])
	amount := new(big.Int).SetBytes(lg.Data[:32])

	txHash := lg.TxHash.Hex()
	depConflict := clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "tx_hash"},
			{Name: "log_index"},
		},
		DoNothing: true,
	}

	switch topic0 {
	case depositedTopic:
		row := models.BankDeposit{
			ChainID:      b.ChainID,
			TxHash:       txHash,
			LogIndex:     lg.Index,
			BlockNumber:  lg.BlockNumber,
			BlockTime:    blockTime,
			TokenAddress: token.Hex(),
			UserAddress:  user.Hex(),
			AmountRaw:    amount.String(),
			CreatedAt:    time.Now().UTC(),
		}
		return tx.Clauses(depConflict).Create(&row).Error
	case withdrawnTopic:
		row := models.BankWithdrawal{
			ChainID:      b.ChainID,
			TxHash:       txHash,
			LogIndex:     lg.Index,
			BlockNumber:  lg.BlockNumber,
			BlockTime:    blockTime,
			TokenAddress: token.Hex(),
			UserAddress:  user.Hex(),
			AmountRaw:    amount.String(),
			CreatedAt:    time.Now().UTC(),
		}
		wConf := clause.OnConflict{
			Columns: []clause.Column{
				{Name: "chain_id"},
				{Name: "tx_hash"},
				{Name: "log_index"},
			},
			DoNothing: true,
		}
		return tx.Clauses(wConf).Create(&row).Error
	default:
		return nil
	}
}
