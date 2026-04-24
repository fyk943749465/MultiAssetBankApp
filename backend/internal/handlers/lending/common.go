package lending

import (
	"strconv"
	"strings"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

const defaultLendingChainID = int64(84532)

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

// resolveLendingChainID 借贷库表 chain_id；优先 query，其次 Handlers 配置，默认 Base Sepolia 84532。
func resolveLendingChainID(c *gin.Context, h *handlers.Handlers) int64 {
	if v := strings.TrimSpace(c.Query("chain_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	if h != nil && h.LendingChainID > 0 {
		return h.LendingChainID
	}
	return defaultLendingChainID
}
