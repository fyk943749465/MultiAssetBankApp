package codepulse

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// ListInitiators 查看 proposal initiator 白名单。
// @Summary      List proposal initiators
// @Tags         code-pulse-admin
// @Produce      json
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/admin/proposal-initiators [get]
func ListInitiators(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		var roles []models.CPWalletRole
		h.DB.Where("role = ? AND scope_type = ? AND active = true", "proposal_initiator", "global").
			Order("created_at DESC").Find(&roles)

		addrs := make([]string, 0, len(roles))
		for _, r := range roles {
			addrs = append(addrs, r.WalletAddress)
		}

		c.JSON(http.StatusOK, gin.H{"initiators": addrs, "total": len(addrs)})
	}
}

// AddInitiator 添加 proposal initiator（记录到 DB 角色表）。
// @Summary      Add proposal initiator
// @Tags         code-pulse-admin
// @Accept       json
// @Produce      json
// @Param        body body object true "{ address: string }"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/admin/proposal-initiators [post]
func AddInitiator(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		var req struct {
			Address string `json:"address" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "address is required"})
			return
		}

		addr := normalizeAddress(req.Address)

		role := models.CPWalletRole{
			WalletAddress: addr,
			Role:          "proposal_initiator",
			ScopeType:     "global",
			Active:        true,
			DerivedFrom:   "admin_api",
		}
		if err := h.DB.Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ?",
			addr, "proposal_initiator", "global").
			FirstOrCreate(&role).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !role.Active {
			h.DB.Model(&role).Update("active", true)
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "address": addr})
	}
}

// RemoveInitiator 移除 proposal initiator。
// @Summary      Remove proposal initiator
// @Tags         code-pulse-admin
// @Produce      json
// @Param        address path string true "Wallet address"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/admin/proposal-initiators/{address} [delete]
func RemoveInitiator(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		addr := normalizeAddress(c.Param("address"))

		result := h.DB.Model(&models.CPWalletRole{}).
			Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ?",
				addr, "proposal_initiator", "global").
			Update("active", false)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "address": addr})
	}
}

// PlatformFunds 平台资金概览。
// @Summary      Platform funds overview
// @Tags         code-pulse-admin
// @Produce      json
// @Param        page      query int false "Page number"
// @Param        page_size query int false "Page size"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/admin/platform-funds [get]
func PlatformFunds(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		page, pageSize, offset := parsePagination(c)

		type sumResult struct {
			Total string
		}
		var donations sumResult
		h.DB.Model(&models.CPPlatformFundMovement{}).
			Select("COALESCE(SUM(amount_wei),0) as total").
			Where("direction = ?", "donation").Scan(&donations)

		var withdrawals sumResult
		h.DB.Model(&models.CPPlatformFundMovement{}).
			Select("COALESCE(SUM(amount_wei),0) as total").
			Where("direction = ?", "withdrawal").Scan(&withdrawals)

		q := h.DB.Model(&models.CPPlatformFundMovement{})
		var total int64
		q.Count(&total)

		var movements []models.CPPlatformFundMovement
		q.Order("block_number DESC, log_index DESC").
			Offset(offset).Limit(pageSize).Find(&movements)

		c.JSON(http.StatusOK, gin.H{
			"total_donations":   donations.Total,
			"total_withdrawals": withdrawals.Total,
			"movements":         movements,
			"pagination":        Pagination{Page: page, PageSize: pageSize, Total: total},
		})
	}
}

// SyncStatus 同步状态查询。
// @Summary      Sync status
// @Tags         code-pulse-admin
// @Produce      json
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/admin/sync-status [get]
func SyncStatus(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		var cursors []models.CPSyncCursor
		h.DB.Find(&cursors)

		var eventCount int64
		h.DB.Model(&models.CPEventLog{}).Count(&eventCount)

		c.JSON(http.StatusOK, gin.H{
			"cursors":     cursors,
			"event_count": eventCount,
		})
	}
}
