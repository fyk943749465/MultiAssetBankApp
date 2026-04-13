package bank

import (
	"net/http"
	"strconv"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// Deposits 查询已索引的充值记录（链上 Deposited 事件）。
// @Summary      Bank deposits (indexed DB)
// @Description  Lists `Deposited` events stored by the local bank indexer. Optional filter by user wallet.
// @Tags         bank
// @Produce      json
// @Param        limit query int false "Max rows (default 50, max 200)" default(50) minimum(1) maximum(200)
// @Param        user  query string false "Filter by user address (0x...), case-insensitive match"
// @Success      200 {object} handlers.BankDepositsResp
// @Failure      503 {object} handlers.ErrorJSON "database not configured"
// @Failure      500 {object} handlers.ErrorJSON
// @Router       /api/bank/deposits [get]
func Deposits(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		q := h.DB.Model(&models.BankDeposit{}).Order("block_number DESC, log_index DESC").Limit(limit)
		if u := strings.TrimSpace(c.Query("user")); u != "" {
			q = q.Where("LOWER(user_address) = ?", strings.ToLower(u))
		}
		var rows []models.BankDeposit
		if err := q.Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"deposits": rows})
	}
}

// Withdrawals 查询已索引的提现记录（链上 Withdrawn 事件）。
// @Summary      Bank withdrawals (indexed DB)
// @Description  Lists `Withdrawn` events stored by the local bank indexer. Optional filter by user wallet.
// @Tags         bank
// @Produce      json
// @Param        limit query int false "Max rows (default 50, max 200)" default(50) minimum(1) maximum(200)
// @Param        user  query string false "Filter by user address (0x...), case-insensitive match"
// @Success      200 {object} handlers.BankWithdrawalsResp
// @Failure      503 {object} handlers.ErrorJSON "database not configured"
// @Failure      500 {object} handlers.ErrorJSON
// @Router       /api/bank/withdrawals [get]
func Withdrawals(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		q := h.DB.Model(&models.BankWithdrawal{}).Order("block_number DESC, log_index DESC").Limit(limit)
		if u := strings.TrimSpace(c.Query("user")); u != "" {
			q = q.Where("LOWER(user_address) = ?", strings.ToLower(u))
		}
		var rows []models.BankWithdrawal
		if err := q.Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"withdrawals": rows})
	}
}
