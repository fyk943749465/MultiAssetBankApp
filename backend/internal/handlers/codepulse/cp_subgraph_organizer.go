package codepulse

import (
	"bytes"
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

// flexGraphScalar 解码子图 BigInt 等在 JSON 中常见的 string 或 number（Graph-node 常发 number，直接 string 会 Unmarshal 失败）。
type flexGraphScalar string

func (f *flexGraphScalar) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexGraphScalar(strings.TrimSpace(s))
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexGraphScalar(strings.TrimSpace(n.String()))
	return nil
}

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
	ProposalID     flexGraphScalar `json:"proposalId"`
	GithubURL      string          `json:"githubUrl"`
	Target         flexGraphScalar `json:"target"`
	Duration       flexGraphScalar `json:"duration"`
	BlockTimestamp flexGraphScalar `json:"blockTimestamp"`
	TxHash         flexGraphScalar `json:"transactionHash"`
}

type sgPropReviewed struct {
	ProposalID flexGraphScalar `json:"proposalId"`
	Approved   bool            `json:"approved"`
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
			ProposalID flexGraphScalar `json:"proposalId"`
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
	if _, ok := n.SetString(s, 0); !ok {
		return 0, fmt.Errorf("parse")
	}
	if !n.IsUint64() {
		return 0, fmt.Errorf("overflow")
	}
	return n.Uint64(), nil
}

// parseGraphQLScalarToString 解码 GraphQL 中 BigInt 等标量在 JSON 里常见的 string 或 number 两种写法。
func parseGraphQLScalarToString(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "", fmt.Errorf("empty scalar")
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", err
		}
		return strings.TrimSpace(s), nil
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err != nil {
		return "", err
	}
	return strings.TrimSpace(n.String()), nil
}

// parseWeiFromGraphScalar 将 amount 等字段解析为 wei（十进制或 0x 十六进制）。
func parseWeiFromGraphScalar(raw json.RawMessage) (*big.Int, bool) {
	s, err := parseGraphQLScalarToString(raw)
	if err != nil || s == "" {
		return nil, false
	}
	v := new(big.Int)
	if _, ok := v.SetString(s, 0); !ok {
		return nil, false
	}
	return v, true
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
