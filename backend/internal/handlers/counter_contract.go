package handlers

import (
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gin-gonic/gin"
)

// CounterValue calls view function get().
// @Summary      Counter current value
// @Description  Reads `get()` from the configured counter contract (requires COUNTER_CONTRACT_ADDRESS and ETH_RPC_URL).
// @Tags         contract
// @Produce      json
// @Success      200 {object} CounterValueResp
// @Failure      503 {object} ErrorJSON "contract or RPC not configured"
// @Failure      502 {object} ErrorJSON "RPC / contract call error"
// @Router       /api/contract/counter/value [get]
func (h *Handlers) CounterValue(c *gin.Context) {
	if h.Counter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "counter contract not configured (set COUNTER_CONTRACT_ADDRESS)"})
		return
	}
	if h.Chain == nil || h.Chain.Eth() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ETH_RPC_URL not configured"})
		return
	}
	v, err := h.Counter.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"value": v.String()})
}

// CounterIncrement sends a transaction calling count().
// @Summary      Counter increment (tx)
// @Description  Sends an on-chain transaction calling `count()` using ETH_PRIVATE_KEY. Request has **no body**.
// @Tags         contract
// @Produce      json
// @Success      200 {object} CounterIncrementResp
// @Failure      503 {object} ErrorJSON "contract, RPC, or private key not configured"
// @Failure      502 {object} ErrorJSON "transaction send error"
// @Failure      500 {object} ErrorJSON
// @Router       /api/contract/counter/count [post]
func (h *Handlers) CounterIncrement(c *gin.Context) {
	if h.Counter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "counter contract not configured (set COUNTER_CONTRACT_ADDRESS)"})
		return
	}
	if h.Chain == nil || h.Chain.Eth() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ETH_RPC_URL not configured"})
		return
	}
	if h.TxKey == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ETH_PRIVATE_KEY not configured"})
		return
	}
	chainID, err := h.Chain.Eth().ChainID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	auth, err := bind.NewKeyedTransactorWithChainID(h.TxKey, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tx, err := h.Counter.Count(c.Request.Context(), auth)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tx_hash": tx.Hash().Hex()})
}
