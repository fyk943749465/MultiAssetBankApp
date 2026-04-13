package chain

import (
	"net/http"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Status returns chain connectivity info when ETH_RPC_URL is set.
// @Summary      Chain status
// @Description  When ETH_RPC_URL is configured and dial succeeds, returns `configured: true` and `chain_id`. Otherwise `configured: false` and a short message.
// @Tags         chain
// @Produce      json
// @Success      200 {object} handlers.ChainStatusResp
// @Failure      503 {object} handlers.ErrorJSON "e.g. chain id read failed"
// @Router       /api/chain/status [get]
func Status(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
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
}
