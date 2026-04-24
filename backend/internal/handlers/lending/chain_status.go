package lending

import (
	"net/http"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// ChainStatus GET /api/lending/chain-status
// @Summary      借贷专用 RPC 连通性
// @Description  使用 LENDING_ETH_RPC_URL 或 BASE_ETH_RPC_URL 建立的客户端，与 GET /api/chain/status（ETH_RPC_URL）完全隔离。
// @Tags         lending
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Router       /api/lending/chain-status [get]
func ChainStatus(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.LendingChain == nil || h.LendingChain.Eth() == nil {
			c.JSON(http.StatusOK, gin.H{
				"configured": false,
				"scope":      "lending_only",
				"message":    "LENDING_ETH_RPC_URL / BASE_ETH_RPC_URL not set or dial failed (isolated from ETH_RPC_URL)",
			})
			return
		}
		id, err := h.LendingChain.ChainID(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"configured": true,
			"scope":      "lending_only",
			"chain_id":   *id,
			"note":       "此 chain_id 来自借贷专用 RPC，与银行/Code Pulse 使用的 ETH_RPC_URL 无关。",
		})
	}
}
