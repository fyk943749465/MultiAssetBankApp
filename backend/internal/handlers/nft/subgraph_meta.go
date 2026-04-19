package nft

import (
	"context"
	"net/http"
	"time"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

const subgraphMetaQuery = `query { _meta { block { number hash } } }`

// SubgraphMeta GET /api/nft/subgraph/meta — 探测 SUBGRAPH_NFT_URL 是否可用。
func SubgraphMeta(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.SubgraphNft == nil || !h.SubgraphNft.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "SUBGRAPH_NFT_URL not configured"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		raw, err := h.SubgraphNft.Query(ctx, subgraphMetaQuery, nil)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", raw)
	}
}
