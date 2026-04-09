package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// BankDeposits 查询已索引的充值记录（链上 Deposited 事件）。
func (h *Handlers) BankDeposits(c *gin.Context) {
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

// BankWithdrawals 查询已索引的提现记录（链上 Withdrawn 事件）。
func (h *Handlers) BankWithdrawals(c *gin.Context) {
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
