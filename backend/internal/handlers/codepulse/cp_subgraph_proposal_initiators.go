package codepulse

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

// 分页拉取 ProposalInitiatorUpdated（immutable），按区块与 id 排序后折叠为当前链上白名单。
const cpSubgraphProposalInitiatorUpdatesPage = `
query CpProposalInitiatorUpdateds($first: Int!, $skip: Int!) {
  proposalInitiatorUpdateds(
    first: $first
    skip: $skip
    orderBy: blockNumber
    orderDirection: asc
  ) {
    id
    account
    allowed
    blockNumber
    blockTimestamp
    transactionHash
  }
}
`

type sgProposalInitiatorUpdatedWire struct {
	ID            string          `json:"id"`
	Account       flexGraphScalar `json:"account"`
	Allowed       bool            `json:"allowed"`
	BlockNumber   flexGraphScalar `json:"blockNumber"`
	BlockTimestamp flexGraphScalar `json:"blockTimestamp"`
	TxHash        flexGraphScalar `json:"transactionHash"`
}

type sgProposalInitiatorUpdatedPage struct {
	ProposalInitiatorUpdateds []sgProposalInitiatorUpdatedWire `json:"proposalInitiatorUpdateds"`
}

func flexScalarUint64(f flexGraphScalar) uint64 {
	s := strings.TrimSpace(string(f))
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// foldProposalInitiatorUpdatedsToActiveAddrs 按时间顺序应用 allowed 标记，返回当前仍为 true 的地址（小写 0x）。
func foldProposalInitiatorUpdatedsToActiveAddrs(rows []sgProposalInitiatorUpdatedWire) []string {
	type keyed struct {
		block uint64
		id    string
		row   sgProposalInitiatorUpdatedWire
	}
	keyedRows := make([]keyed, 0, len(rows))
	for _, r := range rows {
		keyedRows = append(keyedRows, keyed{
			block: flexScalarUint64(r.BlockNumber),
			id:    r.ID,
			row:   r,
		})
	}
	sort.Slice(keyedRows, func(i, j int) bool {
		if keyedRows[i].block != keyedRows[j].block {
			return keyedRows[i].block < keyedRows[j].block
		}
		return keyedRows[i].id < keyedRows[j].id
	})

	state := make(map[string]bool)
	for _, k := range keyedRows {
		raw := strings.TrimSpace(string(k.row.Account))
		if raw == "" {
			continue
		}
		if !strings.HasPrefix(raw, "0x") {
			raw = "0x" + raw
		}
		if !common.IsHexAddress(raw) {
			continue
		}
		addr := normalizeAddress(common.HexToAddress(raw).Hex())
		state[addr] = k.row.Allowed
	}

	out := make([]string, 0, len(state))
	for addr, allowed := range state {
		if allowed {
			out = append(out, addr)
		}
	}
	sort.Strings(out)
	return out
}

const proposalInitiatorSubgraphPageSize = 500
const proposalInitiatorSubgraphMaxPages = 200

// queryProposalInitiatorAllowlistFromSubgraph 从子图事件折叠出当前链上允许的 initiator 地址列表。
func queryProposalInitiatorAllowlistFromSubgraph(ctx context.Context, h *handlers.Handlers) ([]string, error) {
	if h == nil || h.SubgraphCodePulse == nil || !h.SubgraphCodePulse.Configured() {
		return nil, fmt.Errorf("subgraph not configured")
	}
	var all []sgProposalInitiatorUpdatedWire
	for page := 0; page < proposalInitiatorSubgraphMaxPages; page++ {
		skip := page * proposalInitiatorSubgraphPageSize
		raw, err := h.SubgraphCodePulse.Query(ctx, cpSubgraphProposalInitiatorUpdatesPage, map[string]any{
			"first": proposalInitiatorSubgraphPageSize,
			"skip":  skip,
		})
		if err != nil {
			return nil, err
		}
		var wrap sgProposalInitiatorUpdatedPage
		if err := json.Unmarshal(raw, &wrap); err != nil {
			return nil, err
		}
		if len(wrap.ProposalInitiatorUpdateds) == 0 {
			break
		}
		all = append(all, wrap.ProposalInitiatorUpdateds...)
		if len(wrap.ProposalInitiatorUpdateds) < proposalInitiatorSubgraphPageSize {
			break
		}
	}
	return foldProposalInitiatorUpdatedsToActiveAddrs(all), nil
}

// proposalInitiatorAllowlist resolves the display list: subgraph fold first, then PostgreSQL active roles.
func proposalInitiatorAllowlist(ctx context.Context, h *handlers.Handlers) (addrs []string, source string) {
	if h != nil && h.SubgraphCodePulse != nil && h.SubgraphCodePulse.Configured() {
		if sg, err := queryProposalInitiatorAllowlistFromSubgraph(ctx, h); err == nil {
			return sg, "subgraph"
		}
	}
	if h == nil || h.DB == nil {
		return []string{}, "database"
	}
	var roles []models.CPWalletRole
	h.DB.Where("role = ? AND scope_type = ? AND active = true", "proposal_initiator", "global").
		Order("wallet_address ASC").Find(&roles)
	addrs = make([]string, 0, len(roles))
	for _, r := range roles {
		addrs = append(addrs, normalizeAddress(r.WalletAddress))
	}
	return addrs, "database"
}

func initiatorAllowlistAsDashboardRows(addrs []string, derived string) []ginHInitiatorStub {
	out := make([]ginHInitiatorStub, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, ginHInitiatorStub{
			WalletAddress: a,
			Role:          "proposal_initiator",
			ScopeType:     "global",
			Active:        true,
			DerivedFrom:   derived,
			Source:        derived,
		})
	}
	return out
}

// ginHInitiatorStub 与 models.CPWalletRole JSON 子集兼容，供 admin dashboard 在仅知地址时使用。
type ginHInitiatorStub struct {
	WalletAddress string `json:"wallet_address"`
	Role          string `json:"role"`
	ScopeType     string `json:"scope_type"`
	Active        bool   `json:"active"`
	DerivedFrom   string `json:"derived_from"`
	Source        string `json:"source"`
}

func proposalInitiatorAllowlistFromDB(db *gorm.DB) []models.CPWalletRole {
	var roles []models.CPWalletRole
	db.Where("role = ? AND scope_type = ? AND active = true", "proposal_initiator", "global").
		Order("created_at DESC").Find(&roles)
	return roles
}
