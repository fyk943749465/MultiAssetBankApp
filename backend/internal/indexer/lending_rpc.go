package indexer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
)

// Base Sepolia 默认部署（与 migrations/007_lending.sql 种子一致）；当 lending_contracts 无行时作回退。
var defaultLendingWatch84532 = []struct {
	Addr string
	Kind string
	Label string
}{
	{"0x3f0248e6ff7e414485a146c18d6b72dc9e317e5f", "lending_pool", "Pool (Base Sepolia)"},
	{"0xe72ac9c1d557d65094ae92739e409ca56ae12b11", "hybrid_price_oracle", "HybridPriceOracle"},
	{"0x3100b1fd5a2180dac11820106579545d0f1c439b", "chainlink_price_oracle", "ChainlinkPriceOracle"},
	{"0x960e004f33566d0b56863f54532f1785923d2799", "reports_verifier", "ReportsVerifier"},
	{"0xb44d1c69eaf762441d6762e094b18d2614cf1617", "interest_rate_strategy_factory", "InterestRateStrategyFactory"},
	{"0x0f4c88d757e370016b5cfc1ac48d013378be4a27", "interest_rate_strategy", "InterestRateStrategy"},
}

// LendingRPC 在 **借贷专用 JSON-RPC**（Base Sepolia 等）上扫 lending_contracts 登记的合约日志，
// 与 ETH_RPC_URL（Sepolia 上 Bank/Code Pulse/NFT）完全隔离。
type LendingRPC struct {
	mu         sync.Mutex
	DB         *gorm.DB
	Eth        *ethclient.Client
	ChainID    int64
	StartBlock uint64
	cursorName string
}

// NewLendingRPC 创建索引器；eth 须为 LENDING_ETH_RPC_URL 对应链的客户端；chainID 用于 PG 列与游标名（须与节点 ChainID 一致）。
func NewLendingRPC(db *gorm.DB, eth *ethclient.Client, chainID int64, startBlock uint64) *LendingRPC {
	return &LendingRPC{
		DB:         db,
		Eth:        eth,
		ChainID:    chainID,
		StartBlock: startBlock,
		cursorName: fmt.Sprintf("lending_bundle_rpc_%d", chainID),
	}
}

func (l *LendingRPC) Run(ctx context.Context) {
	t := time.NewTicker(PollInterval())
	defer t.Stop()
	for {
		if err := l.SyncOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("lending rpc indexer: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (l *LendingRPC) SyncOnce(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	head, err := ethHeaderByNumber(ctx, l.Eth, nil)
	if err != nil {
		log.Printf("lending rpc indexer: 拉取链头失败（请检查 LENDING_ETH_RPC_URL / BASE_ETH_RPC_URL）: %v", err)
		return err
	}
	safe, err := ethConfirmedTip(ctx, l.Eth, head)
	if err != nil {
		return err
	}
	last, err := l.getOrInitCursor(ctx, safe)
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

	addrs, kindByAddr, poolAddr, err := l.loadWatchAddresses(ctx)
	if err != nil {
		return err
	}
	if len(addrs) == 0 {
		return errors.New("lending rpc indexer: no contract addresses to watch")
	}

	logs, err := l.filterLendingLogsChunked(ctx, addrs, from, to)
	if err != nil {
		return err
	}

	blockTimeCache := make(map[uint64]time.Time)
	getTime := func(num uint64) (time.Time, error) {
		if t0, ok := blockTimeCache[num]; ok {
			return t0, nil
		}
		hdr, err := ethHeaderByNumber(ctx, l.Eth, new(big.Int).SetUint64(num))
		if err != nil {
			return time.Time{}, err
		}
		ts := time.Unix(int64(hdr.Time), 0).UTC()
		blockTimeCache[num] = ts
		return ts, nil
	}

	return l.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, lg := range logs {
			ts, err := getTime(lg.BlockNumber)
			if err != nil {
				return err
			}
			if err := ingestLendingLog(tx, l.ChainID, poolAddr, kindByAddr, lg, ts); err != nil {
				return err
			}
		}
		return tx.Model(&models.ChainIndexerCursor{}).
			Where("name = ?", l.cursorName).
			Updates(map[string]any{
				"last_scanned_block": to,
				"updated_at":         time.Now().UTC(),
			}).Error
	})
}

func (l *LendingRPC) filterLendingLogsChunked(ctx context.Context, addrs []common.Address, from, to uint64) ([]types.Log, error) {
	topics := allLendingTopics()
	base := ethereum.FilterQuery{
		Addresses: addrs,
		Topics:    [][]common.Hash{topics},
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
			chunk, e = l.Eth.FilterLogs(ctx, q)
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

func (l *LendingRPC) getOrInitCursor(ctx context.Context, safe uint64) (uint64, error) {
	var cur models.ChainIndexerCursor
	err := l.DB.WithContext(ctx).Where("name = ?", l.cursorName).First(&cur).Error
	if err == nil {
		return cur.LastScannedBlock, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	start := l.StartBlock
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
		Name:             l.cursorName,
		LastScannedBlock: last,
		UpdatedAt:        time.Now().UTC(),
	}
	if err := l.DB.WithContext(ctx).Create(&cur).Error; err != nil {
		return 0, err
	}
	return last, nil
}

// loadWatchAddresses 合并 lending_contracts 与内嵌默认（84532）；返回监听地址、地址→kind、Pool 地址。
func (l *LendingRPC) loadWatchAddresses(ctx context.Context) ([]common.Address, map[common.Address]string, common.Address, error) {
	kindBy := make(map[common.Address]string)
	var pool common.Address

	var rows []models.LendingContract
	_ = l.DB.WithContext(ctx).Where("chain_id = ?", l.ChainID).Find(&rows).Error
	for _, r := range rows {
		if !common.IsHexAddress(r.Address) {
			continue
		}
		a := common.HexToAddress(r.Address)
		kindBy[a] = r.ContractKind
		if r.ContractKind == "lending_pool" {
			pool = a
		}
	}
	if len(kindBy) == 0 && l.ChainID == 84532 {
		for _, d := range defaultLendingWatch84532 {
			a := common.HexToAddress(d.Addr)
			kindBy[a] = d.Kind
			if d.Kind == "lending_pool" {
				pool = a
			}
		}
		log.Printf("lending rpc indexer: lending_contracts 为空，使用内嵌 Base Sepolia 默认 %d 个地址", len(kindBy))
	}

	addrs := make([]common.Address, 0, len(kindBy))
	for a := range kindBy {
		addrs = append(addrs, a)
	}
	return addrs, kindBy, pool, nil
}

func addrHex(a common.Address) string {
	return strings.ToLower(a.Hex())
}
