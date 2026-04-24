package lending

import (
	"net/http"

	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SyncStatus GET /api/lending/sync-status
// @Summary      借贷读策略与子图配置
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "库表 chain_id，默认 84532"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/sync-status [get]
func SyncStatus(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		chainID := resolveLendingChainID(c, h)
		c.JSON(http.StatusOK, gin.H{
			"chain_id":                        chainID,
			"database_configured":             h.DB != nil,
			"read_policy":                     "subgraph_first_then_database",
			"lending_subgraph_configured":     h.SubgraphLending != nil && h.SubgraphLending.Configured(),
			"lending_subgraph_persists_to_pg": false,
			"lending_rpc_configured":          h.LendingChain != nil && h.LendingChain.Eth() != nil,
			"lending_subgraph_api_key_source": h.LendingSubgraphAPIKeySource,
			"lending_subgraph_bearer_present": h.LendingSubgraphAPIKeyPresent,
			"note":                            "列表类接口：子图已配置且能返回非空结果时优先子图；子图失败或空结果时用 PostgreSQL（RPC 扫块落库为存借还清算等权威事实）。",
			"configuration_hints": gin.H{
				"lending_subgraph_bearer": "借贷子图仅使用 SUBGRAPH_LENDING_API_KEY 或 SUBGRAPH_API_SECOND_KEY（The Graph 另一账户）；绝不使用 SUBGRAPH_API_KEY。",
				"lending_json_rpc":        "借贷专用节点：LENDING_ETH_RPC_URL 优先，否则 BASE_ETH_RPC_URL；与 ETH_RPC_URL（银行/Code Pulse/NFT 索引等）隔离。连通性见 GET /api/lending/chain-status。",
			},
		})
	}
}
