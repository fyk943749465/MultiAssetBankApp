package nft

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// resolveChainID 从 RPC 读取 chain id（与索引器、库中 chain_id 一致）。
func resolveChainID(c *gin.Context, h *handlers.Handlers) (int64, bool) {
	if h.Chain != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()
		id, err := h.Chain.ChainID(ctx)
		if err == nil && id != nil {
			return int64(*id), true
		}
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ETH_RPC_URL / chain client unavailable"})
	return 0, false
}

func queryPage(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}
	return page, pageSize
}
