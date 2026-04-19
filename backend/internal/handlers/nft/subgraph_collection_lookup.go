package nft

import (
	"net/http"
	"strings"

	"go-chain/backend/internal/handlers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

// SubgraphCollectionByAddress GET /api/nft/subgraph/collection?address=0x...
// 用于链上已 deployProxy 但列表里看不到时，直接按「新合集合约地址」查子图是否已索引 CollectionCreated。
func SubgraphCollectionByAddress(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Query("address"))
		if raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing query: address (new collection contract, 0x + 40 hex)"})
			return
		}
		if !common.IsHexAddress(raw) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address"})
			return
		}
		if h.SubgraphNft == nil || !h.SubgraphNft.Configured() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "SUBGRAPH_NFT_URL not configured",
				"hint":  "后端配置子图后，列表才会优先子图；本接口也依赖该 URL。",
			})
			return
		}
		rows, err := fetchSubgraphCollectionsByContractAddress(c.Request.Context(), h.SubgraphNft, raw)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		if len(rows) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "subgraph has no CollectionCreated for this address",
				"hint": "常见原因：① 子图尚未同步到该块（Studio 看同步进度）；② SUBGRAPH_NFT_URL 指向的不是监听该工厂的网络/版本；③ 交易不在子图 data source 的工厂上。请在区块浏览器用「Internal Tx / Logs」确认合集合约地址与交易哈希。",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data_source": "subgraph",
			"address":     common.HexToAddress(raw).Hex(),
			"matches":     rows,
		})
	}
}
