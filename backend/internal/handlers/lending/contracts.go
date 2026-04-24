package lending

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// Contracts GET /api/lending/contracts
// @Summary      借贷已登记合约
// @Description  读取 lending_contracts（006 种子 + 索引器可补充）。
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "EVM chain id，默认 84532（Base Sepolia）"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/contracts [get]
func Contracts(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID := resolveLendingChainID(c, h)
		var rows []models.LendingContract
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
