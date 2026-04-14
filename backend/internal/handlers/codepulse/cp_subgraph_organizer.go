package codepulse

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
)

// 子图：按发起人拉取提案（用于 PG 未同步时工作台展示；流程仍以 DB 为准）。
const cpSubgraphProposalSubmittedByOrganizer = `
query CpOrgProposals($org: Bytes!) {
  proposalSubmitteds(
    first: 100
    orderBy: blockNumber
    orderDirection: desc
    where: { organizer: $org }
  ) {
    proposalId
    githubUrl
    target
    duration
    blockTimestamp
    transactionHash
  }
}
`

const cpSubgraphProposalReviews = `
query CpProposalReviews($ids: [BigInt!]!) {
  proposalRevieweds(where: { proposalId_in: $ids }) {
    proposalId
    approved
  }
}
`

type sgPropSubmitted struct {
	ProposalID     string `json:"proposalId"`
	GithubURL      string `json:"githubUrl"`
	Target         string `json:"target"`
	Duration       string `json:"duration"`
	BlockTimestamp string `json:"blockTimestamp"`
	TxHash         string `json:"transactionHash"`
}

type sgPropReviewed struct {
	ProposalID string `json:"proposalId"`
	Approved   bool   `json:"approved"`
}

// organizerHasProposalInSubgraph 子图中是否存在该地址作为 organizer 的 ProposalSubmitted（轻量，用于解锁工作台入口）。
func organizerHasProposalInSubgraph(ctx context.Context, h *handlers.Handlers, organizerLower string) bool {
	if h == nil || h.SubgraphCodePulse == nil || !h.SubgraphCodePulse.Configured() {
		return false
	}
	if !common.IsHexAddress(organizerLower) {
		return false
	}
	org := common.HexToAddress(organizerLower)
	vars := map[string]any{"org": org.Hex()}
	raw, err := h.SubgraphCodePulse.Query(ctx, cpSubgraphProposalSubmittedByOrganizer, vars)
	if err != nil {
		return false
	}
	var data struct {
		ProposalSubmitteds []struct {
			ProposalID string `json:"proposalId"`
		} `json:"proposalSubmitteds"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return false
	}
	return len(data.ProposalSubmitteds) > 0
}

// mergeSubgraphProposalsForOrganizer 发起人工作台只读列表：与 OrganizerProposalsSubgraphView 相同（以子图事件推导展示状态；动作预检仍以 PG 为准）。
func mergeSubgraphProposalsForOrganizer(ctx context.Context, h *handlers.Handlers, organizerLower string, fromPG []models.CPProposal) ([]models.CPProposal, string) {
	return OrganizerProposalsSubgraphView(ctx, h, organizerLower, fromPG)
}

func parseSubgraphUint(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return 0, fmt.Errorf("parse")
	}
	if !n.IsUint64() {
		return 0, fmt.Errorf("overflow")
	}
	return n.Uint64(), nil
}

func parseSubgraphTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(sec, 0).UTC()
	return &t
}

func mustParseInt64(s string) int64 {
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
