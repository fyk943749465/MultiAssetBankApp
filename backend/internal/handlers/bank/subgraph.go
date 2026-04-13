package bank

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

const bankSubgraphDepositsQuery = `
query BankDeposits($first: Int!, $user: Bytes) {
  depositeds(
    first: $first
    orderBy: blockNumber
    orderDirection: desc
    where: { user: $user }
  ) {
    id
    token
    user
    amount
    blockNumber
    blockTimestamp
    transactionHash
  }
}`

const bankSubgraphWithdrawalsQuery = `
query BankWithdrawals($first: Int!, $user: Bytes) {
  withdrawns(
    first: $first
    orderBy: blockNumber
    orderDirection: desc
    where: { user: $user }
  ) {
    id
    token
    user
    amount
    blockNumber
    blockTimestamp
    transactionHash
  }
}`

type subgraphDepositsData struct {
	Depositeds []subgraphEventJSON `json:"depositeds"`
}

type subgraphWithdrawalsData struct {
	Withdrawns []subgraphEventJSON `json:"withdrawns"`
}

type subgraphEventJSON struct {
	ID              string `json:"id"`
	Token           string `json:"token"`
	User            string `json:"user"`
	Amount          string `json:"amount"`
	BlockNumber     string `json:"blockNumber"`
	BlockTimestamp  string `json:"blockTimestamp"`
	TransactionHash string `json:"transactionHash"`
}

func bankSubgraphLimit(c *gin.Context) int {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	return limit
}

// SubgraphDeposits returns Deposited events from The Graph (same contract as indexer).
// @Summary      Bank deposits (The Graph)
// @Description  Queries the configured subgraph for `Deposited` entities. Requires SUBGRAPH_URL (and optional SUBGRAPH_API_KEY).
// @Tags         bank
// @Produce      json
// @Param        user  query string true "Wallet address (0x-prefixed)" example(0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045)
// @Param        limit query int false "Max rows (default 50, max 200)" default(50) minimum(1) maximum(200)
// @Success      200 {object} handlers.SubgraphDepositsResp
// @Failure      400 {object} handlers.ErrorJSON "missing or invalid user"
// @Failure      503 {object} handlers.ErrorJSON "subgraph not configured"
// @Failure      502 {object} handlers.ErrorJSON "subgraph HTTP or parse error"
// @Router       /api/bank/subgraph/deposits [get]
func SubgraphDeposits(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Subgraph == nil || !h.Subgraph.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "subgraph not configured (set SUBGRAPH_URL and SUBGRAPH_API_KEY)",
			})
			return
		}
		user := strings.TrimSpace(c.Query("user"))
		if user == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing user (wallet address) query param"})
			return
		}
		if !strings.HasPrefix(strings.ToLower(user), "0x") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user must be a 0x-prefixed address"})
			return
		}
		user = strings.ToLower(user)

		vars := map[string]any{
			"first": bankSubgraphLimit(c),
			"user":  user,
		}
		data, err := h.Subgraph.Query(c.Request.Context(), bankSubgraphDepositsQuery, vars)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		var parsed subgraphDepositsData
		if err := json.Unmarshal(data, &parsed); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "subgraph: parse deposits: " + err.Error()})
			return
		}
		rows, err := mapSubgraphEvents(parsed.Depositeds)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"deposits": rows, "source": "subgraph"})
	}
}

// SubgraphWithdrawals returns Withdrawn events from The Graph.
// @Summary      Bank withdrawals (The Graph)
// @Description  Queries the configured subgraph for `Withdrawn` entities. Requires SUBGRAPH_URL (and optional SUBGRAPH_API_KEY).
// @Tags         bank
// @Produce      json
// @Param        user  query string true "Wallet address (0x-prefixed)" example(0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045)
// @Param        limit query int false "Max rows (default 50, max 200)" default(50) minimum(1) maximum(200)
// @Success      200 {object} handlers.SubgraphWithdrawalsResp
// @Failure      400 {object} handlers.ErrorJSON "missing or invalid user"
// @Failure      503 {object} handlers.ErrorJSON "subgraph not configured"
// @Failure      502 {object} handlers.ErrorJSON "subgraph HTTP or parse error"
// @Router       /api/bank/subgraph/withdrawals [get]
func SubgraphWithdrawals(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Subgraph == nil || !h.Subgraph.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "subgraph not configured (set SUBGRAPH_URL and SUBGRAPH_API_KEY)",
			})
			return
		}
		user := strings.TrimSpace(c.Query("user"))
		if user == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing user (wallet address) query param"})
			return
		}
		if !strings.HasPrefix(strings.ToLower(user), "0x") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user must be a 0x-prefixed address"})
			return
		}
		user = strings.ToLower(user)

		vars := map[string]any{
			"first": bankSubgraphLimit(c),
			"user":  user,
		}
		data, err := h.Subgraph.Query(c.Request.Context(), bankSubgraphWithdrawalsQuery, vars)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		var parsed subgraphWithdrawalsData
		if err := json.Unmarshal(data, &parsed); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "subgraph: parse withdrawals: " + err.Error()})
			return
		}
		rows, err := mapSubgraphEvents(parsed.Withdrawns)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"withdrawals": rows, "source": "subgraph"})
	}
}

func mapSubgraphEvents(in []subgraphEventJSON) ([]handlers.SubgraphEventRow, error) {
	out := make([]handlers.SubgraphEventRow, 0, len(in))
	for _, e := range in {
		row, err := mapOneSubgraphEvent(e)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func mapOneSubgraphEvent(e subgraphEventJSON) (handlers.SubgraphEventRow, error) {
	var row handlers.SubgraphEventRow
	row.SubgraphEntityID = e.ID
	row.TokenAddress = normalizeHexAddr(e.Token)
	row.UserAddress = normalizeHexAddr(e.User)
	row.TxHash = normalizeHexHash(e.TransactionHash)
	if e.Amount == "" {
		return handlers.SubgraphEventRow{}, fmt.Errorf("empty amount")
	}
	if _, ok := new(big.Int).SetString(e.Amount, 10); !ok {
		return handlers.SubgraphEventRow{}, fmt.Errorf("invalid amount: %q", e.Amount)
	}
	row.AmountRaw = e.Amount

	bn := strings.TrimSpace(e.BlockNumber)
	if bn == "" {
		return handlers.SubgraphEventRow{}, fmt.Errorf("empty blockNumber")
	}
	u, err := strconv.ParseUint(bn, 10, 64)
	if err != nil {
		return handlers.SubgraphEventRow{}, fmt.Errorf("blockNumber: %w", err)
	}
	row.BlockNumber = u

	ts := strings.TrimSpace(e.BlockTimestamp)
	if ts == "" {
		return handlers.SubgraphEventRow{}, fmt.Errorf("empty blockTimestamp")
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return handlers.SubgraphEventRow{}, fmt.Errorf("blockTimestamp: %w", err)
	}
	row.BlockTime = time.Unix(sec, 0).UTC().Format(time.RFC3339Nano)

	return row, nil
}

func normalizeHexAddr(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "0x") {
		return s
	}
	return "0x" + s
}

func normalizeHexHash(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "0x") {
		return s
	}
	return "0x" + s
}
