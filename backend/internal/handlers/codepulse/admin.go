package codepulse

import (
	"net/http"
	"strconv"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/indexer"
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

		addrs, source := proposalInitiatorAllowlist(c.Request.Context(), h)
		c.JSON(http.StatusOK, gin.H{
			"initiators":   addrs,
			"total":        len(addrs),
			"data_source":  source,
		})
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
// event_count：子图已配置且计数成功时为子图各类实体条数之和（与链上事件一致，通常含部署块）；否则为 cp_event_log 行数。
// event_count_database：始终为 PostgreSQL cp_event_log 行数，便于对照「索引库是否缺早期块」。
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

		var eventCountDB int64
		h.DB.Model(&models.CPEventLog{}).Count(&eventCountDB)

		out := gin.H{
			"cursors":              cursors,
			"event_count":          eventCountDB,
			"event_count_source":   "database",
			"event_count_database": eventCountDB,
		}

		if h.SubgraphCodePulse != nil && h.SubgraphCodePulse.Configured() {
			if n, err := indexer.CountCodePulseSubgraphEventEntities(c.Request.Context(), h.SubgraphCodePulse); err == nil {
				out["event_count"] = n
				out["event_count_source"] = "subgraph"
			}
		}

		if h.Chain != nil && h.Chain.Eth() != nil {
			if heads, err := indexer.FetchChainRPCHeads(c.Request.Context(), h.Chain.Eth()); err == nil {
				out["chain_heads"] = heads
			}
		}

		c.JSON(http.StatusOK, out)
	}
}

// adminSubgraphFeedFirstPerType 按分页估算每类实体拉取条数（上限 500），使合并后的列表尽量覆盖当前页。
func adminSubgraphFeedFirstPerType(offset, pageSize int) int {
	n := offset + pageSize + 64
	per := (n + 17) / 18
	if per < 150 {
		per = 150
	}
	if per > 500 {
		per = 500
	}
	return per
}

func adminEventLogListFromSubgraph(h *handlers.Handlers, c *gin.Context, page, pageSize, offset int) bool {
	if h.SubgraphCodePulse == nil || !h.SubgraphCodePulse.Configured() {
		return false
	}
	first := adminSubgraphFeedFirstPerType(offset, pageSize)
	merged, err := indexer.QueryCodePulseSubgraphAdminFeed(c.Request.Context(), h.SubgraphCodePulse, first)
	if err != nil {
		return false
	}

	eventName := c.Query("event_name")
	var wantPID *uint64
	if v := c.Query("proposal_id"); v != "" {
		if pid, e := strconv.ParseUint(v, 10, 64); e == nil {
			wantPID = &pid
		}
	}
	var wantCID *uint64
	if v := c.Query("campaign_id"); v != "" {
		if cid, e := strconv.ParseUint(v, 10, 64); e == nil {
			wantCID = &cid
		}
	}

	filtered := make([]indexer.NormalizedSubgraphAdminEvent, 0, len(merged))
	for _, ev := range merged {
		if eventName != "" && ev.EventName != eventName {
			continue
		}
		if wantPID != nil && (ev.ProposalID == nil || *ev.ProposalID != *wantPID) {
			continue
		}
		if wantCID != nil && (ev.CampaignID == nil || *ev.CampaignID != *wantCID) {
			continue
		}
		filtered = append(filtered, ev)
	}

	total := int64(len(filtered))
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	var rows []models.CPEventLog
	if offset < len(filtered) {
		chainID := uint64(0)
		if h.Chain != nil {
			if id, e := h.Chain.Eth().ChainID(c.Request.Context()); e == nil {
				chainID = id.Uint64()
			}
		}
		contract := ""
		if h.CodePulse != nil {
			contract = strings.ToLower(h.CodePulse.Address().Hex())
		}
		rows = make([]models.CPEventLog, 0, end-offset)
		for _, ev := range filtered[offset:end] {
			rows = append(rows, models.CPEventLog{
				ChainID:         chainID,
				ContractAddress: contract,
				EventName:       ev.EventName,
				ProposalID:      ev.ProposalID,
				CampaignID:      ev.CampaignID,
				WalletAddress:   ev.WalletAddress,
				TxHash:          ev.TxHash,
				LogIndex:        ev.LogIndex,
				BlockNumber:     ev.BlockNumber,
				BlockTimestamp:  ev.BlockTS.UTC(),
				Payload:         models.JSONB(ev.Payload),
				Source:          "subgraph",
				CreatedAt:       ev.BlockTS.UTC(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      rows,
		"pagination":  Pagination{Page: page, PageSize: pageSize, Total: total},
		"data_source": "subgraph",
	})
	return true
}

// AdminEventLogList 分页列出链上事件：默认优先读 Code Pulse 子图（与增量解析同源）；子图不可用或未配置时回退 cp_event_log。
// 子图模式下 total 为「当前查询窗口内（每类最多 N 条合并后）满足筛选条件的条数」，深分页可能不足一页。
// 查询参数：page、page_size、event_name、proposal_id、campaign_id（可选过滤）。
// @Summary      List indexed event log (admin)
// @Tags         code-pulse-admin
// @Produce      json
// @Router       /api/code-pulse/admin/events [get]
func AdminEventLogList(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, pageSize, offset := parsePagination(c)

		if adminEventLogListFromSubgraph(h, c, page, pageSize, offset) {
			return
		}

		if !requireDB(h, c) {
			return
		}

		q := h.DB.Model(&models.CPEventLog{})

		if v := c.Query("event_name"); v != "" {
			q = q.Where("event_name = ?", v)
		}
		if v := c.Query("proposal_id"); v != "" {
			if pid, err := strconv.ParseUint(v, 10, 64); err == nil {
				q = q.Where("proposal_id = ?", pid)
			}
		}
		if v := c.Query("campaign_id"); v != "" {
			if cid, err := strconv.ParseUint(v, 10, 64); err == nil {
				q = q.Where("campaign_id = ?", cid)
			}
		}

		var total int64
		q.Count(&total)

		var rows []models.CPEventLog
		if err := q.Order("block_number DESC, log_index DESC").
			Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events":      rows,
			"pagination":  Pagination{Page: page, PageSize: pageSize, Total: total},
			"data_source": "database",
		})
	}
}
