package codepulse

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

const (
	errMissingProposalID        = "缺少 proposal_id"
	errMissingCampaignID        = "缺少 campaign_id"
	reasonRoleMissing           = "role_missing"
	reasonStateInvalid          = "state_invalid"
	reasonInitiatorRevokePolicy = "initiator_revoke_blocked"
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
	// AdvisoryCode / AdvisoryMessage 为不阻止交易的补充说明（如撤销 initiator 对已有 organizer 提案的影响）。
	AdvisoryCode    string `json:"advisory_code,omitempty"`
	AdvisoryMessage string `json:"advisory_message,omitempty"`
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

		allowed, cur, rc, rm := codePulseActionGate(h, req)
		resp.CurrentState = cur
		if !allowed {
			resp.ReasonCode = rc
			resp.ReasonMessage = rm
			c.JSON(http.StatusOK, resp)
			return
		}

		resp.Allowed = true
		if ac, am := setProposalInitiatorRevokeAdvisory(h, req); ac != "" {
			resp.AdvisoryCode = ac
			resp.AdvisoryMessage = am
		}
		c.JSON(http.StatusOK, resp)
	}
}

// codePulseActionGate 角色 + 读库状态预检（PostgreSQL 为准）；供 actions/check 与 tx/build、tx/submit 共用。
func codePulseActionGate(h *handlers.Handlers, req ActionCheckReq) (allowed bool, currentState, reasonCode, reasonMsg string) {
	if requiredRole := actionRoleMap[req.Action]; requiredRole != "" {
		if !checkWalletRole(h, normalizeAddress(req.Wallet), requiredRole, req.ProposalID, req.CampaignID) {
			return false, "", reasonRoleMissing, "当前钱包缺少所需角色: " + requiredRole
		}
	}
	if req.Action == "set_proposal_initiator" {
		if blocked, msg := setProposalInitiatorRevokeBlocked(h, req); blocked {
			return false, "", reasonInitiatorRevokePolicy, msg
		}
	}
	stateOK, state, reason := checkActionState(h, req)
	if !stateOK {
		msg := reason
		if stateInvalidNeedsSyncHint(req.Action, reason) {
			msg = reason + actionStateSyncHint(h)
		}
		return false, state, reasonStateInvalid, msg
	}
	return true, state, "", ""
}

// TxBuildToActionCheckReq 将 tx/build 请求体中的 params 映射到 ActionCheckReq（proposal_id 等在 params 内）。
func TxBuildToActionCheckReq(req TxBuildReq) ActionCheckReq {
	ac := ActionCheckReq{
		Action: req.Action,
		Wallet: req.Wallet,
		Params: req.Params,
	}
	if req.Params == nil {
		return ac
	}
	if v, ok := req.Params["proposal_id"]; ok {
		if pid := anyUint64Ptr(v); pid != nil {
			ac.ProposalID = pid
		}
	}
	if v, ok := req.Params["campaign_id"]; ok {
		if cid := anyUint64Ptr(v); cid != nil {
			ac.CampaignID = cid
		}
	}
	if v, ok := req.Params["milestone_index"]; ok {
		if mi := anyIntPtr(v); mi != nil {
			ac.MilestoneIndex = mi
		}
	}
	return ac
}

func anyUint64Ptr(v any) *uint64 {
	switch t := v.(type) {
	case float64:
		if t < 0 {
			return nil
		}
		u := uint64(t)
		return &u
	case json.Number:
		n, err := strconv.ParseUint(t.String(), 10, 64)
		if err != nil {
			return nil
		}
		return &n
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return nil
		}
		return &n
	case int:
		if t < 0 {
			return nil
		}
		u := uint64(t)
		return &u
	case int64:
		if t < 0 {
			return nil
		}
		u := uint64(t)
		return &u
	case uint64:
		return &t
	default:
		return nil
	}
}

func anyIntPtr(v any) *int {
	switch t := v.(type) {
	case float64:
		i := int(t)
		return &i
	case json.Number:
		n, err := strconv.Atoi(t.String())
		if err != nil {
			return nil
		}
		return &n
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return nil
		}
		return &n
	case int:
		return &t
	case int64:
		i := int(t)
		return &i
	default:
		return nil
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

// stateInvalidNeedsSyncHint 对「依赖 cp_proposals / cp_campaigns 等读模型」的失败补充同步说明；参数缺失类不追加。
func stateInvalidNeedsSyncHint(action, reason string) bool {
	if reason == errMissingProposalID || reason == errMissingCampaignID {
		return false
	}
	if strings.HasPrefix(reason, "缺少") {
		return false
	}
	switch action {
	case "review_proposal", "submit_first_round_for_review", "submit_follow_on_round_for_review",
		"review_funding_round", "launch_approved_round",
		"donate", "finalize_campaign", "claim_refund":
		return true
	default:
		return false
	}
}

// actionStateSyncHint 在「状态预检失败」时追加说明：读库可能尚未追上链上/子图展示。
func actionStateSyncHint(h *handlers.Handlers) string {
	if sgAvailable(h) {
		return " 数据可能尚未从链上完全写入 PostgreSQL（例如界面来自子图而索引库未跟上），因此暂时不能预检和构建；请稍后再试，或开启子图同步写库并等待同步完成。"
	}
	return " 数据可能尚未从链上同步到数据库，因此暂时不能预检和构建；请等待 RPC 索引写入后再试。"
}

func checkActionState(h *handlers.Handlers, req ActionCheckReq) (ok bool, state, reason string) {
	switch req.Action {
	case "review_proposal":
		return checkProposalStatus(h, req.ProposalID, "pending_review", "提案在数据库中不是「待审核」状态")
	case "submit_first_round_for_review":
		return checkProposalStatus(h, req.ProposalID, "approved", "提案尚未审核通过")
	case "submit_follow_on_round_for_review":
		return checkProposalStatus(h, req.ProposalID, "settled", "上一轮尚未结算")
	case "review_funding_round":
		return checkReviewFundingRound(h, req)
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
		if strings.TrimSpace(status) == "" {
			return false, status, msg + "（数据库中尚无该提案记录或状态为空）"
		}
		return false, status, msg + "（数据库中当前为：" + status + "）"
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

// paramsApproveTrue 解析 review_funding_round / review_proposal 等传入的 approve；缺省视为 true（与前端默认勾选一致）。
func paramsApproveTrue(req ActionCheckReq) bool {
	m, ok := req.Params.(map[string]any)
	if !ok || m == nil {
		return true
	}
	v, ok := m["approve"]
	if !ok {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		if s == "" {
			return true
		}
		return s != "false" && s != "0" && s != "no"
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	default:
		return true
	}
}

// checkReviewFundingRound 与合约一致：approve=true 仅当 PG 为 round_review_pending；approve=false 允许待审拒绝或已通过未上线时的撤销。
func checkReviewFundingRound(h *handlers.Handlers, req ActionCheckReq) (bool, string, string) {
	if req.ProposalID == nil {
		return false, "", errMissingProposalID
	}
	okP, st, msg := checkProposalStatus(h, req.ProposalID, "approved", "提案在数据库中不是「已通过」状态，不能审核本轮众筹")
	if !okP {
		return okP, st, msg
	}
	if paramsApproveTrue(req) {
		return checkRoundReviewState(h, req.ProposalID, "round_review_pending", "没有待审核的 funding round")
	}
	return checkReviewFundingRoundRejectOrRevoke(h, req.ProposalID)
}

func checkReviewFundingRoundRejectOrRevoke(h *handlers.Handlers, proposalID *uint64) (bool, string, string) {
	if proposalID == nil {
		return false, "", errMissingProposalID
	}
	var rs *string
	h.DB.Model(&models.CPProposal{}).Select("round_review_state").
		Where(whereProposalID, *proposalID).Scan(&rs)
	if rs == nil || strings.TrimSpace(*rs) == "" {
		return false, "", "没有可拒绝或撤销的 funding round（数据库轮次审核态为空，可能索引尚未写入）"
	}
	s := strings.TrimSpace(*rs)
	switch s {
	case "round_review_pending", "round_review_approved":
		return true, s, ""
	default:
		return false, s, "当前轮次状态不允许拒绝或撤销（须为「待审核本轮」或「本轮已通过、尚未上线」）"
	}
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
