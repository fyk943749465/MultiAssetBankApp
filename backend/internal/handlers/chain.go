package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChainStatus returns chain connectivity info when ETH_RPC_URL is set.
func (h *Handlers) ChainStatus(c *gin.Context) {
	if h.Chain == nil || h.Chain.Eth() == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured": false,
			"message":    "ETH_RPC_URL not set or dial failed",
		})
		return
	}
	id, err := h.Chain.ChainID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"configured": true, "chain_id": *id})
}
