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

	"go-chain/backend/internal/contracts"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
)

// CodePulseRPC 与 Bank 索引器相同：仅依赖 RPC eth_getLogs + 已确认区块游标，将 Code Pulse 事件写入 PostgreSQL（权威读模型）。
type CodePulseRPC struct {
	mu         sync.Mutex
	DB         *gorm.DB
	Eth        *ethclient.Client
	Contract   common.Address
	ChainID    uint64
	StartBlock uint64
	cpABI      abi.ABI
	cursorName string
}

// NewCodePulseRPC 构造扫块索引器。StartBlock 为 0 时首次运行与 Bank 相同：从 safe 头往前 lookbackBlocks 起扫。
func NewCodePulseRPC(db *gorm.DB, eth *ethclient.Client, contract common.Address, chainID uint64, startBlock uint64) (*CodePulseRPC, error) {
	parsed, err := contracts.LoadCodePulseABI()
	if err != nil {
		return nil, err
	}
	return &CodePulseRPC{
		DB:         db,
		Eth:        eth,
		Contract:   contract,
		ChainID:    chainID,
		StartBlock: startBlock,
		cpABI:      parsed,
		cursorName: fmt.Sprintf("code_pulse_rpc_%d_%s", chainID, contract.Hex()),
	}, nil
}

// Run 定时扫块（阻塞，请在 goroutine 中调用）。
func (c *CodePulseRPC) Run(ctx context.Context) {
	t := time.NewTicker(PollInterval())
	defer t.Stop()
	for {
		if err := c.SyncOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("code-pulse rpc indexer: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// SyncOnce 处理一段已确认区块内的合约日志。
func (c *CodePulseRPC) SyncOnce(ctx context.Context) error {
	if c == nil || c.DB == nil || c.Eth == nil {
		return errors.New("code-pulse rpc indexer: not configured")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	head, err := ethHeaderByNumber(ctx, c.Eth, nil)
	if err != nil {
		log.Printf("code-pulse rpc indexer: 拉取链头失败: %v", err)
		return err
	}
	safe, err := ethConfirmedTip(ctx, c.Eth, head)
	if err != nil {
		return err
	}
	last, err := c.getOrInitCursor(ctx, safe)
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

	logs, err := c.filterCodePulseLogsChunked(ctx, from, to)
	if err != nil {
		return err
	}
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
		hdr, err := ethHeaderByNumber(ctx, c.Eth, new(big.Int).SetUint64(num))
		if err != nil {
			return time.Time{}, err
		}
		ts := time.Unix(int64(hdr.Time), 0).UTC()
		blockTimeCache[num] = ts
		return ts, nil
	}

	contractLower := strings.ToLower(c.Contract.Hex())

	return c.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, lg := range logs {
			ts, err := getTime(lg.BlockNumber)
			if err != nil {
				return err
			}
			ev, err := c.logToNormalized(lg, ts)
			if err != nil {
				log.Printf("code-pulse rpc indexer: skip log %s #%d: %v", lg.TxHash.Hex(), lg.Index, err)
				continue
			}
			if err := applyCodePulseEventTx(tx, c.ChainID, contractLower, ev); err != nil {
				return err
			}
		}
		return tx.Model(&models.ChainIndexerCursor{}).
			Where("name = ?", c.cursorName).
			Updates(map[string]any{
				"last_scanned_block": to,
				"updated_at":         time.Now().UTC(),
			}).Error
	})
}

func (c *CodePulseRPC) filterCodePulseLogsChunked(ctx context.Context, from, to uint64) ([]types.Log, error) {
	base := ethereum.FilterQuery{
		Addresses: []common.Address{c.Contract},
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
			chunk, e = c.Eth.FilterLogs(ctx, q)
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

func (c *CodePulseRPC) getOrInitCursor(ctx context.Context, safe uint64) (uint64, error) {
	var cur models.ChainIndexerCursor
	err := c.DB.WithContext(ctx).Where("name = ?", c.cursorName).First(&cur).Error
	if err == nil {
		return cur.LastScannedBlock, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	start := c.StartBlock
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
		Name:             c.cursorName,
		LastScannedBlock: last,
		UpdatedAt:        time.Now().UTC(),
	}
	if err := c.DB.WithContext(ctx).Create(&cur).Error; err != nil {
		return 0, err
	}
	return last, nil
}

// go-ethereum v1.17 的 abi.ABI 无 UnpackLog；用 ParseTopicsIntoMap + UnpackIntoMap 解码事件（与 bind.BoundContract.UnpackLog 等价思路）。
func cpIndexedArgs(args abi.Arguments) abi.Arguments {
	var out abi.Arguments
	for _, a := range args {
		if a.Indexed {
			out = append(out, a)
		}
	}
	return out
}

func unpackCodePulseLog(ev *abi.Event, lg types.Log) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	idx := cpIndexedArgs(ev.Inputs)
	if len(idx) != len(lg.Topics)-1 {
		return nil, fmt.Errorf("event %s: indexed args %d vs topics-1 %d", ev.Name, len(idx), len(lg.Topics)-1)
	}
	if len(idx) > 0 {
		if err := abi.ParseTopicsIntoMap(out, idx, lg.Topics[1:]); err != nil {
			return nil, err
		}
	}
	if err := ev.Inputs.UnpackIntoMap(out, lg.Data); err != nil {
		return nil, err
	}
	return out, nil
}

func rpcBaseJSON(blk uint64, ts time.Time, txh string) map[string]interface{} {
	return map[string]interface{}{
		"blockNumber":     fmt.Sprintf("%d", blk),
		"blockTimestamp":  fmt.Sprintf("%d", ts.Unix()),
		"transactionHash": txh,
	}
}

func rpcMerge(base map[string]interface{}, fields map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base)+len(fields))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range fields {
		out[k] = v
	}
	return out
}

func rpcBigStr(v interface{}) string {
	switch t := v.(type) {
	case *big.Int:
		if t == nil {
			return "0"
		}
		return t.String()
	case nil:
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func rpcAddrStr(v interface{}) string {
	switch t := v.(type) {
	case common.Address:
		return t.Hex()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (c *CodePulseRPC) logToNormalized(lg types.Log, blockTime time.Time) (normalizedEvent, error) {
	if len(lg.Topics) == 0 {
		return normalizedEvent{}, fmt.Errorf("empty topics")
	}
	evABI, err := c.cpABI.EventByID(lg.Topics[0])
	if err != nil {
		return normalizedEvent{}, err
	}
	m, err := unpackCodePulseLog(evABI, lg)
	if err != nil {
		return normalizedEvent{}, err
	}
	ts := blockTime
	txh := lg.TxHash.Hex()
	li := int(lg.Index)
	blk := lg.BlockNumber
	base := rpcBaseJSON(blk, ts, txh)

	switch evABI.Name {
	case "ProposalSubmitted":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"proposalId": rpcBigStr(m["proposalId"]),
			"organizer":  rpcAddrStr(m["organizer"]),
			"githubUrl":  m["githubUrl"],
			"target":     rpcBigStr(m["target"]),
			"duration":   rpcBigStr(m["duration"]),
		}))
		return normalizedEvent{Name: "ProposalSubmitted", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyProposalSubmitted, IndexerSource: "rpc"}, nil

	case "ProposalReviewed":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"proposalId": rpcBigStr(m["proposalId"]),
			"approved":   m["approved"],
		}))
		return normalizedEvent{Name: "ProposalReviewed", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyProposalReviewed, IndexerSource: "rpc"}, nil

	case "FundingRoundSubmittedForReview":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"proposalId":   rpcBigStr(m["proposalId"]),
			"roundOrdinal": rpcBigStr(m["roundOrdinal"]),
		}))
		return normalizedEvent{Name: "FundingRoundSubmittedForReview", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyFundingRoundSubmitted, IndexerSource: "rpc"}, nil

	case "FundingRoundReviewed":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"proposalId": rpcBigStr(m["proposalId"]),
			"approved":   m["approved"],
		}))
		return normalizedEvent{Name: "FundingRoundReviewed", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyFundingRoundReviewed, IndexerSource: "rpc"}, nil

	case "CrowdfundingLaunched":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"proposalId": rpcBigStr(m["proposalId"]),
			"campaignId": rpcBigStr(m["campaignId"]),
			"organizer":  rpcAddrStr(m["organizer"]),
			"githubUrl":  m["githubUrl"],
			"target":     rpcBigStr(m["target"]),
			"deadline":   rpcBigStr(m["deadline"]),
			"roundIndex": rpcBigStr(m["roundIndex"]),
		}))
		return normalizedEvent{Name: "CrowdfundingLaunched", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyCrowdfundingLaunched, IndexerSource: "rpc"}, nil

	case "Donated":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId":  rpcBigStr(m["campaignId"]),
			"contributor": rpcAddrStr(m["contributor"]),
			"amount":      rpcBigStr(m["amount"]),
		}))
		return normalizedEvent{Name: "Donated", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyDonated, IndexerSource: "rpc"}, nil

	case "CampaignFinalized":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId": rpcBigStr(m["campaignId"]),
			"successful": m["successful"],
		}))
		return normalizedEvent{Name: "CampaignFinalized", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyCampaignFinalized, IndexerSource: "rpc"}, nil

	case "RefundClaimed":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId":  rpcBigStr(m["campaignId"]),
			"contributor": rpcAddrStr(m["contributor"]),
			"amount":      rpcBigStr(m["amount"]),
		}))
		return normalizedEvent{Name: "RefundClaimed", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyRefundClaimed, IndexerSource: "rpc"}, nil

	case "DeveloperAdded":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId": rpcBigStr(m["campaignId"]),
			"developer":  rpcAddrStr(m["developer"]),
		}))
		return normalizedEvent{Name: "DeveloperAdded", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyDeveloperAdded, IndexerSource: "rpc"}, nil

	case "DeveloperRemoved":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId": rpcBigStr(m["campaignId"]),
			"developer":  rpcAddrStr(m["developer"]),
		}))
		return normalizedEvent{Name: "DeveloperRemoved", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyDeveloperRemoved, IndexerSource: "rpc"}, nil

	case "MilestoneApproved":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId":     rpcBigStr(m["campaignId"]),
			"milestoneIndex": rpcBigStr(m["milestoneIndex"]),
		}))
		return normalizedEvent{Name: "MilestoneApproved", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyMilestoneApproved, IndexerSource: "rpc"}, nil

	case "MilestoneShareClaimed":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId":     rpcBigStr(m["campaignId"]),
			"milestoneIndex": rpcBigStr(m["milestoneIndex"]),
			"developer":      rpcAddrStr(m["developer"]),
			"amount":         rpcBigStr(m["amount"]),
		}))
		return normalizedEvent{Name: "MilestoneShareClaimed", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyMilestoneShareClaimed, IndexerSource: "rpc"}, nil

	case "PlatformDonated":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"donor":  rpcAddrStr(m["donor"]),
			"amount": rpcBigStr(m["amount"]),
		}))
		return normalizedEvent{Name: "PlatformDonated", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyPlatformDonated, IndexerSource: "rpc"}, nil

	case "PlatformFundsWithdrawn":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"to":     rpcAddrStr(m["to"]),
			"amount": rpcBigStr(m["amount"]),
		}))
		return normalizedEvent{Name: "PlatformFundsWithdrawn", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyPlatformFundsWithdrawn, IndexerSource: "rpc"}, nil

	case "OwnershipTransferred":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"previousOwner": rpcAddrStr(m["previousOwner"]),
			"newOwner":      rpcAddrStr(m["newOwner"]),
		}))
		return normalizedEvent{Name: "OwnershipTransferred", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyOwnershipTransferred, IndexerSource: "rpc"}, nil

	case "Paused":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"account": rpcAddrStr(m["account"]),
		}))
		return normalizedEvent{Name: "Paused", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyPaused, IndexerSource: "rpc"}, nil

	case "Unpaused":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"account": rpcAddrStr(m["account"]),
		}))
		return normalizedEvent{Name: "Unpaused", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyUnpaused, IndexerSource: "rpc"}, nil

	case "ProposalInitiatorUpdated":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"account": rpcAddrStr(m["account"]),
			"allowed": m["allowed"],
		}))
		return normalizedEvent{Name: "ProposalInitiatorUpdated", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyProposalInitiatorUpdated, IndexerSource: "rpc"}, nil

	case "StaleFundsSwept":
		raw, _ := json.Marshal(rpcMerge(base, map[string]interface{}{
			"campaignId": rpcBigStr(m["campaignId"]),
			"amount":       rpcBigStr(m["amount"]),
		}))
		return normalizedEvent{Name: "StaleFundsSwept", Block: blk, LogIndex: li, TxHash: txh, TS: ts, Raw: raw, Apply: applyStaleFundsSwept, IndexerSource: "rpc"}, nil

	default:
		return normalizedEvent{}, fmt.Errorf("unsupported event %q", evABI.Name)
	}
}
