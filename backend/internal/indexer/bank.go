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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	depositedTopic = crypto.Keccak256Hash([]byte("Deposited(address,address,uint256)"))
	withdrawnTopic = crypto.Keccak256Hash([]byte("Withdrawn(address,address,uint256)"))
)

const (
	// fallbackConfirmations：节点不支持 finalized/safe 标签（部分 L2、旧节点）时，用 latest 减该值作为扫描上界。
	fallbackConfirmations = uint64(12)
	// defaultPoll：略拉长可降低与同一 RPC Key 上其他客户端的总请求率（Infura 等易 429）。
	defaultPoll = 20 * time.Second
	lookbackBlocks = uint64(2000)
	// maxFilterLogBlockSpan：单次 eth_getLogs 的区块跨度上限（含端点），避免 Infura 等对大范围查询限流或报错。
	maxFilterLogBlockSpan = uint64(1000)
	// filterChunkPause：连续多次 getLogs 之间的间隔，减轻突发流量导致的 429。
	filterChunkPause = 250 * time.Millisecond
	rpcRetryMax      = 10
	rpcRetryInitial  = 400 * time.Millisecond
	rpcRetryMaxWait  = 45 * time.Second
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

func isRateLimitedRPC(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "429") ||
		strings.Contains(s, "too many requests") ||
		strings.Contains(s, "-32005") ||
		strings.Contains(s, "rate limit") ||
		strings.Contains(s, "exceeded")
}

// withRPCRetry 在 Infura 等返回 429 / -32005 时做指数退避重试；ctx 取消时立即结束。
func withRPCRetry(ctx context.Context, op func() error) error {
	wait := rpcRetryInitial
	var last error
	for attempt := 0; attempt < rpcRetryMax; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = op()
		if last == nil {
			return nil
		}
		if !isRateLimitedRPC(last) {
			return last
		}
		if attempt == rpcRetryMax-1 {
			break
		}
		log.Printf("bank indexer: RPC rate limited, backing off %v (%v)", wait, last)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		next := wait * 2
		if next > rpcRetryMaxWait {
			next = rpcRetryMaxWait
		}
		wait = next
	}
	return last
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

	head, err := b.headerByNumber(ctx, nil)
	if err != nil {
		log.Printf("bank indexer: 拉取链头失败，无法初始化/更新游标（请检查 ETH_RPC_URL）: %v", err)
		return err
	}
	safe, err := b.confirmedTip(ctx, head)
	if err != nil {
		return err
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

	logs, err := b.filterBankLogsChunked(ctx, from, to)
	if err != nil {
		return err
	}

	blockTimeCache := make(map[uint64]time.Time)
	getTime := func(num uint64) (time.Time, error) {
		if t0, ok := blockTimeCache[num]; ok {
			return t0, nil
		}
		hdr, err := b.headerByNumber(ctx, new(big.Int).SetUint64(num))
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

// filterBankLogsChunked 按 maxFilterLogBlockSpan 分段调用 FilterLogs，合并结果（按区块顺序递增）。
func (b *Bank) filterBankLogsChunked(ctx context.Context, from, to uint64) ([]types.Log, error) {
	base := ethereum.FilterQuery{
		Addresses: []common.Address{b.BankAddr},
		Topics: [][]common.Hash{
			{depositedTopic, withdrawnTopic},
		},
	}
	var out []types.Log
	for chunkFrom := from; chunkFrom <= to; {
		chunkTo := chunkFrom + maxFilterLogBlockSpan - 1
		if chunkTo > to {
			chunkTo = to
		}
		q := base
		q.FromBlock = new(big.Int).SetUint64(chunkFrom)
		q.ToBlock = new(big.Int).SetUint64(chunkTo)
		var chunk []types.Log
		err := withRPCRetry(ctx, func() error {
			var e error
			chunk, e = b.Eth.FilterLogs(ctx, q)
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
			case <-time.After(filterChunkPause):
			}
		}
	}
	return out, nil
}

func (b *Bank) headerByNumber(ctx context.Context, num *big.Int) (*types.Header, error) {
	var h *types.Header
	err := withRPCRetry(ctx, func() error {
		var e error
		h, e = b.Eth.HeaderByNumber(ctx, num)
		return e
	})
	return h, err
}

// confirmedTip 返回索引可安全扫到的最高区块号：优先 PoS 的 finalized，其次 safe，最后回退为 latest - fallbackConfirmations。
func (b *Bank) confirmedTip(ctx context.Context, latest *types.Header) (uint64, error) {
	if latest == nil {
		return 0, errors.New("bank indexer: nil latest header")
	}
	latestNum := latest.Number.Uint64()

	try := func(tag int64) (uint64, bool) {
		h, err := b.headerByNumber(ctx, big.NewInt(tag))
		if err != nil || h == nil {
			return 0, false
		}
		n := h.Number.Uint64()
		if n > latestNum {
			return 0, false
		}
		return n, true
	}

	if n, ok := try(int64(rpc.FinalizedBlockNumber)); ok {
		return n, nil
	}
	if n, ok := try(int64(rpc.SafeBlockNumber)); ok {
		return n, nil
	}

	if latestNum > fallbackConfirmations {
		return latestNum - fallbackConfirmations, nil
	}
	return 0, nil
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
		Name:             b.cursorName,
		LastScannedBlock: last,
		UpdatedAt:        time.Now().UTC(),
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
