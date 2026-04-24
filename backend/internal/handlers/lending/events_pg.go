package lending

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// Withdrawals GET /api/lending/withdrawals
// @Summary      Withdraw 事件列表
// @Description  PostgreSQL（RPC 扫块落库为权威）。支持 pool_address、user_address、分页。
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        pool_address query string false "Pool 地址过滤"
// @Param        user_address query string false "用户地址过滤"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/withdrawals [get]
func Withdrawals(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "withdrawals", fetchWithdrawalsPG)
}

// Borrows GET /api/lending/borrows
// @Summary      Borrow 事件列表
// @Description  PostgreSQL（RPC 扫块落库为权威）。
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        pool_address query string false "Pool 地址过滤"
// @Param        user_address query string false "用户地址过滤"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/borrows [get]
func Borrows(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "borrows", fetchBorrowsPG)
}

// Repays GET /api/lending/repays
// @Summary      Repay 事件列表
// @Description  PostgreSQL（RPC 扫块落库为权威）。
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        pool_address query string false "Pool 地址过滤"
// @Param        user_address query string false "用户地址过滤"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/repays [get]
func Repays(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "repays", fetchRepaysPG)
}

// Liquidations GET /api/lending/liquidations
// @Summary      Liquidation 事件列表
// @Description  PostgreSQL（RPC 扫块落库为权威）。user_address 匹配 borrower 或 liquidator。
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        pool_address query string false "Pool 地址过滤"
// @Param        user_address query string false "借款人或清算人地址过滤"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/liquidations [get]
func Liquidations(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "liquidations", fetchLiquidationsPG)
}

type pgLister func(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error)

func listPGOnly(h *handlers.Handlers, key string, list pgLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID := resolveLendingChainID(c, h)
		page, pageSize := queryPage(c)
		total, rows, err := list(h, c, chainID, page, pageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"chain_id":    chainID,
			"data_source": "database",
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			key:           rows,
		})
	}
}

func fetchWithdrawalsPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingWithdrawal{}).Where("chain_id = ?", chainID)
	q = applyPoolAddressFilter(c, q)
	q = applyUserOrParticipantFilter(c, q, "lending_withdrawals")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingWithdrawal{}).Where("chain_id = ?", chainID)
	q2 = applyPoolAddressFilter(c, q2)
	q2 = applyUserOrParticipantFilter(c, q2, "lending_withdrawals")
	var rows []models.LendingWithdrawal
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchBorrowsPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingBorrow{}).Where("chain_id = ?", chainID)
	q = applyPoolAddressFilter(c, q)
	q = applyUserOrParticipantFilter(c, q, "lending_borrows")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingBorrow{}).Where("chain_id = ?", chainID)
	q2 = applyPoolAddressFilter(c, q2)
	q2 = applyUserOrParticipantFilter(c, q2, "lending_borrows")
	var rows []models.LendingBorrow
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchRepaysPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingRepay{}).Where("chain_id = ?", chainID)
	q = applyPoolAddressFilter(c, q)
	q = applyUserOrParticipantFilter(c, q, "lending_repays")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingRepay{}).Where("chain_id = ?", chainID)
	q2 = applyPoolAddressFilter(c, q2)
	q2 = applyUserOrParticipantFilter(c, q2, "lending_repays")
	var rows []models.LendingRepay
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchLiquidationsPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingLiquidation{}).Where("chain_id = ?", chainID)
	q = applyPoolAddressFilter(c, q)
	q = applyUserOrParticipantFilter(c, q, "lending_liquidations")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingLiquidation{}).Where("chain_id = ?", chainID)
	q2 = applyPoolAddressFilter(c, q2)
	q2 = applyUserOrParticipantFilter(c, q2, "lending_liquidations")
	var rows []models.LendingLiquidation
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}
