package indexer

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"time"

	"go-chain/backend/internal/models"
	"go-chain/backend/internal/subgraph"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	codePulseSyncName     = "code_pulse_subgraph"
	defaultCPPoll         = 25 * time.Second
	defaultCPFetchFirst   = 150 // 每类实体上限；18 类同询，过大易超时
	proposalPendingReview = "pending_review"
	proposalApproved      = "approved"
	proposalRejected      = "rejected"
	proposalRoundPending  = "round_review_pending"
	proposalRoundApproved = "round_review_approved"
	proposalRoundRejected = "round_review_rejected"
	proposalSettled       = "settled"

	campaignFundraising      = "fundraising"
	campaignSuccessful       = "successful"
	campaignFailedRefundable = "failed_refundable"
)

// GraphQL：一次拉取子图中与 Code Pulse 读模型相关的实体（按块增量）。
const codePulseSyncQuery = `
query CodePulseSync($block: BigInt!, $first: Int!) {
  proposalSubmitteds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id proposalId organizer githubUrl target duration blockNumber blockTimestamp transactionHash
  }
  proposalRevieweds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id proposalId approved blockNumber blockTimestamp transactionHash
  }
  fundingRoundSubmittedForReviews(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id proposalId roundOrdinal blockNumber blockTimestamp transactionHash
  }
  fundingRoundRevieweds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id proposalId approved blockNumber blockTimestamp transactionHash
  }
  crowdfundingLauncheds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id proposalId campaignId organizer githubUrl target deadline roundIndex blockNumber blockTimestamp transactionHash
  }
  donateds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId contributor amount blockNumber blockTimestamp transactionHash
  }
  campaignFinalizeds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId successful blockNumber blockTimestamp transactionHash
  }
  refundClaimeds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId contributor amount blockNumber blockTimestamp transactionHash
  }
  developerAddeds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId developer blockNumber blockTimestamp transactionHash
  }
  developerRemoveds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId developer blockNumber blockTimestamp transactionHash
  }
  milestoneApproveds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId milestoneIndex blockNumber blockTimestamp transactionHash
  }
  milestoneShareClaimeds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId milestoneIndex developer amount blockNumber blockTimestamp transactionHash
  }
  platformDonateds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id donor amount blockNumber blockTimestamp transactionHash
  }
  platformFundsWithdrawns(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id to amount blockNumber blockTimestamp transactionHash
  }
  ownershipTransferreds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id previousOwner newOwner blockNumber blockTimestamp transactionHash
  }
  pauseds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id account blockNumber blockTimestamp transactionHash
  }
  unpauseds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id account blockNumber blockTimestamp transactionHash
  }
  proposalInitiatorUpdateds(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id account allowed blockNumber blockTimestamp transactionHash
  }
  staleFundsSwepts(first: $first, orderBy: blockNumber, orderDirection: asc, where: { blockNumber_gt: $block }) {
    id campaignId amount blockNumber blockTimestamp transactionHash
  }
}
`

// GraphQL：管理端事件流水（每类最近 N 条，按块号降序），无 block 游标，不入库。
const codePulseAdminFeedQuery = `
query CodePulseAdminFeed($first: Int!) {
  proposalSubmitteds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id proposalId organizer githubUrl target duration blockNumber blockTimestamp transactionHash
  }
  proposalRevieweds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id proposalId approved blockNumber blockTimestamp transactionHash
  }
  fundingRoundSubmittedForReviews(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id proposalId roundOrdinal blockNumber blockTimestamp transactionHash
  }
  fundingRoundRevieweds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id proposalId approved blockNumber blockTimestamp transactionHash
  }
  crowdfundingLauncheds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id proposalId campaignId organizer githubUrl target deadline roundIndex blockNumber blockTimestamp transactionHash
  }
  donateds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId contributor amount blockNumber blockTimestamp transactionHash
  }
  campaignFinalizeds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId successful blockNumber blockTimestamp transactionHash
  }
  refundClaimeds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId contributor amount blockNumber blockTimestamp transactionHash
  }
  developerAddeds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId developer blockNumber blockTimestamp transactionHash
  }
  developerRemoveds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId developer blockNumber blockTimestamp transactionHash
  }
  milestoneApproveds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId milestoneIndex blockNumber blockTimestamp transactionHash
  }
  milestoneShareClaimeds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId milestoneIndex developer amount blockNumber blockTimestamp transactionHash
  }
  platformDonateds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id donor amount blockNumber blockTimestamp transactionHash
  }
  platformFundsWithdrawns(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id to amount blockNumber blockTimestamp transactionHash
  }
  ownershipTransferreds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id previousOwner newOwner blockNumber blockTimestamp transactionHash
  }
  pauseds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id account blockNumber blockTimestamp transactionHash
  }
  unpauseds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id account blockNumber blockTimestamp transactionHash
  }
  proposalInitiatorUpdateds(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id account allowed blockNumber blockTimestamp transactionHash
  }
  staleFundsSwepts(first: $first, orderBy: blockNumber, orderDirection: desc) {
    id campaignId amount blockNumber blockTimestamp transactionHash
  }
}
`

// CodePulseSubgraph 将 Code Pulse 子图事件增量写入 PostgreSQL（读模型）。
type CodePulseSubgraph struct {
	DB           *gorm.DB
	SG           *subgraph.Client
	ChainID      uint64
	ContractAddr string // checksummed or lower, normalized on use
	StartBlock   uint64
	PollInterval time.Duration
	FetchFirst   int
}

// NewCodePulseSubgraph 构造子图同步器。PollInterval 为 0 时用 defaultCPPoll；FetchFirst 为 0 时用 defaultCPFetchFirst。
func NewCodePulseSubgraph(db *gorm.DB, sg *subgraph.Client, chainID uint64, contractHex string, startBlock uint64, poll time.Duration) *CodePulseSubgraph {
	if poll <= 0 {
		poll = defaultCPPoll
	}
	first := defaultCPFetchFirst
	addr := strings.TrimSpace(contractHex)
	if common.IsHexAddress(addr) {
		addr = common.HexToAddress(addr).Hex()
	}
	return &CodePulseSubgraph{
		DB:           db,
		SG:           sg,
		ChainID:      chainID,
		ContractAddr: addr,
		StartBlock:   startBlock,
		PollInterval: poll,
		FetchFirst:   first,
	}
}

// Run 常驻循环（请在 goroutine 中调用）。
func (x *CodePulseSubgraph) Run(ctx context.Context) {
	t := time.NewTicker(x.PollInterval)
	defer t.Stop()
	for {
		if err := x.SyncOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("code-pulse subgraph sync: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// SyncOnce 执行一轮增量同步（可单独测试或手动触发思路）。
func (x *CodePulseSubgraph) SyncOnce(ctx context.Context) error {
	if x == nil || x.DB == nil || x.SG == nil || !x.SG.Configured() {
		return fmt.Errorf("code-pulse subgraph sync: not configured")
	}
	contract := strings.ToLower(x.ContractAddr)

	fromBlock, err := x.loadCursor()
	if err != nil {
		return err
	}

	batch, maxBlk, err := x.fetchBatch(ctx, fromBlock)
	if err != nil {
		x.noteSubgraphFailure(err)
		return err
	}
	if err := x.noteSubgraphQueryOK(); err != nil {
		return err
	}
	if len(batch) == 0 {
		return nil
	}
	sort.Slice(batch, func(i, j int) bool {
		if batch[i].Block != batch[j].Block {
			return batch[i].Block < batch[j].Block
		}
		if batch[i].LogIndex != batch[j].LogIndex {
			return batch[i].LogIndex < batch[j].LogIndex
		}
		return batch[i].TxHash < batch[j].TxHash
	})
	var maxSeen uint64
	for _, ev := range batch {
		if err := x.applyEvent(ctx, contract, ev); err != nil {
			x.noteSubgraphFailure(err)
			return err
		}
		if ev.Block > maxSeen {
			maxSeen = ev.Block
		}
	}
	if maxBlk > maxSeen {
		maxSeen = maxBlk
	}
	if maxSeen > 0 {
		return x.saveCursor(maxSeen)
	}
	return nil
}

func (x *CodePulseSubgraph) loadCursor() (uint64, error) {
	var cur models.CPSyncCursor
	err := x.DB.Where("sync_name = ?", codePulseSyncName).First(&cur).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var start uint64
		if x.StartBlock > 0 {
			if x.StartBlock >= 1 {
				start = x.StartBlock - 1
			}
		}
		nc := models.CPSyncCursor{
			SyncName:        codePulseSyncName,
			LastBlockNumber: u64Ptr(start),
			UpdatedAt:       time.Now(),
		}
		if err := x.DB.Create(&nc).Error; err != nil {
			return 0, err
		}
		return start, nil
	}
	if err != nil {
		return 0, err
	}
	if cur.LastBlockNumber == nil {
		return 0, nil
	}
	return *cur.LastBlockNumber, nil
}

func (x *CodePulseSubgraph) saveCursor(block uint64) error {
	now := time.Now()
	return x.DB.Model(&models.CPSyncCursor{}).
		Where("sync_name = ?", codePulseSyncName).
		Updates(map[string]any{
			"last_block_number":    block,
			"last_block_timestamp": now,
			"updated_at":           now,
		}).Error
}

// noteSubgraphQueryOK 在子图 GraphQL 成功返回并解析后调用：刷新「上次成功拉取」时间并清零连续失败计数。
func (x *CodePulseSubgraph) noteSubgraphQueryOK() error {
	if x == nil || x.DB == nil {
		return nil
	}
	now := time.Now()
	return x.DB.Model(&models.CPSyncCursor{}).
		Where("sync_name = ?", codePulseSyncName).
		Updates(map[string]any{
			"last_subgraph_query_ok_at":   now,
			"subgraph_consecutive_errors": 0,
			"last_subgraph_error":         gorm.Expr("NULL"),
			"last_subgraph_error_at":      gorm.Expr("NULL"),
			"updated_at":                  now,
		}).Error
}

// noteSubgraphFailure 记录子图请求失败或入库失败；用于告警与 /admin/sync-status 观测。
func (x *CodePulseSubgraph) noteSubgraphFailure(err error) {
	if x == nil || x.DB == nil || err == nil {
		return
	}
	msg := err.Error()
	const maxLen = 4000
	if len(msg) > maxLen {
		msg = msg[:maxLen]
	}
	now := time.Now()
	_ = x.DB.Model(&models.CPSyncCursor{}).
		Where("sync_name = ?", codePulseSyncName).
		Updates(map[string]any{
			"last_subgraph_error":         msg,
			"last_subgraph_error_at":      now,
			"subgraph_consecutive_errors": gorm.Expr("subgraph_consecutive_errors + 1"),
			"updated_at":                  now,
		}).Error
}

type cpRawEvent struct {
	ID        string       `json:"id"`
	Block     jsonUint64   `json:"blockNumber"`
	Timestamp jsonUint64   `json:"blockTimestamp"`
	TxHash    jsonBytesHex `json:"transactionHash"`
}

type jsonUint64 uint64

func (j *jsonUint64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return fmt.Errorf("jsonUint64: %s", s)
	}
	if !n.IsUint64() {
		return fmt.Errorf("jsonUint64 overflow")
	}
	*j = jsonUint64(n.Uint64())
	return nil
}

type jsonBytesHex []byte

func (j *jsonBytesHex) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	dec, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return err
	}
	*j = dec
	return nil
}

func (j jsonBytesHex) Hex() string {
	if len(j) == 0 {
		return ""
	}
	return "0x" + hex.EncodeToString(j)
}

type syncBatch struct {
	ProposalSubmitteds              []psRow  `json:"proposalSubmitteds"`
	ProposalRevieweds               []prRow  `json:"proposalRevieweds"`
	FundingRoundSubmittedForReviews []frsRow `json:"fundingRoundSubmittedForReviews"`
	FundingRoundRevieweds           []frrRow `json:"fundingRoundRevieweds"`
	CrowdfundingLauncheds           []clRow  `json:"crowdfundingLauncheds"`
	Donateds                        []donRow `json:"donateds"`
	CampaignFinalizeds              []cfRow  `json:"campaignFinalizeds"`
	RefundClaimeds                  []rcRow  `json:"refundClaimeds"`
	DeveloperAddeds                 []daRow  `json:"developerAddeds"`
	DeveloperRemoveds               []drRow  `json:"developerRemoveds"`
	MilestoneApproveds              []maRow  `json:"milestoneApproveds"`
	MilestoneShareClaimeds          []mscRow `json:"milestoneShareClaimeds"`
	PlatformDonateds                []pdRow  `json:"platformDonateds"`
	PlatformFundsWithdrawns         []pfwRow `json:"platformFundsWithdrawns"`
	OwnershipTransferreds           []otRow  `json:"ownershipTransferreds"`
	Pauseds                         []paRow  `json:"pauseds"`
	Unpauseds                       []upRow  `json:"unpauseds"`
	ProposalInitiatorUpdateds       []piuRow `json:"proposalInitiatorUpdateds"`
	StaleFundsSwepts                []sfsRow `json:"staleFundsSwepts"`
}

type psRow struct {
	cpRawEvent
	ProposalID jsonUint64   `json:"proposalId"`
	Organizer  jsonBytesHex `json:"organizer"`
	GithubURL  string       `json:"githubUrl"`
	Target     string       `json:"target"`
	Duration   string       `json:"duration"`
}

type prRow struct {
	cpRawEvent
	ProposalID jsonUint64 `json:"proposalId"`
	Approved   bool       `json:"approved"`
}

type frsRow struct {
	cpRawEvent
	ProposalID   jsonUint64 `json:"proposalId"`
	RoundOrdinal string     `json:"roundOrdinal"`
}

type frrRow struct {
	cpRawEvent
	ProposalID jsonUint64 `json:"proposalId"`
	Approved   bool       `json:"approved"`
}

type clRow struct {
	cpRawEvent
	ProposalID jsonUint64   `json:"proposalId"`
	CampaignID jsonUint64   `json:"campaignId"`
	Organizer  jsonBytesHex `json:"organizer"`
	GithubURL  string       `json:"githubUrl"`
	Target     string       `json:"target"`
	Deadline   string       `json:"deadline"`
	RoundIndex string       `json:"roundIndex"`
}

type donRow struct {
	cpRawEvent
	CampaignID  jsonUint64   `json:"campaignId"`
	Contributor jsonBytesHex `json:"contributor"`
	Amount      string       `json:"amount"`
}

type cfRow struct {
	cpRawEvent
	CampaignID jsonUint64 `json:"campaignId"`
	Successful bool       `json:"successful"`
}

type rcRow struct {
	cpRawEvent
	CampaignID  jsonUint64   `json:"campaignId"`
	Contributor jsonBytesHex `json:"contributor"`
	Amount      string       `json:"amount"`
}

type daRow struct {
	cpRawEvent
	CampaignID jsonUint64   `json:"campaignId"`
	Developer  jsonBytesHex `json:"developer"`
}

type drRow struct {
	cpRawEvent
	CampaignID jsonUint64   `json:"campaignId"`
	Developer  jsonBytesHex `json:"developer"`
}

type maRow struct {
	cpRawEvent
	CampaignID     jsonUint64 `json:"campaignId"`
	MilestoneIndex string     `json:"milestoneIndex"`
}

type mscRow struct {
	cpRawEvent
	CampaignID     jsonUint64   `json:"campaignId"`
	MilestoneIndex string       `json:"milestoneIndex"`
	Developer      jsonBytesHex `json:"developer"`
	Amount         string       `json:"amount"`
}

type pdRow struct {
	cpRawEvent
	Donor  jsonBytesHex `json:"donor"`
	Amount string       `json:"amount"`
}

type pfwRow struct {
	cpRawEvent
	To     jsonBytesHex `json:"to"`
	Amount string       `json:"amount"`
}

type otRow struct {
	cpRawEvent
	PreviousOwner jsonBytesHex `json:"previousOwner"`
	NewOwner      jsonBytesHex `json:"newOwner"`
}

type paRow struct {
	cpRawEvent
	Account jsonBytesHex `json:"account"`
}

type upRow struct {
	cpRawEvent
	Account jsonBytesHex `json:"account"`
}

type piuRow struct {
	cpRawEvent
	Account jsonBytesHex `json:"account"`
	Allowed bool         `json:"allowed"`
}

type sfsRow struct {
	cpRawEvent
	CampaignID jsonUint64 `json:"campaignId"`
	Amount     string     `json:"amount"`
}

type normalizedEvent struct {
	Name     string
	Block    uint64
	LogIndex int
	TxHash   string
	TS       time.Time
	Raw      json.RawMessage
	Apply    func(*gorm.DB, string, normalizedEvent) error
	// IndexerSource 写入 cp_event_log.source：subgraph | rpc。空则按 subgraph 处理（兼容旧数据路径）。
	IndexerSource string
}

// normalizedEventsFromSyncBatch 将子图 syncBatch JSON 转为与增量同步相同的 normalizedEvent 列表（不写入 DB）。
func normalizedEventsFromSyncBatch(data *syncBatch) ([]normalizedEvent, uint64) {
	var out []normalizedEvent
	var maxB uint64
	appendEv := func(name string, id string, blk, ts jsonUint64, tx jsonBytesHex, raw json.RawMessage, fn func(*gorm.DB, string, normalizedEvent) error) {
		txH, li, err := subgraphEntityIDToTxLog(id, tx)
		if err != nil {
			log.Printf("code-pulse subgraph sync: skip %s bad id: %v", name, err)
			return
		}
		t := time.Unix(int64(ts), 0).UTC()
		ev := normalizedEvent{Name: name, Block: uint64(blk), LogIndex: li, TxHash: txH, TS: t, Raw: raw, Apply: fn, IndexerSource: "subgraph"}
		out = append(out, ev)
		if uint64(blk) > maxB {
			maxB = uint64(blk)
		}
	}

	for _, r := range data.ProposalSubmitteds {
		raw, _ := json.Marshal(r)
		appendEv("ProposalSubmitted", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyProposalSubmitted)
	}
	for _, r := range data.ProposalRevieweds {
		raw, _ := json.Marshal(r)
		appendEv("ProposalReviewed", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyProposalReviewed)
	}
	for _, r := range data.FundingRoundSubmittedForReviews {
		raw, _ := json.Marshal(r)
		appendEv("FundingRoundSubmittedForReview", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyFundingRoundSubmitted)
	}
	for _, r := range data.FundingRoundRevieweds {
		raw, _ := json.Marshal(r)
		appendEv("FundingRoundReviewed", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyFundingRoundReviewed)
	}
	for _, r := range data.CrowdfundingLauncheds {
		raw, _ := json.Marshal(r)
		appendEv("CrowdfundingLaunched", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyCrowdfundingLaunched)
	}
	for _, r := range data.Donateds {
		raw, _ := json.Marshal(r)
		appendEv("Donated", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyDonated)
	}
	for _, r := range data.CampaignFinalizeds {
		raw, _ := json.Marshal(r)
		appendEv("CampaignFinalized", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyCampaignFinalized)
	}
	for _, r := range data.RefundClaimeds {
		raw, _ := json.Marshal(r)
		appendEv("RefundClaimed", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyRefundClaimed)
	}
	for _, r := range data.DeveloperAddeds {
		raw, _ := json.Marshal(r)
		appendEv("DeveloperAdded", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyDeveloperAdded)
	}
	for _, r := range data.DeveloperRemoveds {
		raw, _ := json.Marshal(r)
		appendEv("DeveloperRemoved", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyDeveloperRemoved)
	}
	for _, r := range data.MilestoneApproveds {
		raw, _ := json.Marshal(r)
		appendEv("MilestoneApproved", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyMilestoneApproved)
	}
	for _, r := range data.MilestoneShareClaimeds {
		raw, _ := json.Marshal(r)
		appendEv("MilestoneShareClaimed", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyMilestoneShareClaimed)
	}
	for _, r := range data.PlatformDonateds {
		raw, _ := json.Marshal(r)
		appendEv("PlatformDonated", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyPlatformDonated)
	}
	for _, r := range data.PlatformFundsWithdrawns {
		raw, _ := json.Marshal(r)
		appendEv("PlatformFundsWithdrawn", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyPlatformFundsWithdrawn)
	}
	for _, r := range data.OwnershipTransferreds {
		raw, _ := json.Marshal(r)
		appendEv("OwnershipTransferred", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyOwnershipTransferred)
	}
	for _, r := range data.Pauseds {
		raw, _ := json.Marshal(r)
		appendEv("Paused", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyPaused)
	}
	for _, r := range data.Unpauseds {
		raw, _ := json.Marshal(r)
		appendEv("Unpaused", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyUnpaused)
	}
	for _, r := range data.ProposalInitiatorUpdateds {
		raw, _ := json.Marshal(r)
		appendEv("ProposalInitiatorUpdated", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyProposalInitiatorUpdated)
	}
	for _, r := range data.StaleFundsSwepts {
		raw, _ := json.Marshal(r)
		appendEv("StaleFundsSwept", r.ID, r.Block, r.Timestamp, r.TxHash, raw, applyStaleFundsSwept)
	}

	return out, maxB
}

func (x *CodePulseSubgraph) fetchBatch(ctx context.Context, fromBlock uint64) ([]normalizedEvent, uint64, error) {
	q := codePulseSyncQuery
	vars := map[string]any{"block": fmt.Sprintf("%d", fromBlock), "first": x.FetchFirst}
	raw, err := x.SG.Query(ctx, q, vars)
	if err != nil {
		return nil, 0, err
	}
	var data syncBatch
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, 0, err
	}
	out, maxB := normalizedEventsFromSyncBatch(&data)
	return out, maxB, nil
}

func subgraphEntityIDToTxLog(id string, txFallback jsonBytesHex) (txHash string, logIndex int, err error) {
	s := strings.TrimSpace(id)
	s = strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return "", 0, err
	}
	if len(b) < 36 {
		// 个别托管端可能只返回 tx hash，退化为 logIndex=0
		if len(txFallback) == 32 {
			return common.BytesToHash(txFallback).Hex(), 0, nil
		}
		return "", 0, fmt.Errorf("entity id too short: %d", len(b))
	}
	txB := b[:32]
	li := int(binary.BigEndian.Uint32(b[len(b)-4:]))
	return common.BytesToHash(txB).Hex(), li, nil
}

func inferProposalCampaignWallet(ev normalizedEvent) (proposalID *uint64, campaignID *uint64, wallet *string) {
	switch ev.Name {
	case "ProposalSubmitted":
		var r psRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		p := uint64(r.ProposalID)
		w := strings.ToLower(r.Organizer.Hex())
		return &p, nil, &w
	case "ProposalReviewed":
		var r prRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		p := uint64(r.ProposalID)
		return &p, nil, nil
	case "FundingRoundSubmittedForReview":
		var r frsRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		p := uint64(r.ProposalID)
		return &p, nil, nil
	case "FundingRoundReviewed":
		var r frrRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		p := uint64(r.ProposalID)
		return &p, nil, nil
	case "CrowdfundingLaunched":
		var r clRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		p := uint64(r.ProposalID)
		c := uint64(r.CampaignID)
		w := strings.ToLower(r.Organizer.Hex())
		return &p, &c, &w
	case "Donated", "RefundClaimed":
		var r donRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		c := uint64(r.CampaignID)
		w := strings.ToLower(r.Contributor.Hex())
		return nil, &c, &w
	case "CampaignFinalized":
		var r cfRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		c := uint64(r.CampaignID)
		return nil, &c, nil
	case "DeveloperAdded":
		var r daRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		c := uint64(r.CampaignID)
		w := strings.ToLower(r.Developer.Hex())
		return nil, &c, &w
	case "DeveloperRemoved":
		var r drRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		c := uint64(r.CampaignID)
		w := strings.ToLower(r.Developer.Hex())
		return nil, &c, &w
	case "MilestoneApproved":
		var r maRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		c := uint64(r.CampaignID)
		return nil, &c, nil
	case "StaleFundsSwept":
		var r sfsRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		c := uint64(r.CampaignID)
		return nil, &c, nil
	case "MilestoneShareClaimed":
		var r mscRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		c := uint64(r.CampaignID)
		w := strings.ToLower(r.Developer.Hex())
		return nil, &c, &w
	case "PlatformDonated":
		var r pdRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		w := strings.ToLower(r.Donor.Hex())
		return nil, nil, &w
	case "PlatformFundsWithdrawn":
		var r pfwRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		w := strings.ToLower(r.To.Hex())
		return nil, nil, &w
	case "ProposalInitiatorUpdated", "Paused", "Unpaused":
		var r piuRow
		if json.Unmarshal(ev.Raw, &r) != nil {
			return nil, nil, nil
		}
		w := strings.ToLower(r.Account.Hex())
		return nil, nil, &w
	default:
		return nil, nil, nil
	}
}

func applyProposalSubmitted(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r psRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	pid := uint64(r.ProposalID)
	org := strings.ToLower(r.Organizer.Hex())
	txh := ev.TxHash
	blk := ev.Block
	p := models.CPProposal{
		ProposalID:           pid,
		OrganizerAddress:     org,
		GithubURL:            r.GithubURL,
		TargetWei:            r.Target,
		DurationSeconds:      mustInt64String(r.Duration),
		Status:               proposalPendingReview,
		StatusCode:           1,
		SubmittedTxHash:      &txh,
		SubmittedBlockNumber: &blk,
		SubmittedAt:          &ev.TS,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "proposal_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"organizer_address", "github_url", "target_wei", "duration_seconds",
			"submitted_tx_hash", "submitted_block_number", "submitted_at", "updated_at",
		}),
	}).Create(&p).Error
}

func applyProposalReviewed(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r prRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	pid := uint64(r.ProposalID)
	updates := map[string]any{"reviewed_at": ev.TS, "updated_at": time.Now()}
	if r.Approved {
		updates["status"] = proposalApproved
		updates["status_code"] = 2
		updates["approved_at"] = ev.TS
		updates["rejected_at"] = nil
	} else {
		updates["status"] = proposalRejected
		updates["status_code"] = 3
		updates["rejected_at"] = ev.TS
		updates["approved_at"] = nil
		updates["round_review_state"] = nil
		updates["round_review_state_code"] = nil
	}
	return db.Model(&models.CPProposal{}).Where("proposal_id = ?", pid).Updates(updates).Error
}

func applyFundingRoundSubmitted(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r frsRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	pid := uint64(r.ProposalID)
	pending := proposalRoundPending
	code := 1
	return db.Model(&models.CPProposal{}).Where("proposal_id = ?", pid).Updates(map[string]any{
		"status":                  proposalApproved,
		"status_code":             2,
		"round_review_state":      pending,
		"round_review_state_code": code,
		"updated_at":              time.Now(),
	}).Error
}

func applyFundingRoundReviewed(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r frrRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	pid := uint64(r.ProposalID)
	updates := map[string]any{"updated_at": time.Now()}
	if r.Approved {
		updates["round_review_state"] = proposalRoundApproved
		updates["round_review_state_code"] = 2
	} else {
		updates["round_review_state"] = proposalRoundRejected
		updates["round_review_state_code"] = 3
		updates["status"] = proposalRoundRejected
		updates["status_code"] = 6
	}
	return db.Model(&models.CPProposal{}).Where("proposal_id = ?", pid).Updates(updates).Error
}

func applyCrowdfundingLaunched(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r clRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	pid := uint64(r.ProposalID)
	cid := uint64(r.CampaignID)
	org := strings.ToLower(r.Organizer.Hex())
	deadlineUnix := mustInt64String(r.Deadline)
	deadline := time.Unix(deadlineUnix, 0).UTC()
	ri := int(mustInt64String(r.RoundIndex))

	camp := models.CPCampaign{
		CampaignID:             cid,
		ProposalID:             pid,
		RoundIndex:             ri,
		OrganizerAddress:       org,
		GithubURL:              r.GithubURL,
		TargetWei:              r.Target,
		DeadlineAt:             deadline,
		AmountRaisedWei:        "0",
		TotalWithdrawnWei:      "0",
		UnclaimedRefundPoolWei: "0",
		State:                  campaignFundraising,
		StateCode:              1,
		LaunchedTxHash:         ev.TxHash,
		LaunchedBlockNumber:    ev.Block,
		LaunchedAt:             ev.TS,
	}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "campaign_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"proposal_id", "round_index", "organizer_address", "github_url", "target_wei",
			"deadline_at", "launched_tx_hash", "launched_block_number", "launched_at", "updated_at",
		}),
	}).Create(&camp).Error; err != nil {
		return err
	}

	return db.Model(&models.CPProposal{}).Where("proposal_id = ?", pid).Updates(map[string]any{
		"last_campaign_id":               cid,
		"current_round_count":            gorm.Expr("current_round_count + ?", 1),
		"round_review_state":             nil,
		"round_review_state_code":        nil,
		"pending_round_target_wei":       nil,
		"pending_round_duration_seconds": nil,
		"status":                         proposalApproved,
		"status_code":                    2,
		"updated_at":                     time.Now(),
	}).Error
}

func applyDonated(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r donRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	contrib := strings.ToLower(r.Contributor.Hex())
	amt := r.Amount

	var existing models.CPContribution
	err := db.Where("campaign_id = ? AND LOWER(contributor_address) = ?", cid, contrib).First(&existing).Error
	isNew := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !isNew {
		return err
	}
	if isNew {
		co := models.CPContribution{
			CampaignID:          cid,
			ContributorAddress:  contrib,
			TotalContributedWei: amt,
			LastDonatedAt:       &ev.TS,
		}
		if err := db.Create(&co).Error; err != nil {
			return err
		}
		if err := db.Model(&models.CPCampaign{}).Where("campaign_id = ?", cid).
			Updates(map[string]any{
				"amount_raised_wei": gorm.Expr("amount_raised_wei + ?::numeric", amt),
				"donor_count":       gorm.Expr("donor_count + 1"),
				"updated_at":        time.Now(),
			}).Error; err != nil {
			return err
		}
		return nil
	}
	if err := db.Model(&models.CPContribution{}).
		Where("campaign_id = ? AND LOWER(contributor_address) = ?", cid, contrib).
		Updates(map[string]any{
			"total_contributed_wei": gorm.Expr("total_contributed_wei + ?::numeric", amt),
			"last_donated_at":       ev.TS,
			"updated_at":            time.Now(),
		}).Error; err != nil {
		return err
	}
	return db.Model(&models.CPCampaign{}).Where("campaign_id = ?", cid).
		Updates(map[string]any{
			"amount_raised_wei": gorm.Expr("amount_raised_wei + ?::numeric", amt),
			"updated_at":        time.Now(),
		}).Error
}

func applyCampaignFinalized(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r cfRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	state := campaignFailedRefundable
	code := 3
	if r.Successful {
		state = campaignSuccessful
		code = 2
	}
	updates := map[string]any{
		"state":        state,
		"state_code":   code,
		"finalized_at": ev.TS,
		"updated_at":   time.Now(),
	}
	if r.Successful {
		updates["success_at"] = ev.TS
	}
	if err := db.Model(&models.CPCampaign{}).Where("campaign_id = ?", cid).Updates(updates).Error; err != nil {
		return err
	}

	if !r.Successful {
		return nil
	}
	var camp models.CPCampaign
	if err := db.Where("campaign_id = ?", cid).First(&camp).Error; err != nil {
		return err
	}
	return db.Model(&models.CPProposal{}).Where("proposal_id = ?", camp.ProposalID).Updates(map[string]any{
		"status":      proposalSettled,
		"status_code": 7,
		"updated_at":  time.Now(),
	}).Error
}

func applyRefundClaimed(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r rcRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	contrib := strings.ToLower(r.Contributor.Hex())
	amt := r.Amount
	return db.Model(&models.CPContribution{}).
		Where("campaign_id = ? AND LOWER(contributor_address) = ?", cid, contrib).
		Updates(map[string]any{
			"refund_claimed_wei": gorm.Expr("refund_claimed_wei + ?::numeric", amt),
			"last_refund_at":     ev.TS,
			"updated_at":         time.Now(),
		}).Error
}

func applyDeveloperAdded(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r daRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	dev := strings.ToLower(r.Developer.Hex())
	row := models.CPCampaignDeveloper{
		CampaignID:       cid,
		DeveloperAddress: dev,
		IsActive:         true,
		AddedTxHash:      strPtr(ev.TxHash),
		AddedAt:          &ev.TS,
	}
	if err := db.Create(&row).Error; err != nil {
		return err
	}
	return db.Model(&models.CPCampaign{}).Where("campaign_id = ?", cid).
		Updates(map[string]any{
			"developer_count": gorm.Expr("developer_count + 1"),
			"updated_at":      time.Now(),
		}).Error
}

func applyDeveloperRemoved(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r drRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	dev := strings.ToLower(r.Developer.Hex())
	res := db.Model(&models.CPCampaignDeveloper{}).
		Where("campaign_id = ? AND LOWER(developer_address) = ? AND is_active = ?", cid, dev, true).
		Updates(map[string]any{
			"is_active":       false,
			"removed_tx_hash": ev.TxHash,
			"removed_at":      ev.TS,
			"updated_at":      time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return db.Model(&models.CPCampaign{}).Where("campaign_id = ?", cid).
		Updates(map[string]any{
			"developer_count": gorm.Expr("GREATEST(developer_count - 1, 0)"),
			"updated_at":      time.Now(),
		}).Error
}

func applyMilestoneApproved(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r maRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	mi := int(mustInt64String(r.MilestoneIndex))
	return db.Model(&models.CPCampaignMilestone{}).
		Where("campaign_id = ? AND milestone_index = ?", cid, mi).
		Updates(map[string]any{
			"approved":    true,
			"approved_at": ev.TS,
			"updated_at":  time.Now(),
		}).Error
}

func applyMilestoneShareClaimed(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r mscRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	mi := int(mustInt64String(r.MilestoneIndex))
	dev := strings.ToLower(r.Developer.Hex())

	claim := models.CPMilestoneClaim{
		CampaignID:       cid,
		MilestoneIndex:   mi,
		DeveloperAddress: dev,
		ClaimedAmountWei: r.Amount,
		ClaimedTxHash:    ev.TxHash,
		ClaimedAt:        ev.TS,
	}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "campaign_id"},
			{Name: "milestone_index"},
			{Name: "developer_address"},
		},
		DoNothing: true,
	}).Create(&claim).Error; err != nil {
		return err
	}
	return db.Model(&models.CPCampaignMilestone{}).
		Where("campaign_id = ? AND milestone_index = ?", cid, mi).
		Updates(map[string]any{"claimed": true, "updated_at": time.Now()}).Error
}

func applyPlatformDonated(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r pdRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	row := models.CPPlatformFundMovement{
		Direction:      "donation",
		WalletAddress:  strings.ToLower(r.Donor.Hex()),
		AmountWei:      r.Amount,
		TxHash:         ev.TxHash,
		LogIndex:       ev.LogIndex,
		BlockNumber:    ev.Block,
		BlockTimestamp: ev.TS,
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func applyPlatformFundsWithdrawn(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r pfwRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	row := models.CPPlatformFundMovement{
		Direction:      "withdrawal",
		WalletAddress:  strings.ToLower(r.To.Hex()),
		AmountWei:      r.Amount,
		TxHash:         ev.TxHash,
		LogIndex:       ev.LogIndex,
		BlockNumber:    ev.Block,
		BlockTimestamp: ev.TS,
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func applyOwnershipTransferred(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r otRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	newO := strings.ToLower(r.NewOwner.Hex())
	src := cpIndexerSource(ev)
	row := models.CPSystemState{
		ContractAddress:   contract,
		OwnerAddress:      newO,
		Paused:            false,
		Source:            src,
		SourceBlockNumber: &ev.Block,
		SyncedAt:          &ev.TS,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "contract_address"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"owner_address", "source", "source_block_number", "synced_at", "updated_at",
		}),
	}).Create(&row).Error
}

func applyPaused(db *gorm.DB, contract string, ev normalizedEvent) error {
	return db.Model(&models.CPSystemState{}).Where("contract_address = ?", contract).
		Updates(map[string]any{"paused": true, "synced_at": ev.TS, "updated_at": time.Now()}).Error
}

func applyUnpaused(db *gorm.DB, contract string, ev normalizedEvent) error {
	return db.Model(&models.CPSystemState{}).Where("contract_address = ?", contract).
		Updates(map[string]any{"paused": false, "synced_at": ev.TS, "updated_at": time.Now()}).Error
}

func applyProposalInitiatorUpdated(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r piuRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	addr := strings.ToLower(r.Account.Hex())
	blk := ev.Block
	src := cpIndexerSource(ev)
	if r.Allowed {
		var existing models.CPWalletRole
		err := db.Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ? AND (scope_id IS NULL OR scope_id = '')",
			addr, "proposal_initiator", "global").First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row := models.CPWalletRole{
				WalletAddress:     addr,
				Role:              "proposal_initiator",
				ScopeType:         "global",
				Active:            true,
				DerivedFrom:       src,
				Source:            src,
				SourceBlockNumber: &blk,
				SyncedAt:          &ev.TS,
			}
			return db.Create(&row).Error
		}
		if err != nil {
			return err
		}
		return db.Model(&existing).Updates(map[string]any{
			"active": true, "derived_from": src, "source": src,
			"source_block_number": blk, "synced_at": ev.TS, "updated_at": time.Now(),
		}).Error
	}
	return db.Model(&models.CPWalletRole{}).
		Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ? AND (scope_id IS NULL OR scope_id = '')",
			addr, "proposal_initiator", "global").
		Updates(map[string]any{"active": false, "updated_at": time.Now()}).Error
}

func applyStaleFundsSwept(db *gorm.DB, contract string, ev normalizedEvent) error {
	var r sfsRow
	if err := json.Unmarshal(ev.Raw, &r); err != nil {
		return err
	}
	cid := uint64(r.CampaignID)
	return db.Model(&models.CPCampaign{}).Where("campaign_id = ?", cid).
		Updates(map[string]any{"dormant_funds_swept": true, "updated_at": time.Now()}).Error
}

func mustInt64String(s string) int64 {
	s = strings.TrimSpace(s)
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return 0
	}
	if !n.IsInt64() {
		return 0
	}
	return n.Int64()
}

func u64Ptr(u uint64) *uint64 { return &u }

func strPtr(s string) *string { return &s }
