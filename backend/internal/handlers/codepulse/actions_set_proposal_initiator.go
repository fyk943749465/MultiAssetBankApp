package codepulse

import (
	"fmt"
	"os"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
)

// envBlockInitiatorRevokeOrganizer 为 true 时：若撤销 initiator 的目标地址仍是「非拒绝态」提案的 organizer，则预检/构建直接拒绝（管理员需先处理提案或关闭该策略）。
func envBlockInitiatorRevokeOrganizer() bool {
	v := strings.TrimSpace(os.Getenv("CODE_PULSE_BLOCK_INITIATOR_REVOKE_ORGANIZER"))
	return strings.EqualFold(v, "true") || v == "1"
}

func paramsAsMap(req ActionCheckReq) (map[string]any, bool) {
	if req.Params == nil {
		return nil, false
	}
	m, ok := req.Params.(map[string]any)
	return m, ok
}

func parseSetProposalInitiatorRevoke(req ActionCheckReq) (targetLower string, revoke bool, ok bool) {
	m, ok := paramsAsMap(req)
	if !ok {
		return "", false, false
	}
	raw, ha := m["account"]
	if !ha {
		return "", false, false
	}
	s, isStr := raw.(string)
	if !isStr {
		return "", false, false
	}
	s = strings.TrimSpace(s)
	if s == "" || !common.IsHexAddress(s) {
		return "", false, false
	}
	targetLower = normalizeAddress(common.HexToAddress(s).Hex())

	allowed := true
	if v, has := m["allowed"]; has {
		switch t := v.(type) {
		case bool:
			allowed = t
		case string:
			allowed = strings.EqualFold(strings.TrimSpace(t), "true") || strings.TrimSpace(t) == "1"
		case float64:
			allowed = t != 0
		}
	}
	return targetLower, !allowed, true
}

// countOrganizerProposalsNonRejected 统计该地址作为 organizer、且提案状态不是明确拒绝类的条数（读库；用于撤销 initiator 风险提示）。
func countOrganizerProposalsNonRejected(h *handlers.Handlers, organizerLower string) int64 {
	if h == nil || h.DB == nil {
		return 0
	}
	var n int64
	h.DB.Model(&models.CPProposal{}).
		Where("LOWER(organizer_address) = ? AND status NOT IN ?", organizerLower, []string{"rejected", "round_review_rejected"}).
		Count(&n)
	return n
}

// setProposalInitiatorRevokeBlocked 策略硬拦：见 envBlockInitiatorRevokeOrganizer。
func setProposalInitiatorRevokeBlocked(h *handlers.Handlers, req ActionCheckReq) (blocked bool, message string) {
	target, revoke, ok := parseSetProposalInitiatorRevoke(req)
	if !ok || !revoke || target == "" {
		return false, ""
	}
	if !envBlockInitiatorRevokeOrganizer() {
		return false, ""
	}
	n := countOrganizerProposalsNonRejected(h, target)
	if n == 0 {
		return false, ""
	}
	return true, fmt.Sprintf(
		"策略禁止撤销 initiator：该地址在库中仍是 %d 条提案的 organizer（状态非 rejected / round_review_rejected）。链上在提交提案后通常以 organizer 继续融资与 launch；若确需撤销，请先处理相关提案或关闭环境变量 CODE_PULSE_BLOCK_INITIATOR_REVOKE_ORGANIZER。",
		n,
	)
}

// setProposalInitiatorRevokeAdvisory 软提示：不拦交易，供预检与 tx/build 展示。
func setProposalInitiatorRevokeAdvisory(h *handlers.Handlers, req ActionCheckReq) (code, message string) {
	target, revoke, ok := parseSetProposalInitiatorRevoke(req)
	if !ok || !revoke || target == "" {
		return "", ""
	}
	if envBlockInitiatorRevokeOrganizer() {
		return "", ""
	}
	n := countOrganizerProposalsNonRejected(h, target)
	if n == 0 {
		return "", ""
	}
	return "initiator_revoke_organizer_proposals",
		fmt.Sprintf(
			"该地址在库中仍是 %d 条提案的 organizer（非拒绝态）。撤销 initiator 白名单后，其不能再提交「新」提案；已绑定提案在链上通常仍以 organizer 身份完成轮次审核、launch 与活动管理。若你希望彻底禁止其操作，需结合合约与治理策略评估（本接口仅提示，不代替链上规则）。",
			n,
		)
}
