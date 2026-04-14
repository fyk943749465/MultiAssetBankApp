package codepulse

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// AdminDashboard 管理员工作台。
// @Summary      Admin dashboard
// @Tags         code-pulse
// @Produce      json
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/admin/dashboard [get]
func AdminDashboard(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		sgPending, sgRounds, sgLive, sgOK := sgQueryAdminDashboard(ctx, h)

		if !requireDB(h, c) {
			if sgOK {
				initAddrs, initSrc := proposalInitiatorAllowlist(ctx, h)
				var ini any
				if initSrc == "subgraph" {
					ini = initiatorAllowlistAsDashboardRows(initAddrs, "subgraph")
				} else {
					ini = []any{}
				}
				c.JSON(http.StatusOK, gin.H{
					"pending_proposals":    emptyIfNilP(sgPending),
					"pending_rounds":       emptyIfNilP(sgRounds),
					"live_campaigns":       emptyIfNilC(sgLive),
					"pending_milestones":   []any{},
					"initiators":           ini,
					"platform_donations":   "0",
					"platform_withdrawals": "0",
					"data_source":          "subgraph",
				})
				return
			}
			return
		}

		type milestoneRow struct {
			models.CPCampaignMilestone
			GithubURL string `json:"github_url"`
		}
		var pendingMilestones []milestoneRow
		h.DB.Model(&models.CPCampaignMilestone{}).
			Select("cp_campaign_milestones.*, cp_campaigns.github_url").
			Joins("JOIN cp_campaigns ON cp_campaigns.campaign_id = cp_campaign_milestones.campaign_id").
			Where("cp_campaign_milestones.approved = false AND cp_campaigns.state IN ?",
				[]string{"successful", "milestone_in_progress"}).
			Order("cp_campaign_milestones.campaign_id, cp_campaign_milestones.milestone_index").
			Limit(50).Find(&pendingMilestones)

		initAddrs, initSrc := proposalInitiatorAllowlist(ctx, h)
		var initiators any
		if initSrc == "subgraph" {
			initiators = initiatorAllowlistAsDashboardRows(initAddrs, "subgraph")
		} else {
			initiators = proposalInitiatorAllowlistFromDB(h.DB)
		}

		type sumResult struct {
			Total string
		}
		var platformDonations sumResult
		h.DB.Model(&models.CPPlatformFundMovement{}).
			Select("COALESCE(SUM(amount_wei),0) as total").
			Where("direction = ?", "donation").Scan(&platformDonations)

		var platformWithdrawals sumResult
		h.DB.Model(&models.CPPlatformFundMovement{}).
			Select("COALESCE(SUM(amount_wei),0) as total").
			Where("direction = ?", "withdrawal").Scan(&platformWithdrawals)

		if sgOK {
			c.JSON(http.StatusOK, gin.H{
				"pending_proposals":    emptyIfNilP(sgPending),
				"pending_rounds":       emptyIfNilP(sgRounds),
				"live_campaigns":       emptyIfNilC(sgLive),
				"pending_milestones":   pendingMilestones,
				"initiators":           initiators,
				"platform_donations":   platformDonations.Total,
				"platform_withdrawals": platformWithdrawals.Total,
				"data_source":          "subgraph",
			})
			return
		}

		initiators = proposalInitiatorAllowlistFromDB(h.DB)

		var pendingProposals []models.CPProposal
		h.DB.Where("status = ?", "pending_review").
			Order("submitted_at DESC NULLS LAST").Limit(50).Find(&pendingProposals)

		var pendingRounds []models.CPProposal
		h.DB.Where("round_review_state = ?", "round_review_pending").
			Order("updated_at DESC").Limit(50).Find(&pendingRounds)

		var liveCampaigns []models.CPCampaign
		h.DB.Where("state = ?", "fundraising").
			Order("deadline_at ASC").Limit(50).Find(&liveCampaigns)

		c.JSON(http.StatusOK, gin.H{
			"pending_proposals":    pendingProposals,
			"pending_rounds":       pendingRounds,
			"live_campaigns":       liveCampaigns,
			"pending_milestones":   pendingMilestones,
			"initiators":           initiators,
			"platform_donations":   platformDonations.Total,
			"platform_withdrawals": platformWithdrawals.Total,
			"data_source":          "database",
		})
	}
}

// InitiatorDashboard 提案发起人工作台。
// @Summary      Initiator dashboard
// @Tags         code-pulse
// @Produce      json
// @Param        address path string true "Wallet address"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/initiators/{address}/dashboard [get]
func InitiatorDashboard(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		addr := normalizeAddress(c.Param("address"))

		var myProposals []models.CPProposal
		h.DB.Where("LOWER(organizer_address) = ?", addr).
			Order("created_at DESC").Limit(100).Find(&myProposals)

		myProposals, sgNote := OrganizerProposalsSubgraphView(c.Request.Context(), h, addr, myProposals)

		pendingReview := filterProposals(myProposals, func(p models.CPProposal) bool { return p.Status == "pending_review" })
		approvedWaiting := filterProposals(myProposals, proposalAwaitingFirstRoundSubmit)
		roundPending := filterProposals(myProposals, func(p models.CPProposal) bool {
			return p.RoundReviewState != nil && *p.RoundReviewState == "round_review_pending"
		})
		roundApproved := filterProposals(myProposals, func(p models.CPProposal) bool {
			return p.RoundReviewState != nil && *p.RoundReviewState == "round_review_approved"
		})
		rejected := filterProposals(myProposals, func(p models.CPProposal) bool {
			return p.Status == "rejected" || p.Status == "round_review_rejected"
		})
		settled := filterProposals(myProposals, func(p models.CPProposal) bool { return p.Status == "settled" })

		var myCampaigns []models.CPCampaign
		h.DB.Where("LOWER(organizer_address) = ?", addr).
			Order("launched_at DESC").Limit(100).Find(&myCampaigns)

		fundraising, campNote := OrganizerFundraisingCampaignsSubgraphView(c.Request.Context(), h, addr, myCampaigns)

		resp := gin.H{
			"proposals_total":         len(myProposals),
			"pending_review":          pendingReview,
			"approved_waiting":        approvedWaiting,
			"round_review_pending":    roundPending,
			"round_review_approved":   roundApproved,
			"rejected":                rejected,
			"settled_can_follow_on":   settled,
			"fundraising_campaigns":   fundraising,
			"campaigns_total":         len(myCampaigns),
		}
		if sgNote != "" || campNote != "" {
			resp["view_data_source"] = "subgraph"
		}
		c.JSON(http.StatusOK, resp)
	}
}

// ContributorDashboard 捐款人工作台。
// @Summary      Contributor dashboard
// @Tags         code-pulse
// @Produce      json
// @Param        address path string true "Wallet address"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/contributors/{address}/dashboard [get]
func ContributorDashboard(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		addr := normalizeAddress(c.Param("address"))
		ctx := c.Request.Context()

		if sv := sgQueryContributorDashboard(ctx, h, addr); sv.OK {
			c.JSON(http.StatusOK, gin.H{
				"contributions_total": len(sv.All),
				"total_donated_wei":   sv.TotalDonated.String(),
				"all":                 sv.All,
				"refundable":          sv.Refundable,
				"fundraising":         sv.Fundraising,
				"successful":          sv.Successful,
				"data_source":         "subgraph",
			})
			return
		}

		if !requireDB(h, c) {
			return
		}

		var contributions []models.CPContribution
		h.DB.Where("LOWER(contributor_address) = ?", addr).
			Order("updated_at DESC").Find(&contributions)

		campaignIDs := make([]uint64, 0, len(contributions))
		for _, co := range contributions {
			campaignIDs = append(campaignIDs, co.CampaignID)
		}

		var campaigns []models.CPCampaign
		if len(campaignIDs) > 0 {
			h.DB.Where("campaign_id IN ?", campaignIDs).Find(&campaigns)
		}

		campaignMap := make(map[uint64]models.CPCampaign, len(campaigns))
		for _, ca := range campaigns {
			campaignMap[ca.CampaignID] = ca
		}

		type enrichedContribution struct {
			models.CPContribution
			CampaignState string `json:"campaign_state"`
			GithubURL     string `json:"github_url"`
		}

		refundable := []enrichedContribution{}
		fundraisingList := []enrichedContribution{}
		successfulList := []enrichedContribution{}
		all := make([]enrichedContribution, 0, len(contributions))

		for _, co := range contributions {
			ca := campaignMap[co.CampaignID]
			ec := enrichedContribution{
				CPContribution: co,
				CampaignState:  ca.State,
				GithubURL:      ca.GithubURL,
			}
			all = append(all, ec)
			switch ca.State {
			case "failed_refundable":
				refundable = append(refundable, ec)
			case "fundraising":
				fundraisingList = append(fundraisingList, ec)
			case "successful", "milestone_in_progress", "completed":
				successfulList = append(successfulList, ec)
			}
		}

		type sumResult struct {
			Total string
		}
		var totalDonated sumResult
		h.DB.Model(&models.CPContribution{}).
			Select("COALESCE(SUM(total_contributed_wei),0) as total").
			Where("LOWER(contributor_address) = ?", addr).Scan(&totalDonated)

		c.JSON(http.StatusOK, gin.H{
			"contributions_total": len(contributions),
			"total_donated_wei":   totalDonated.Total,
			"all":                 all,
			"refundable":          refundable,
			"fundraising":         fundraisingList,
			"successful":          successfulList,
			"data_source":         "database",
		})
	}
}

// DeveloperDashboard 开发者工作台。
// @Summary      Developer dashboard
// @Tags         code-pulse
// @Produce      json
// @Param        address path string true "Wallet address"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/developers/{address}/dashboard [get]
func DeveloperDashboard(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		addr := normalizeAddress(c.Param("address"))
		ctx := c.Request.Context()

		if sv := sgQueryDeveloperDashboard(ctx, h, addr); sv.OK {
			c.JSON(http.StatusOK, gin.H{
				"campaigns":          sv.Campaigns,
				"claims":             sv.Claims,
				"total_claimed_wei":  sv.TotalClaimedWei,
				"pending_milestones": sv.PendingMilestones,
				"data_source":        "subgraph",
			})
			return
		}

		if !requireDB(h, c) {
			return
		}

		var devEntries []models.CPCampaignDeveloper
		h.DB.Where("LOWER(developer_address) = ? AND is_active = true", addr).Find(&devEntries)

		campaignIDs := make([]uint64, 0, len(devEntries))
		for _, d := range devEntries {
			campaignIDs = append(campaignIDs, d.CampaignID)
		}

		var campaigns []models.CPCampaign
		if len(campaignIDs) > 0 {
			h.DB.Where("campaign_id IN ?", campaignIDs).Find(&campaigns)
		}

		var claims []models.CPMilestoneClaim
		h.DB.Where("LOWER(developer_address) = ?", addr).
			Order("claimed_at DESC").Find(&claims)

		type sumResult struct {
			Total string
		}
		var totalClaimed sumResult
		h.DB.Model(&models.CPMilestoneClaim{}).
			Select("COALESCE(SUM(claimed_amount_wei),0) as total").
			Where("LOWER(developer_address) = ?", addr).Scan(&totalClaimed)

		var pendingMilestones []models.CPCampaignMilestone
		if len(campaignIDs) > 0 {
			h.DB.Where("campaign_id IN ? AND approved = false", campaignIDs).
				Order("campaign_id, milestone_index").Find(&pendingMilestones)
		}

		c.JSON(http.StatusOK, gin.H{
			"campaigns":          campaigns,
			"claims":             claims,
			"total_claimed_wei":  totalClaimed.Total,
			"pending_milestones": pendingMilestones,
			"data_source":        "database",
		})
	}
}

func filterProposals(all []models.CPProposal, pred func(models.CPProposal) bool) []models.CPProposal {
	out := []models.CPProposal{}
	for _, p := range all {
		if pred(p) {
			out = append(out, p)
		}
	}
	return out
}

func filterCampaigns(all []models.CPCampaign, pred func(models.CPCampaign) bool) []models.CPCampaign {
	out := []models.CPCampaign{}
	for _, ca := range all {
		if pred(ca) {
			out = append(out, ca)
		}
	}
	return out
}

func emptyIfNilP(s []models.CPProposal) []models.CPProposal {
	if s == nil {
		return []models.CPProposal{}
	}
	return s
}

func emptyIfNilC(s []models.CPCampaign) []models.CPCampaign {
	if s == nil {
		return []models.CPCampaign{}
	}
	return s
}
