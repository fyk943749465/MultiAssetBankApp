package nft

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// Contracts GET /api/nft/contracts
func Contracts(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		var rows []models.NFTContract
		if err := h.DB.Where("chain_id = ?", chainID).Order("contract_kind, id").Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"chain_id":    chainID,
			"data_source": "database",
			"contracts":   rows,
		})
	}
}
