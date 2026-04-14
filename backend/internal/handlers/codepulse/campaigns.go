package codepulse

import (
	"net/http"
	"sort"
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
		ctx := c.Request.Context()
		page, pageSize, offset := parsePagination(c)

		hasDBOnlyFilter := c.Query("developer") != "" || c.Query("contributor") != ""
		if !hasDBOnlyFilter {
			if sgRows, ok := sgQueryAllCampaigns(ctx, h); ok {
				filtered := sgFilterCampaigns(sgRows, c)
				total := int64(len(filtered))

				switch c.DefaultQuery("sort", "launched_at_desc") {
				case "deadline_at_asc":
					sort.Slice(filtered, func(i, j int) bool {
						return filtered[i].DeadlineAt.Before(filtered[j].DeadlineAt)
					})
				case "amount_raised_desc":
					sort.Slice(filtered, func(i, j int) bool {
						return filtered[i].AmountRaisedWei > filtered[j].AmountRaisedWei
					})
				default:
					sort.Slice(filtered, func(i, j int) bool {
						return filtered[i].LaunchedAt.After(filtered[j].LaunchedAt)
					})
				}

				end := offset + pageSize
				if end > len(filtered) {
					end = len(filtered)
				}
				paged := []models.CPCampaign{}
				if offset < len(filtered) {
					paged = filtered[offset:end]
				}

				c.JSON(http.StatusOK, gin.H{
					"campaigns":   paged,
					"pagination":  Pagination{Page: page, PageSize: pageSize, Total: total},
					"data_source": "subgraph",
				})
				return
			}
		}

		if !requireDB(h, c) {
			return
		}

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
			"campaigns":   rows,
			"pagination":  Pagination{Page: page, PageSize: pageSize, Total: total},
			"data_source": "database",
		})
	}
}

func sgFilterCampaigns(all []models.CPCampaign, c *gin.Context) []models.CPCampaign {
	state := c.Query("state")
	proposalID := c.Query("proposal_id")
	organizer := c.Query("organizer")

	if state == "" && proposalID == "" && organizer == "" {
		return all
	}

	var pidFilter uint64
	hasPidFilter := false
	if proposalID != "" {
		if pid, err := strconv.ParseUint(proposalID, 10, 64); err == nil {
			pidFilter = pid
			hasPidFilter = true
		}
	}
	orgNorm := ""
	if organizer != "" {
		orgNorm = normalizeAddress(organizer)
	}

	out := make([]models.CPCampaign, 0, len(all))
	for _, ca := range all {
		if state != "" && ca.State != state {
			continue
		}
		if hasPidFilter && ca.ProposalID != pidFilter {
			continue
		}
		if orgNorm != "" && normalizeAddress(ca.OrganizerAddress) != orgNorm {
			continue
		}
		out = append(out, ca)
	}
	return out
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
		cid, err := strconv.ParseUint(c.Param("campaignId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidCampaign})
			return
		}

		ctx := c.Request.Context()
		var ca *models.CPCampaign
		if sgCamps, ok := sgQueryAllCampaigns(ctx, h); ok {
			for i := range sgCamps {
				if sgCamps[i].CampaignID == cid {
					ca = &sgCamps[i]
					break
				}
			}
		}
		// 全量列表只含「最近 1000 次 launch」；活动较旧时不在列表里，但子图仍有 launch/donated，需按 campaignId 定向查，避免在 PG 扫链前误退回 database 全 0。
		if ca == nil && sgAvailable(h) {
			if one, ok := sgQuerySingleCampaignFromSubgraph(ctx, h, cid); ok {
				ca = one
			}
		}
		if ca != nil {
			resp := gin.H{
				"campaign":    *ca,
				"donor_count": ca.DonorCount,
				"data_source": "subgraph",
			}
			if h.DB != nil {
				var milestones []models.CPCampaignMilestone
				h.DB.Where(whereCampaignID, cid).Order("milestone_index").Find(&milestones)
				resp["milestones"] = milestones
				var developers []models.CPCampaignDeveloper
				h.DB.Where("campaign_id = ? AND is_active = true", cid).Find(&developers)
				resp["developers"] = developers
			}
			c.JSON(http.StatusOK, resp)
			return
		}

		if !requireDB(h, c) {
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
			"data_source": "database",
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
		ctx := c.Request.Context()
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

		sgRows, sgOK := sgFetchCampaignTimelineFromSubgraph(ctx, h, cid)
		var merged []models.CPEventLog
		if sgOK {
			merged = dedupEventLogsByTxLog(sgRows)
		} else if h.DB != nil {
			pgRows, err := pgFetchCampaignTimelineEvents(h.DB, cid, campaignEvents)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			merged = dedupEventLogsByTxLog(pgRows)
		}
		sortTimelineEventsDesc(merged)

		total := int64(len(merged))
		end := offset + pageSize
		if end > len(merged) {
			end = len(merged)
		}
		var pageRows []models.CPEventLog
		if offset < len(merged) {
			pageRows = merged[offset:end]
		}

		c.JSON(http.StatusOK, gin.H{
			"events":      pageRows,
			"pagination":  Pagination{Page: page, PageSize: pageSize, Total: total},
			"data_source": contributionsListDataSource(sgOK, len(merged)),
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
		ctx := c.Request.Context()
		cid, err := strconv.ParseUint(c.Param("campaignId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidCampaign})
			return
		}

		page, pageSize, offset := parsePagination(c)

		sgRows, sgOK := sgFetchDonationsForCampaign(ctx, h, cid)
		var merged []models.CampaignDonationEntry
		if sgOK {
			merged = dedupDonationsByTxLog(sgRows)
		} else if h.DB != nil {
			pgRows, err := pgFetchDonationsFromEventLog(h.DB, cid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			merged = dedupDonationsByTxLog(pgRows)
		}

		if v := c.Query("contributor"); v != "" {
			addr := normalizeAddress(v)
			var filtered []models.CampaignDonationEntry
			for _, e := range merged {
				if normalizeAddress(e.ContributorAddress) == addr {
					filtered = append(filtered, e)
				}
			}
			merged = filtered
		}

		sortCampaignDonations(merged, c.DefaultQuery("sort", "amount_desc"))

		total := int64(len(merged))
		end := offset + pageSize
		if end > len(merged) {
			end = len(merged)
		}
		var pageRows []models.CampaignDonationEntry
		if offset < len(merged) {
			pageRows = merged[offset:end]
		}

		c.JSON(http.StatusOK, gin.H{
			"contributions": pageRows,
			"pagination":    Pagination{Page: page, PageSize: pageSize, Total: total},
			"data_source":   contributionsListDataSource(sgOK, len(merged)),
		})
	}
}
