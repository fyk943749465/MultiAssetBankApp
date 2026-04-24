package lending

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go-chain/backend/internal/handlers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

// NativeBalance GET /api/lending/native-balance
// @Summary      查询地址在借贷链上的原生 ETH 余额
// @Description  使用借贷专用 RPC（LENDING_ETH_RPC_URL / BASE_ETH_RPC_URL），与 ETH_RPC_URL 隔离。余额为 wei 十进制字符串，避免 JSON 大整数精度问题。
// @Tags         lending
// @Produce      json
// @Param        address query string false "0x 地址，与 user_address 二选一"
// @Param        user_address query string false "同 address，便于与其它借贷接口一致"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      502 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Router       /api/lending/native-balance [get]
func NativeBalance(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		addrStr := strings.TrimSpace(c.Query("address"))
		if addrStr == "" {
			addrStr = strings.TrimSpace(c.Query("user_address"))
		}
		if addrStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing address: set query address or user_address"})
			return
		}
		if !common.IsHexAddress(addrStr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hex address"})
			return
		}
		if h.LendingChain == nil || h.LendingChain.Eth() == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "lending RPC not configured: set LENDING_ETH_RPC_URL or BASE_ETH_RPC_URL",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		addr := common.HexToAddress(addrStr)
		bal, err := h.LendingChain.Eth().BalanceAt(ctx, addr, nil)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		var chainID uint64
		if id, errCID := h.LendingChain.ChainID(ctx); errCID == nil && id != nil {
			chainID = *id
		}

		c.JSON(http.StatusOK, gin.H{
			"address":     addr.Hex(),
			"balance_wei": bal.String(),
			"chain_id":    chainID,
			"rpc_scope":   "lending_only",
		})
	}
}
