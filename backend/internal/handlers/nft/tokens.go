package nft

import (
	"net/http"
	"strconv"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

type tokenRow struct {
	models.NFTToken
	OwnerAddress string `json:"owner_address"`
}

// CollectionTokens GET /api/nft/collections/:id/tokens
func CollectionTokens(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		collectionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || collectionID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid collection id"})
			return
		}
		var n int64
		if err := h.DB.Model(&models.NFTCollection{}).Where("chain_id = ? AND id = ?", chainID, collectionID).Count(&n).Error; err != nil || n == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
			return
		}
		page, pageSize := queryPage(c)
		offset := (page - 1) * pageSize

		var total int64
		if err := h.DB.Model(&models.NFTToken{}).Where("collection_id = ?", collectionID).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var rows []tokenRow
		if err := h.DB.Table("nft_tokens AS t").
			Select("t.*, lower(na.address) AS owner_address").
			Joins("JOIN nft_accounts na ON na.id = t.owner_account_id").
			Where("t.collection_id = ?", collectionID).
			Order("t.token_id ASC").
			Limit(pageSize).
			Offset(offset).
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data_source":   "database",
			"collection_id": collectionID,
			"page":          page,
			"page_size":     pageSize,
			"total":         total,
			"tokens":        rows,
		})
	}
}
