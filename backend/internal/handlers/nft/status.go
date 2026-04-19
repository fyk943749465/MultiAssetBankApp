package nft

import (
	"net/http"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SyncStatus GET /api/nft/sync-status
func SyncStatus(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"chain_id":                    chainID,
			"read_policy":                 "subgraph_first",
			"nft_subgraph_configured":     h.SubgraphNft != nil && h.SubgraphNft.Configured(),
			"nft_subgraph_persists_to_pg": false,
			"note": "合集列表、挂单列表：已配置 SUBGRAPH_NFT_URL 且子图本页有数据时优先子图（通常快于扫块入库）；子图不可用或本页无数据时用 PostgreSQL。子图不向 PG 写入。",
		})
	}
}
