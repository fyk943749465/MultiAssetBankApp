package nft

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

type activeListingRow struct {
	models.NFTActiveListing
	SellerAddress string `json:"seller_address"`
}

// ActiveListings GET /api/nft/listings/active
// 读策略：子图已配置 → 始终先查子图（含空列表）；仅当子图请求/解析失败时用 PostgreSQL 兜底，并返回 subgraph_fallback_error。
func ActiveListings(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		page, pageSize := queryPage(c)
		offset := (page - 1) * pageSize

		if h.SubgraphNft != nil && h.SubgraphNft.Configured() {
			sgRows, hasMore, errSG := fetchSubgraphListingsFallback(c.Request.Context(), h.SubgraphNft, page, pageSize)
			if errSG == nil {
				c.JSON(http.StatusOK, gin.H{
					"chain_id":      chainID,
					"data_source":   "subgraph",
					"subgraph_note": subgraphReorgNote + " 以下为子图「上架」事件，不等价于当前仍有效的活跃挂单。",
					"page":          page,
					"page_size":     pageSize,
					"total":         int64(len(sgRows)),
					"total_note":    "本页条数；子图未提供全量 total。",
					"has_more":      hasMore,
					"listings":      sgRows,
				})
				return
			}

			var total int64
			if err := h.DB.Model(&models.NFTActiveListing{}).
				Where("chain_id = ? AND listing_status = ?", chainID, "active").
				Count(&total).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			var rows []activeListingRow
			if err := h.DB.Table("nft_active_listings AS l").
				Select("l.*, lower(na.address) AS seller_address").
				Joins("JOIN nft_accounts na ON na.id = l.seller_account_id").
				Where("l.chain_id = ? AND l.listing_status = ?", chainID, "active").
				Order("l.price_wei ASC, l.id ASC").
				Limit(pageSize).
				Offset(offset).
				Scan(&rows).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"chain_id":                chainID,
				"data_source":             "database",
				"subgraph_fallback_error": errSG.Error(),
				"page":                    page,
				"page_size":               pageSize,
				"total":                   total,
				"listings":                rows,
			})
			return
		}

		var total int64
		if err := h.DB.Model(&models.NFTActiveListing{}).
			Where("chain_id = ? AND listing_status = ?", chainID, "active").
			Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var rows []activeListingRow
		if err := h.DB.Table("nft_active_listings AS l").
			Select("l.*, lower(na.address) AS seller_address").
			Joins("JOIN nft_accounts na ON na.id = l.seller_account_id").
			Where("l.chain_id = ? AND l.listing_status = ?", chainID, "active").
			Order("l.price_wei ASC, l.id ASC").
			Limit(pageSize).
			Offset(offset).
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"chain_id":    chainID,
			"data_source": "database",
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"listings":    rows,
		})
	}
}
