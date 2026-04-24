package lending

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// Supplies GET /api/lending/supplies
// @Summary      Supply 事件列表
// @Description  子图已配置且返回非空时优先子图；否则 PostgreSQL。权威事实以 RPC 落库为准。
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532"
// @Param        pool_address query string false "Pool 地址过滤"
// @Param        user_address query string false "用户地址过滤"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/supplies [get]
func Supplies(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID := resolveLendingChainID(c, h)
		page, pageSize := queryPage(c)
		offset := (page - 1) * pageSize

		if h.SubgraphLending != nil && h.SubgraphLending.Configured() {
			sgRows, errSG := fetchSubgraphSupplies(c.Request.Context(), h.SubgraphLending, pageSize, offset)
			if errSG == nil && len(sgRows) > 0 {
				c.JSON(http.StatusOK, gin.H{
					"chain_id":      chainID,
					"data_source":   "subgraph",
					"page":          page,
					"page_size":     pageSize,
					"total":         len(sgRows),
					"total_note":    "本页条数；子图未提供全量 total。",
					"supplies":      sgRows,
				})
				return
			}
			total, rows, errPG := fetchSuppliesPG(h, c, chainID, page, pageSize)
			if errPG != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": errPG.Error()})
				return
			}
			reason := "subgraph_empty"
			if errSG != nil {
				reason = errSG.Error()
			}
			c.JSON(http.StatusOK, gin.H{
				"chain_id":                chainID,
				"data_source":             "database",
				"subgraph_fallback_reason": reason,
				"page":                    page,
				"page_size":               pageSize,
				"total":                   total,
				"supplies":                rows,
			})
			return
		}

		total, rows, err := fetchSuppliesPG(h, c, chainID, page, pageSize)
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
			"supplies":    rows,
		})
	}
}

func fetchSuppliesPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, []models.LendingSupply, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingSupply{}).Where("chain_id = ?", chainID)
	q = applyPoolAddressFilter(c, q)
	q = applyUserOrParticipantFilter(c, q, "lending_supplies")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingSupply{}).Where("chain_id = ?", chainID)
	q2 = applyPoolAddressFilter(c, q2)
	q2 = applyUserOrParticipantFilter(c, q2, "lending_supplies")
	var rows []models.LendingSupply
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}
