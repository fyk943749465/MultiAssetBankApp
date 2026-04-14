package codepulse

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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
// @Param        review_state         query string false "Filter by round_review_state"
// @Param        waiting_launch_queue query string false "If true, only proposals waiting to enter fundraising (approved + round_review_approved, or approved with no round state and never launched)"
// @Param        page          query int    false "Page number (default 1)"
// @Param        page_size     query int    false "Page size (default 20, max 100)"
// @Param        sort          query string false "Sort: submitted_at_desc (default), submitted_at_asc"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/proposals [get]
func Proposals(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		page, pageSize, offset := parsePagination(c)

		if sgRows, ok := sgQueryAllProposals(ctx, h); ok {
			filtered := sgFilterProposals(sgRows, c)
			total := int64(len(filtered))

			switch c.DefaultQuery("sort", "submitted_at_desc") {
			case "submitted_at_asc":
				sort.Slice(filtered, func(i, j int) bool {
					return proposalSubmitTime(filtered[i]).Before(proposalSubmitTime(filtered[j]))
				})
			default:
				sort.Slice(filtered, func(i, j int) bool {
					return proposalSubmitTime(filtered[i]).After(proposalSubmitTime(filtered[j]))
				})
			}

			end := offset + pageSize
			if end > len(filtered) {
				end = len(filtered)
			}
			paged := []models.CPProposal{}
			if offset < len(filtered) {
				paged = filtered[offset:end]
			}

			c.JSON(http.StatusOK, gin.H{
				"proposals":   paged,
				"pagination":  Pagination{Page: page, PageSize: pageSize, Total: total},
				"data_source": "subgraph",
			})
			return
		}

		if !requireDB(h, c) {
			return
		}

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
		if c.Query("waiting_launch_queue") == "true" {
			q = q.Where(`status = 'approved' AND (
				round_review_state = 'round_review_approved'
				OR (
					(round_review_state IS NULL OR round_review_state = '')
					AND last_campaign_id IS NULL
				)
			)`)
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
			"proposals":   rows,
			"pagination":  Pagination{Page: page, PageSize: pageSize, Total: total},
			"data_source": "database",
		})
	}
}

// proposalAwaitingFirstRoundSubmit 发起人工作台「已通过待发起」：提案已通过且尚未进入轮次审核流、且从未 launch 过（launch 后轮次态会清空，用 last_campaign_id 区分）。
func proposalAwaitingFirstRoundSubmit(p models.CPProposal) bool {
	if p.Status != "approved" {
		return false
	}
	if p.RoundReviewState != nil && strings.TrimSpace(*p.RoundReviewState) != "" {
		return false
	}
	if p.LastCampaignID != nil && *p.LastCampaignID != 0 {
		return false
	}
	return true
}

// proposalInLaunchQueue 首页 Launch Queue：待提交首轮众筹审核，或本轮已通过审核等待 launch（不含已 launch 后轮次态清空的情况）。
func proposalInLaunchQueue(p models.CPProposal) bool {
	if proposalAwaitingFirstRoundSubmit(p) {
		return true
	}
	if p.Status != "approved" {
		return false
	}
	if p.RoundReviewState != nil && strings.TrimSpace(*p.RoundReviewState) == "round_review_approved" {
		return true
	}
	return false
}

func sgFilterProposals(all []models.CPProposal, c *gin.Context) []models.CPProposal {
	status := c.Query("status")
	organizer := c.Query("organizer")
	reviewState := c.Query("review_state")
	waitLaunch := c.Query("waiting_launch_queue") == "true"

	if status == "" && organizer == "" && reviewState == "" && !waitLaunch {
		return all
	}

	orgNorm := ""
	if organizer != "" {
		orgNorm = normalizeAddress(organizer)
	}

	out := make([]models.CPProposal, 0, len(all))
	for _, p := range all {
		if status != "" && p.Status != status {
			continue
		}
		if orgNorm != "" && normalizeAddress(p.OrganizerAddress) != orgNorm {
			continue
		}
		if reviewState != "" {
			if p.RoundReviewState == nil || *p.RoundReviewState != reviewState {
				continue
			}
		}
		if waitLaunch && !proposalInLaunchQueue(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func proposalSubmitTime(p models.CPProposal) time.Time {
	if p.SubmittedAt != nil {
		return *p.SubmittedAt
	}
	return p.CreatedAt
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
		pid, err := strconv.ParseUint(c.Param("proposalId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal_id"})
			return
		}

		ctx := c.Request.Context()
		if sgProposals, ok := sgQueryAllProposals(ctx, h); ok {
			for _, p := range sgProposals {
				if p.ProposalID != pid {
					continue
				}
				resp := gin.H{
					"proposal":    p,
					"data_source": "subgraph",
				}
				if h.DB != nil {
					var milestones []models.CPProposalMilestone
					h.DB.Where(whereProposalID, pid).Order("round_ordinal, milestone_index").Find(&milestones)
					resp["milestones"] = milestones
					var campaigns []models.CPCampaign
					h.DB.Where(whereProposalID, pid).Order("round_index").Find(&campaigns)
					resp["campaigns"] = campaigns
				}
				c.JSON(http.StatusOK, resp)
				return
			}
		}

		if !requireDB(h, c) {
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
			"proposal":    proposal,
			"milestones":  milestones,
			"campaigns":   campaigns,
			"data_source": "database",
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
