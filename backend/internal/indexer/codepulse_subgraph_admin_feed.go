package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"go-chain/backend/internal/subgraph"
)

// NormalizedSubgraphAdminEvent 子图管理端事件流水单条（与 cp_event_log 字段对齐用途，不入库）。
type NormalizedSubgraphAdminEvent struct {
	EventName       string
	BlockNumber     uint64
	LogIndex        int
	TxHash          string
	BlockTS         time.Time
	ProposalID      *uint64
	CampaignID      *uint64
	WalletAddress   *string
	Payload         json.RawMessage
}

// QueryCodePulseSubgraphAdminFeed 拉取各类事件最近 firstPerType 条（按块降序），合并后按块号、log_index、tx 全局排序。
// 用于管理端事件列表在「子图同步写库未开」时仍可直接读链上索引视图；与增量 syncBatch 解析逻辑一致。
func QueryCodePulseSubgraphAdminFeed(ctx context.Context, sg *subgraph.Client, firstPerType int) ([]NormalizedSubgraphAdminEvent, error) {
	if sg == nil || !sg.Configured() {
		return nil, fmt.Errorf("code-pulse subgraph not configured")
	}
	if firstPerType < 1 {
		firstPerType = 150
	}
	if firstPerType > 500 {
		firstPerType = 500
	}
	raw, err := sg.Query(ctx, codePulseAdminFeedQuery, map[string]any{"first": firstPerType})
	if err != nil {
		return nil, err
	}
	var data syncBatch
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	norms, _ := normalizedEventsFromSyncBatch(&data)
	sort.Slice(norms, func(i, j int) bool {
		if norms[i].Block != norms[j].Block {
			return norms[i].Block > norms[j].Block
		}
		if norms[i].LogIndex != norms[j].LogIndex {
			return norms[i].LogIndex > norms[j].LogIndex
		}
		return norms[i].TxHash > norms[j].TxHash
	})
	out := make([]NormalizedSubgraphAdminEvent, 0, len(norms))
	for _, ev := range norms {
		pid, cid, wal := inferProposalCampaignWallet(ev)
		out = append(out, NormalizedSubgraphAdminEvent{
			EventName:     ev.Name,
			BlockNumber:   ev.Block,
			LogIndex:      ev.LogIndex,
			TxHash:        ev.TxHash,
			BlockTS:       ev.TS,
			ProposalID:    pid,
			CampaignID:    cid,
			WalletAddress: wal,
			Payload:       ev.Raw,
		})
	}
	return out, nil
}
