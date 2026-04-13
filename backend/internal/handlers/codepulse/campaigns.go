package codepulse

import (
	"net/http"
	"strconv"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// Campaigns 众筹列表。
// @Summary      List campaigns
// @Tags         code-pulse
// @Produce      json
// @Param        state       query string false "Filter by state"
// @Param        proposal_id query int    false "Filter by proposal_id"
// @Param        organizer   query string false "Filter by organizer address"
// @Param        developer   query string false "Filter by developer (joined)"
// @Param        contributor query string false "Filter by contributor (joined)"
// @Param        page        query int    false "Page number"
// @Param        page_size   query int    false "Page size"
// @Param        sort        query string false "Sort: launched_at_desc (default), deadline_at_asc"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/campaigns [get]
func Campaigns(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		page, pageSize, offset := parsePagination(c)
		q := h.DB.Model(&models.CPCampaign{})

		if v := c.Query("state"); v != "" {
			q = q.Where("state = ?", v)
		}
		if v := c.Query("proposal_id"); v != "" {
			if pid, err := strconv.ParseUint(v, 10, 64); err == nil {
				q = q.Where("proposal_id = ?", pid)
			}
		}
		if v := c.Query("organizer"); v != "" {
			q = q.Where("LOWER(organizer_address) = ?", normalizeAddress(v))
		}
		if v := c.Query("developer"); v != "" {
			q = q.Where("campaign_id IN (?)",
				h.DB.Model(&models.CPCampaignDeveloper{}).
					Select("campaign_id").
					Where("LOWER(developer_address) = ? AND is_active = true", normalizeAddress(v)),
			)
		}
		if v := c.Query("contributor"); v != "" {
			q = q.Where("campaign_id IN (?)",
				h.DB.Model(&models.CPContribution{}).
					Select("campaign_id").
					Where("LOWER(contributor_address) = ?", normalizeAddress(v)),
			)
		}

		var total int64
		q.Count(&total)

		switch c.DefaultQuery("sort", "launched_at_desc") {
		case "deadline_at_asc":
			q = q.Order("deadline_at ASC")
		case "amount_raised_desc":
			q = q.Order("amount_raised_wei DESC")
		default:
			q = q.Order("launched_at DESC")
		}

		var rows []models.CPCampaign
		if err := q.Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"campaigns":  rows,
			"pagination": Pagination{Page: page, PageSize: pageSize, Total: total},
		})
	}
}

// CampaignDetail 众筹详情。
// @Summary      Campaign detail
// @Tags         code-pulse
// @Produce      json
// @Param        campaignId path int true "Campaign ID"
// @Success      200 {object} map[string]any
// @Failure      404 {object} handlers.ErrorJSON
// @Router       /api/code-pulse/campaigns/{campaignId} [get]
func CampaignDetail(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		cid, err := strconv.ParseUint(c.Param("campaignId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidCampaign})
			return
		}

		var campaign models.CPCampaign
		if err := h.DB.First(&campaign, whereCampaignID, cid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
			return
		}

		var milestones []models.CPCampaignMilestone
		h.DB.Where(whereCampaignID, cid).Order("milestone_index").Find(&milestones)

		var developers []models.CPCampaignDeveloper
		h.DB.Where("campaign_id = ? AND is_active = true", cid).Find(&developers)

		var donorCount int64
		h.DB.Model(&models.CPContribution{}).Where(whereCampaignID, cid).Count(&donorCount)

		c.JSON(http.StatusOK, gin.H{
			"campaign":    campaign,
			"milestones":  milestones,
			"developers":  developers,
			"donor_count": donorCount,
		})
	}
}

// CampaignTimeline 众筹时间线。
// @Summary      Campaign timeline
// @Tags         code-pulse
// @Produce      json
// @Param        campaignId path int true "Campaign ID"
// @Param        page       query int false "Page number"
// @Param        page_size  query int false "Page size"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/campaigns/{campaignId}/timeline [get]
func CampaignTimeline(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		cid, err := strconv.ParseUint(c.Param("campaignId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidCampaign})
			return
		}

		page, pageSize, offset := parsePagination(c)

		campaignEvents := []string{
			"CrowdfundingLaunched",
			"Donated",
			"CampaignFinalized",
			"RefundClaimed",
			"DeveloperAdded",
			"DeveloperRemoved",
			"MilestoneApproved",
			"MilestoneShareClaimed",
			"StaleFundsSwept",
		}

		q := h.DB.Model(&models.CPEventLog{}).
			Where("campaign_id = ? AND event_name IN ?", cid, campaignEvents)

		var total int64
		q.Count(&total)

		var events []models.CPEventLog
		if err := q.Order("block_number DESC, log_index DESC").
			Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events":     events,
			"pagination": Pagination{Page: page, PageSize: pageSize, Total: total},
		})
	}
}

// CampaignContributions 众筹贡献榜。
// @Summary      Campaign contributions
// @Tags         code-pulse
// @Produce      json
// @Param        campaignId  path  int    true  "Campaign ID"
// @Param        contributor query string false "Filter by contributor address"
// @Param        sort        query string false "Sort: amount_desc (default), latest"
// @Param        page        query int    false "Page number"
// @Param        page_size   query int    false "Page size"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/campaigns/{campaignId}/contributions [get]
func CampaignContributions(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		cid, err := strconv.ParseUint(c.Param("campaignId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidCampaign})
			return
		}

		page, pageSize, offset := parsePagination(c)
		q := h.DB.Model(&models.CPContribution{}).Where(whereCampaignID, cid)

		if v := c.Query("contributor"); v != "" {
			q = q.Where("LOWER(contributor_address) = ?", normalizeAddress(v))
		}

		var total int64
		q.Count(&total)

		switch c.DefaultQuery("sort", "amount_desc") {
		case "latest":
			q = q.Order("last_donated_at DESC NULLS LAST")
		default:
			q = q.Order("total_contributed_wei DESC")
		}

		var rows []models.CPContribution
		if err := q.Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"contributions": rows,
			"pagination":    Pagination{Page: page, PageSize: pageSize, Total: total},
		})
	}
}
