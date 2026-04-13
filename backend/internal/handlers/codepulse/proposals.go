package codepulse

import (
	"net/http"
	"strconv"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// Proposals 提案列表。
// @Summary      List proposals
// @Tags         code-pulse
// @Produce      json
// @Param        status        query string false "Filter by status"
// @Param        organizer     query string false "Filter by organizer address"
// @Param        review_state  query string false "Filter by round_review_state"
// @Param        page          query int    false "Page number (default 1)"
// @Param        page_size     query int    false "Page size (default 20, max 100)"
// @Param        sort          query string false "Sort: submitted_at_desc (default), submitted_at_asc"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/proposals [get]
func Proposals(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		page, pageSize, offset := parsePagination(c)
		q := h.DB.Model(&models.CPProposal{})

		if v := c.Query("status"); v != "" {
			q = q.Where("status = ?", v)
		}
		if v := c.Query("organizer"); v != "" {
			q = q.Where("LOWER(organizer_address) = ?", normalizeAddress(v))
		}
		if v := c.Query("review_state"); v != "" {
			q = q.Where("round_review_state = ?", v)
		}
		if c.Query("has_pending_round") == "true" {
			q = q.Where("pending_round_target_wei IS NOT NULL")
		}

		var total int64
		q.Count(&total)

		switch c.DefaultQuery("sort", "submitted_at_desc") {
		case "submitted_at_asc":
			q = q.Order("submitted_at ASC NULLS LAST")
		default:
			q = q.Order("submitted_at DESC NULLS LAST")
		}

		var rows []models.CPProposal
		if err := q.Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"proposals":  rows,
			"pagination": Pagination{Page: page, PageSize: pageSize, Total: total},
		})
	}
}

// ProposalDetail 提案详情。
// @Summary      Proposal detail
// @Tags         code-pulse
// @Produce      json
// @Param        proposalId path int true "Proposal ID"
// @Success      200 {object} map[string]any
// @Failure      404 {object} handlers.ErrorJSON
// @Router       /api/code-pulse/proposals/{proposalId} [get]
func ProposalDetail(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		pid, err := strconv.ParseUint(c.Param("proposalId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal_id"})
			return
		}

		var proposal models.CPProposal
		if err := h.DB.First(&proposal, whereProposalID, pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "proposal not found"})
			return
		}

		var milestones []models.CPProposalMilestone
		h.DB.Where(whereProposalID, pid).Order("round_ordinal, milestone_index").Find(&milestones)

		var campaigns []models.CPCampaign
		h.DB.Where(whereProposalID, pid).Order("round_index").Find(&campaigns)

		c.JSON(http.StatusOK, gin.H{
			"proposal":   proposal,
			"milestones": milestones,
			"campaigns":  campaigns,
		})
	}
}

// ProposalTimeline 提案时间线。
// @Summary      Proposal timeline
// @Tags         code-pulse
// @Produce      json
// @Param        proposalId path int true "Proposal ID"
// @Param        page       query int false "Page number"
// @Param        page_size  query int false "Page size"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/proposals/{proposalId}/timeline [get]
func ProposalTimeline(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		pid, err := strconv.ParseUint(c.Param("proposalId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal_id"})
			return
		}

		page, pageSize, offset := parsePagination(c)

		proposalEvents := []string{
			"ProposalSubmitted",
			"ProposalReviewed",
			"FundingRoundSubmittedForReview",
			"FundingRoundReviewed",
			"CrowdfundingLaunched",
		}

		q := h.DB.Model(&models.CPEventLog{}).
			Where("proposal_id = ? AND event_name IN ?", pid, proposalEvents)

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
