package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

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

type subgraphEventRow struct {
	SubgraphEntityID string `json:"subgraph_entity_id"`
	TokenAddress     string `json:"token_address"`
	UserAddress      string `json:"user_address"`
	AmountRaw        string `json:"amount_raw"`
	BlockNumber      uint64 `json:"block_number"`
	BlockTime        string `json:"block_time"`
	TxHash           string `json:"tx_hash"`
}

type subgraphDepositsData struct {
	Depositeds []subgraphEventJSON `json:"depositeds"`
}

type subgraphWithdrawalsData struct {
	Withdrawns []subgraphEventJSON `json:"withdrawns"`
}

// Subgraph returns JSON with BigInt as string; Bytes as 0x-prefixed hex.
type subgraphEventJSON struct {
	ID               string `json:"id"`
	Token            string `json:"token"`
	User             string `json:"user"`
	Amount           string `json:"amount"`
	BlockNumber      string `json:"blockNumber"`
	BlockTimestamp   string `json:"blockTimestamp"`
	TransactionHash  string `json:"transactionHash"`
}

func (h *Handlers) bankSubgraphLimit(c *gin.Context) int {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	return limit
}

// BankSubgraphDeposits returns Deposited events from The Graph (same contract as indexer).
func (h *Handlers) BankSubgraphDeposits(c *gin.Context) {
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
		"first": h.bankSubgraphLimit(c),
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

// BankSubgraphWithdrawals returns Withdrawn events from The Graph.
func (h *Handlers) BankSubgraphWithdrawals(c *gin.Context) {
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
		"first": h.bankSubgraphLimit(c),
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

func mapSubgraphEvents(in []subgraphEventJSON) ([]subgraphEventRow, error) {
	out := make([]subgraphEventRow, 0, len(in))
	for _, e := range in {
		row, err := mapOneSubgraphEvent(e)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func mapOneSubgraphEvent(e subgraphEventJSON) (subgraphEventRow, error) {
	var row subgraphEventRow
	row.SubgraphEntityID = e.ID
	row.TokenAddress = normalizeHexAddr(e.Token)
	row.UserAddress = normalizeHexAddr(e.User)
	row.TxHash = normalizeHexHash(e.TransactionHash)
	if e.Amount == "" {
		return subgraphEventRow{}, fmt.Errorf("empty amount")
	}
	if _, ok := new(big.Int).SetString(e.Amount, 10); !ok {
		return subgraphEventRow{}, fmt.Errorf("invalid amount: %q", e.Amount)
	}
	row.AmountRaw = e.Amount

	bn := strings.TrimSpace(e.BlockNumber)
	if bn == "" {
		return subgraphEventRow{}, fmt.Errorf("empty blockNumber")
	}
	u, err := strconv.ParseUint(bn, 10, 64)
	if err != nil {
		return subgraphEventRow{}, fmt.Errorf("blockNumber: %w", err)
	}
	row.BlockNumber = u

	ts := strings.TrimSpace(e.BlockTimestamp)
	if ts == "" {
		return subgraphEventRow{}, fmt.Errorf("empty blockTimestamp")
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return subgraphEventRow{}, fmt.Errorf("blockTimestamp: %w", err)
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
