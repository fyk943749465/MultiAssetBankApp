package codepulse

import (
	"math/big"
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

// Summary 返回首页总览数据。
// @Summary      Code Pulse summary
// @Tags         code-pulse
// @Produce      json
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/summary [get]
func Summary(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		if sg := sgQuerySummary(ctx, h); sg.OK {
			approvedWaiting := sg.Approved
			if rows, ok := sgQueryAllProposals(ctx, h); ok {
				var aw int64
				for _, p := range rows {
					if proposalInLaunchQueue(p) {
						aw++
					}
				}
				approvedWaiting = aw
			}
			c.JSON(http.StatusOK, gin.H{
				"proposal_total":     sg.ProposalTotal,
				"pending_review":     sg.PendingReview,
				"approved_waiting":   approvedWaiting,
				"campaign_total":     sg.CampaignTotal,
				"fundraising":        sg.Fundraising,
				"successful":         sg.Successful,
				"failed":             sg.Failed,
				"total_raised_wei":   sg.TotalRaisedWei.String(),
				"total_refunded_wei": sg.TotalRefundWei.String(),
				"data_source":        "subgraph",
			})
			return
		}

		if !requireDB(h, c) {
			return
		}

		var proposalTotal int64
		h.DB.Model(&models.CPProposal{}).Count(&proposalTotal)

		var pendingReview int64
		h.DB.Model(&models.CPProposal{}).Where("status = ?", "pending_review").Count(&pendingReview)

		var approvedWaiting int64
		h.DB.Model(&models.CPProposal{}).
			Where(`status = 'approved' AND (
				round_review_state = 'round_review_approved'
				OR (
					(round_review_state IS NULL OR round_review_state = '')
					AND last_campaign_id IS NULL
				)
			)`).
			Count(&approvedWaiting)

		var campaignTotal int64
		h.DB.Model(&models.CPCampaign{}).Count(&campaignTotal)

		var fundraising int64
		h.DB.Model(&models.CPCampaign{}).Where("state = ?", "fundraising").Count(&fundraising)

		var successful int64
		h.DB.Model(&models.CPCampaign{}).Where("state IN ?", []string{"successful", "milestone_in_progress", "completed"}).Count(&successful)

		var failed int64
		h.DB.Model(&models.CPCampaign{}).Where("state = ?", "failed_refundable").Count(&failed)

		type sumResult struct {
			Total string
		}
		var raised sumResult
		h.DB.Model(&models.CPCampaign{}).Select("COALESCE(SUM(amount_raised_wei),0) as total").Scan(&raised)

		var refunded sumResult
		h.DB.Model(&models.CPContribution{}).Select("COALESCE(SUM(refund_claimed_wei),0) as total").Scan(&refunded)

		c.JSON(http.StatusOK, gin.H{
			"proposal_total":     proposalTotal,
			"pending_review":     pendingReview,
			"approved_waiting":   approvedWaiting,
			"campaign_total":     campaignTotal,
			"fundraising":        fundraising,
			"successful":         successful,
			"failed":             failed,
			"total_raised_wei":   raised.Total,
			"total_refunded_wei": refunded.Total,
			"data_source":        "database",
		})
	}
}

// Config 返回链上规则常量与合约地址。
// @Summary      Code Pulse config
// @Tags         code-pulse
// @Produce      json
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/config [get]
func Config(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		contractAddr := ""
		ownerAddr := ""
		paused := false
		stateSource := ""
		var stateBlockNumber *uint64
		var stateSyncedAt any
		// Fallbacks
		milestoneNum := uint8(3)
		minCampaignTarget := "10000000000000000"
		minCampaignDuration := uint64(86400)
		maxDevelopers := uint64(20)
		maxGithubUrlLength := uint64(256)
		staleFundsSweepDelay := uint64(2592000)
		milestoneUnlockDelays := []int{0, 604800, 1209600}

		if h.CodePulse != nil {
			contractAddr = h.CodePulse.Address().Hex()
			if state, err := resolveSystemState(h, false); err == nil && state != nil {
				ownerAddr = state.OwnerAddress
				paused = state.Paused
				stateSource = state.Source
				stateBlockNumber = state.SourceBlockNumber
				stateSyncedAt = state.SyncedAt
			}

			// Dynamically fetch from contract
			ctx := c.Request.Context()
			if out, err := h.CodePulse.CallView(ctx, "MILESTONE_NUM"); err == nil && len(out) == 1 {
				if v, ok := out[0].(uint8); ok { milestoneNum = v }
			}
			if out, err := h.CodePulse.CallView(ctx, "MIN_CAMPAIGN_TARGET"); err == nil && len(out) == 1 {
				if v, ok := out[0].(*big.Int); ok { minCampaignTarget = v.String() }
			}
			if out, err := h.CodePulse.CallView(ctx, "MIN_CAMPAIGN_DURATION"); err == nil && len(out) == 1 {
				if v, ok := out[0].(*big.Int); ok { minCampaignDuration = v.Uint64() }
			}
			if out, err := h.CodePulse.CallView(ctx, "MAX_DEVELOPERS_PER_CAMPAIGN"); err == nil && len(out) == 1 {
				if v, ok := out[0].(*big.Int); ok { maxDevelopers = v.Uint64() }
			}
			if out, err := h.CodePulse.CallView(ctx, "MAX_GITHUB_URL_LENGTH"); err == nil && len(out) == 1 {
				if v, ok := out[0].(*big.Int); ok { maxGithubUrlLength = v.Uint64() }
			}
			if out, err := h.CodePulse.CallView(ctx, "STALE_FUNDS_SWEEP_DELAY"); err == nil && len(out) == 1 {
				if v, ok := out[0].(*big.Int); ok { staleFundsSweepDelay = v.Uint64() }
			}
			var delays []int
			for i := uint8(0); i < milestoneNum; i++ {
				if out, err := h.CodePulse.CallView(ctx, "milestoneUnlockDelay", i); err == nil && len(out) == 1 {
					if v, ok := out[0].(*big.Int); ok { delays = append(delays, int(v.Int64())) }
				}
			}
			if len(delays) == int(milestoneNum) { milestoneUnlockDelays = delays }
		}

		serverTx := h.CodePulseServerTx && h.TxKey != nil
		relayer := ""
		if serverTx {
			relayer = crypto.PubkeyToAddress(h.TxKey.PublicKey).Hex()
		}
		c.JSON(http.StatusOK, gin.H{
			"contract_address":               contractAddr,
			"contract_configured":            h.CodePulse != nil,
			"subgraph_configured":            h.SubgraphCodePulse != nil,
			"code_pulse_server_tx_enabled":   serverTx,
			"server_tx_relayer_address":      relayer,
			"owner_address":               ownerAddr,
			"paused":                      paused,
			"state_source":                stateSource,
			"state_source_block_number":   stateBlockNumber,
			"state_synced_at":             stateSyncedAt,
			"milestone_num":               milestoneNum,
			"min_campaign_target":         minCampaignTarget,
			"min_campaign_duration":       minCampaignDuration,
			"max_developers_per_campaign": maxDevelopers,
			"max_github_url_length":       maxGithubUrlLength,
			"stale_funds_sweep_delay":     staleFundsSweepDelay,
			"milestone_unlock_delays":     milestoneUnlockDelays,
		})
	}
}
