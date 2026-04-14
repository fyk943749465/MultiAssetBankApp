package codepulse

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

const (
	errMissingProposalID = "缺少 proposal_id"
	errMissingCampaignID = "缺少 campaign_id"
	reasonRoleMissing    = "role_missing"
	reasonStateInvalid   = "state_invalid"
)

var validActions = map[string]bool{
	"submit_proposal":                   true,
	"review_proposal":                   true,
	"submit_first_round_for_review":     true,
	"submit_follow_on_round_for_review": true,
	"review_funding_round":              true,
	"launch_approved_round":             true,
	"donate":                            true,
	"donate_to_platform":                true,
	"finalize_campaign":                 true,
	"claim_refund":                      true,
	"add_developer":                     true,
	"remove_developer":                  true,
	"approve_milestone":                 true,
	"claim_milestone_share":             true,
	"sweep_stale_funds":                 true,
	"set_proposal_initiator":            true,
	"withdraw_platform_funds":           true,
	"pause":                             true,
	"unpause":                           true,
	"transfer_ownership":                true,
	"renounce_ownership":                true,
}

var actionRoleMap = map[string]string{
	"submit_proposal":                   "proposal_initiator",
	"review_proposal":                   "admin",
	"submit_first_round_for_review":     "organizer",
	"submit_follow_on_round_for_review": "organizer",
	"review_funding_round":              "admin",
	"launch_approved_round":             "organizer",
	"donate":                            "",
	"donate_to_platform":                "",
	"finalize_campaign":                 "",
	"claim_refund":                      "donor",
	"add_developer":                     "organizer",
	"remove_developer":                  "organizer",
	"approve_milestone":                 "admin",
	"claim_milestone_share":             "developer",
	"sweep_stale_funds":                 "organizer",
	"set_proposal_initiator":            "admin",
	"withdraw_platform_funds":           "admin",
	"pause":                             "admin",
	"unpause":                           "admin",
	"transfer_ownership":                "admin",
	"renounce_ownership":                "admin",
}

// ActionCheckReq 动作预检请求体。
type ActionCheckReq struct {
	Action         string  `json:"action" binding:"required"`
	Wallet         string  `json:"wallet" binding:"required"`
	ProposalID     *uint64 `json:"proposal_id"`
	CampaignID     *uint64 `json:"campaign_id"`
	MilestoneIndex *int    `json:"milestone_index"`
	Params         any     `json:"params"`
}

// ActionCheckResp 动作预检响应。
type ActionCheckResp struct {
	Allowed         bool   `json:"allowed"`
	RequiredRole    string `json:"required_role"`
	CurrentState    string `json:"current_state,omitempty"`
	ReasonCode      string `json:"reason_code,omitempty"`
	ReasonMessage   string `json:"reason_message,omitempty"`
	RevertErrorName string `json:"revert_error_name,omitempty"`
	RevertErrorArgs any    `json:"revert_error_args,omitempty"`
}

// ActionCheck 动作预检。
// @Summary      Action pre-check
// @Tags         code-pulse
// @Accept       json
// @Produce      json
// @Param        body body ActionCheckReq true "Action check request"
// @Success      200 {object} ActionCheckResp
// @Router       /api/code-pulse/actions/check [post]
func ActionCheck(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		var req ActionCheckReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "action and wallet are required"})
			return
		}

		if !validActions[req.Action] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action: " + req.Action})
			return
		}

		resp := ActionCheckResp{RequiredRole: actionRoleMap[req.Action]}

		if requiredRole := actionRoleMap[req.Action]; requiredRole != "" {
			if !checkWalletRole(h, normalizeAddress(req.Wallet), requiredRole, req.ProposalID, req.CampaignID) {
				resp.ReasonCode = reasonRoleMissing
				resp.ReasonMessage = "当前钱包缺少所需角色: " + requiredRole
				c.JSON(http.StatusOK, resp)
				return
			}
		}

		stateOK, state, reason := checkActionState(h, req)
		resp.CurrentState = state
		if !stateOK {
			resp.ReasonCode = reasonStateInvalid
			resp.ReasonMessage = reason
			c.JSON(http.StatusOK, resp)
			return
		}

		resp.Allowed = true
		c.JSON(http.StatusOK, resp)
	}
}

func checkWalletRole(h *handlers.Handlers, addr, role string, proposalID, campaignID *uint64) bool {
	switch role {
	case "admin", "proposal_initiator":
		active, err := resolveGlobalRole(h, addr, role, true)
		return err == nil && active
	}

	var count int64
	h.DB.Table("cp_wallet_roles").
		Where("LOWER(wallet_address) = ? AND role = ? AND active = true", addr, role).
		Count(&count)
	if count > 0 {
		return true
	}

	switch role {
	case "organizer":
		return isOrganizerOf(h, addr, proposalID, campaignID)
	case "developer":
		return isDeveloperOf(h, addr, campaignID)
	case "donor":
		return isDonorOf(h, addr, campaignID)
	}
	return false
}

func isOrganizerOf(h *handlers.Handlers, addr string, proposalID, campaignID *uint64) bool {
	var count int64
	if proposalID != nil {
		h.DB.Table("cp_proposals").
			Where(whereProposalID+" AND LOWER(organizer_address) = ?", *proposalID, addr).
			Count(&count)
		if count > 0 {
			return true
		}
	}
	if campaignID != nil {
		h.DB.Table("cp_campaigns").
			Where(whereCampaignID+" AND LOWER(organizer_address) = ?", *campaignID, addr).
			Count(&count)
		if count > 0 {
			return true
		}
	}
	return false
}

func isDeveloperOf(h *handlers.Handlers, addr string, campaignID *uint64) bool {
	if campaignID == nil {
		return false
	}
	var count int64
	h.DB.Table("cp_campaign_developers").
		Where(whereCampaignID+" AND LOWER(developer_address) = ? AND is_active = true", *campaignID, addr).
		Count(&count)
	return count > 0
}

func isDonorOf(h *handlers.Handlers, addr string, campaignID *uint64) bool {
	if campaignID == nil {
		return false
	}
	var count int64
	h.DB.Table("cp_contributions").
		Where(whereCampaignID+" AND LOWER(contributor_address) = ?", *campaignID, addr).
		Count(&count)
	return count > 0
}

func checkActionState(h *handlers.Handlers, req ActionCheckReq) (ok bool, state, reason string) {
	switch req.Action {
	case "review_proposal":
		return checkProposalStatus(h, req.ProposalID, "pending_review", "提案不在待审核状态")
	case "submit_first_round_for_review":
		return checkProposalStatus(h, req.ProposalID, "approved", "提案尚未审核通过")
	case "submit_follow_on_round_for_review":
		return checkProposalStatus(h, req.ProposalID, "settled", "上一轮尚未结算")
	case "review_funding_round":
		return checkRoundReviewState(h, req.ProposalID, "round_review_pending", "没有待审核的 funding round")
	case "launch_approved_round":
		return checkRoundReviewState(h, req.ProposalID, "round_review_approved", "funding round 尚未审核通过")
	case "donate":
		return checkCampaignState(h, req.CampaignID, "fundraising", "当前不在众筹阶段")
	case "finalize_campaign":
		return checkCampaignState(h, req.CampaignID, "fundraising", "当前众筹不在可结算状态")
	case "claim_refund":
		return checkCampaignState(h, req.CampaignID, "failed_refundable", "当前项目不可退款")
	case "approve_milestone":
		return checkMilestoneNotApproved(h, req.CampaignID, req.MilestoneIndex)
	case "claim_milestone_share":
		return checkMilestoneApproved(h, req.CampaignID, req.MilestoneIndex)
	default:
		return true, "", ""
	}
}

func checkProposalStatus(h *handlers.Handlers, proposalID *uint64, expect, msg string) (bool, string, string) {
	if proposalID == nil {
		return false, "", errMissingProposalID
	}
	var status string
	h.DB.Model(&models.CPProposal{}).Select("status").
		Where(whereProposalID, *proposalID).Scan(&status)
	if status != expect {
		return false, status, msg
	}
	return true, status, ""
}

func checkRoundReviewState(h *handlers.Handlers, proposalID *uint64, expect, msg string) (bool, string, string) {
	if proposalID == nil {
		return false, "", errMissingProposalID
	}
	var state *string
	h.DB.Model(&models.CPProposal{}).Select("round_review_state").
		Where(whereProposalID, *proposalID).Scan(&state)
	if state == nil || *state != expect {
		s := ""
		if state != nil {
			s = *state
		}
		return false, s, msg
	}
	return true, *state, ""
}

func checkCampaignState(h *handlers.Handlers, campaignID *uint64, expect, msg string) (bool, string, string) {
	if campaignID == nil {
		return false, "", errMissingCampaignID
	}
	var state string
	h.DB.Model(&models.CPCampaign{}).Select("state").
		Where(whereCampaignID, *campaignID).Scan(&state)
	if state != expect {
		return false, state, msg
	}
	return true, state, ""
}

func checkMilestoneNotApproved(h *handlers.Handlers, campaignID *uint64, milestoneIndex *int) (bool, string, string) {
	if campaignID == nil || milestoneIndex == nil {
		return false, "", "缺少 campaign_id 或 milestone_index"
	}
	var approved bool
	h.DB.Model(&models.CPCampaignMilestone{}).Select("approved").
		Where("campaign_id = ? AND milestone_index = ?", *campaignID, *milestoneIndex).
		Scan(&approved)
	if approved {
		return false, "approved", "该里程碑已审批通过"
	}
	return true, "pending", ""
}

func checkMilestoneApproved(h *handlers.Handlers, campaignID *uint64, milestoneIndex *int) (bool, string, string) {
	if campaignID == nil || milestoneIndex == nil {
		return false, "", "缺少 campaign_id 或 milestone_index"
	}
	var approved bool
	h.DB.Model(&models.CPCampaignMilestone{}).Select("approved").
		Where("campaign_id = ? AND milestone_index = ?", *campaignID, *milestoneIndex).
		Scan(&approved)
	if !approved {
		return false, "not_approved", "该里程碑尚未审批通过"
	}
	return true, "approved", ""
}
