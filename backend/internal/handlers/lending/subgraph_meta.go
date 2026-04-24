package lending

import (
	"context"
	"net/http"
	"time"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

const subgraphMetaQuery = `query { _meta { block { number hash } } }`

// SubgraphMeta GET /api/lending/subgraph/meta
// @Summary      探测借贷子图 GraphQL
// @Tags         lending
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/subgraph/meta [get]
func SubgraphMeta(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.SubgraphLending == nil || !h.SubgraphLending.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "SUBGRAPH_LENDING_URL not configured"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		raw, err := h.SubgraphLending.Query(ctx, subgraphMetaQuery, nil)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", raw)
	}
}
