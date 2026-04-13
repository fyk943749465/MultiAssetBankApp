package codepulse

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"go-chain/backend/internal/contracts"
	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

var actionToMethod = map[string]string{
	"submit_proposal":                   "submitProposal",
	"review_proposal":                   "reviewProposal",
	"submit_first_round_for_review":     "submitFirstRoundForReview",
	"submit_follow_on_round_for_review": "submitFollowOnRoundForReview",
	"review_funding_round":              "reviewFundingRound",
	"launch_approved_round":             "launchApprovedRound",
	"donate":                            "donate",
	"donate_to_platform":                "donateToPlatform",
	"finalize_campaign":                 "finalizeCampaign",
	"claim_refund":                      "claimRefund",
	"add_developer":                     "addCampaignDeveloper",
	"remove_developer":                  "removeCampaignDeveloper",
	"approve_milestone":                 "approveMilestone",
	"claim_milestone_share":             "claimMilestoneShare",
	"sweep_stale_funds":                 "sweepStaleFunds",
	"set_proposal_initiator":            "setProposalInitiator",
	"withdraw_platform_funds":           "withdrawPlatformFunds",
	"pause":                             "pause",
	"unpause":                           "unpause",
	"transfer_ownership":                "transferOwnership",
	"renounce_ownership":                "renounceOwnership",
}

// TxBuildReq 构造/提交交易的请求体。
type TxBuildReq struct {
	Action string         `json:"action" binding:"required"`
	Wallet string         `json:"wallet" binding:"required"`
	Params map[string]any `json:"params"`
}

// TxBuild 构造交易 calldata 并模拟。
// @Summary      Build transaction calldata
// @Tags         code-pulse
// @Accept       json
// @Produce      json
// @Param        body body TxBuildReq true "Action + wallet + params"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/tx/build [post]
func TxBuild(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp, req, method, ok := parseTxRequest(h, c)
		if !ok {
			return
		}

		args, value, err := newParamExtractor(req.Params).packForMethod(method)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数编码失败: " + err.Error()})
			return
		}

		data, err := cp.PackCalldata(method, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ABI 编码失败: " + err.Error()})
			return
		}

		ctx := c.Request.Context()
		from := common.HexToAddress(req.Wallet)
		simOK, revertData, simErr := cp.Simulate(ctx, from, data, value)

		resp := gin.H{
			"to":            cp.Address().Hex(),
			"data":          "0x" + common.Bytes2Hex(data),
			"value":         formatValue(value),
			"simulation_ok": simOK,
		}
		if h.Chain != nil && h.Chain.Eth() != nil {
			if chainID, cidErr := h.Chain.Eth().ChainID(ctx); cidErr == nil {
				resp["chain_id"] = chainID.Uint64()
			}
		}

		if !simOK {
			fillRevertInfo(resp, cp, revertData, simErr)
			c.JSON(http.StatusOK, resp)
			return
		}

		if gas, gasErr := cp.EstimateGas(ctx, from, data, value); gasErr == nil {
			resp["gas_estimate"] = gas
		}
		c.JSON(http.StatusOK, resp)
	}
}

// TxSubmit 后端代签发送交易。
// @Summary      Submit transaction (server-signed)
// @Tags         code-pulse
// @Accept       json
// @Produce      json
// @Param        body body TxBuildReq true "Action + params"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/tx/submit [post]
func TxSubmit(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp, req, method, ok := parseTxRequest(h, c)
		if !ok {
			return
		}
		if h.TxKey == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ETH_PRIVATE_KEY 未配置，无法代发交易"})
			return
		}

		args, value, err := newParamExtractor(req.Params).packForMethod(method)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数编码失败: " + err.Error()})
			return
		}

		if h.Chain == nil || h.Chain.Eth() == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ETH_RPC_URL 未配置"})
			return
		}
		ctx := c.Request.Context()
		chainID, err := h.Chain.Eth().ChainID(ctx)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "读取 chainId 失败: " + err.Error()})
			return
		}

		auth, err := bind.NewKeyedTransactorWithChainID(h.TxKey, chainID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if value != nil && value.Sign() > 0 {
			auth.Value = value
		}

		tx, err := cp.Transact(ctx, auth, method, args...)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		if h.DB != nil {
			payloadJSON, _ := json.Marshal(req.Params)
			simOK := true
			attempt := models.CPTxAttempt{
				WalletAddress:  auth.From.Hex(),
				RoleSnapshot:   models.JSONB([]byte(`{"source":"server_key"}`)),
				Action:         req.Action,
				RequestPayload: models.JSONB(payloadJSON),
				SimulationOK:   &simOK,
				TxHash:         strPtr(tx.Hash().Hex()),
				TxStatus:       "submitted",
			}
			h.DB.Create(&attempt)
		}

		c.JSON(http.StatusOK, gin.H{"tx_hash": tx.Hash().Hex(), "action": req.Action})
	}
}

// TxDetail 查询交易尝试状态。
// @Summary      Get transaction attempt
// @Tags         code-pulse
// @Produce      json
// @Param        attemptId path int true "Attempt ID"
// @Success      200 {object} map[string]any
// @Failure      404 {object} handlers.ErrorJSON
// @Router       /api/code-pulse/tx/{attemptId} [get]
func TxDetail(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}
		aid, err := strconv.ParseUint(c.Param("attemptId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attempt_id"})
			return
		}
		var attempt models.CPTxAttempt
		if err := h.DB.First(&attempt, aid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "attempt not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"attempt": attempt})
	}
}

func parseTxRequest(h *handlers.Handlers, c *gin.Context) (*contracts.CodePulse, TxBuildReq, string, bool) {
	if h.CodePulse == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "CodePulse 合约未配置（需要 CODE_PULSE_ADDRESS 和 ETH_RPC_URL）"})
		return nil, TxBuildReq{}, "", false
	}
	var req TxBuildReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action and wallet are required"})
		return nil, TxBuildReq{}, "", false
	}
	method, ok := actionToMethod[req.Action]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action: " + req.Action})
		return nil, TxBuildReq{}, "", false
	}
	return h.CodePulse, req, method, true
}

func fillRevertInfo(resp gin.H, cp *contracts.CodePulse, revertData []byte, simErr error) {
	if revertData != nil {
		errName, errArgs, _ := cp.DecodeRevertError(revertData)
		resp["revert_error_name"] = errName
		resp["revert_error_args"] = errArgs
		if msg, exists := contracts.CustomErrorMessages[errName]; exists {
			resp["revert_message"] = msg
		}
	} else if simErr != nil {
		resp["revert_message"] = simErr.Error()
	}
}

func formatValue(v *big.Int) string {
	if v != nil && v.Sign() > 0 {
		return v.String()
	}
	return "0"
}

func strPtr(s string) *string { return &s }

const errMissingParam = "missing param: %s"

type paramExtractor struct {
	p map[string]any
}

func newParamExtractor(p map[string]any) *paramExtractor {
	if p == nil {
		p = map[string]any{}
	}
	return &paramExtractor{p: p}
}

func (pe *paramExtractor) bigInt(key string) (*big.Int, error) {
	v, ok := pe.p[key]
	if !ok {
		return nil, fmt.Errorf(errMissingParam, key)
	}
	switch val := v.(type) {
	case string:
		n, ok := new(big.Int).SetString(val, 10)
		if !ok {
			return nil, fmt.Errorf("invalid number: %s", val)
		}
		return n, nil
	case float64:
		return big.NewInt(int64(val)), nil
	case json.Number:
		n, ok := new(big.Int).SetString(val.String(), 10)
		if !ok {
			return nil, fmt.Errorf("invalid number: %s", val)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("param %s: expected number, got %T", key, v)
	}
}

func (pe *paramExtractor) str(key string) (string, error) {
	v, ok := pe.p[key]
	if !ok {
		return "", fmt.Errorf(errMissingParam, key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("param %s: expected string", key)
	}
	return s, nil
}

func (pe *paramExtractor) boolean(key string) (bool, error) {
	v, ok := pe.p[key]
	if !ok {
		return false, fmt.Errorf(errMissingParam, key)
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("param %s: expected bool", key)
	}
	return b, nil
}

func (pe *paramExtractor) address(key string) (common.Address, error) {
	s, err := pe.str(key)
	if err != nil {
		return common.Address{}, err
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("param %s: invalid address", key)
	}
	return common.HexToAddress(s), nil
}

func (pe *paramExtractor) stringSlice(key string) ([]string, error) {
	v, ok := pe.p[key]
	if !ok {
		return nil, fmt.Errorf(errMissingParam, key)
	}
	switch val := v.(type) {
	case []any:
		return toStringSlice(key, val)
	case []string:
		return val, nil
	default:
		return nil, fmt.Errorf("param %s: expected string array", key)
	}
}

func toStringSlice(key string, items []any) ([]string, error) {
	strs := make([]string, len(items))
	for i, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("param %s[%d]: expected string", key, i)
		}
		strs[i] = s
	}
	return strs, nil
}

func (pe *paramExtractor) packForMethod(method string) ([]any, *big.Int, error) {
	switch method {
	case "submitProposal":
		return pe.packSubmitProposal()
	case "submitFollowOnRoundForReview":
		return pe.packFollowOnRound()
	case "reviewProposal", "reviewFundingRound":
		return pe.packProposalBool()
	case "submitFirstRoundForReview", "launchApprovedRound":
		return pe.packProposalOnly()
	case "donate":
		return pe.packCampaignPayable()
	case "donateToPlatform":
		return pe.packPayableNoArgs()
	case "finalizeCampaign", "claimRefund", "sweepStaleFunds":
		return pe.packCampaignOnly()
	case "addCampaignDeveloper", "removeCampaignDeveloper":
		return pe.packCampaignAddress()
	case "approveMilestone", "claimMilestoneShare":
		return pe.packCampaignMilestone()
	case "setProposalInitiator":
		return pe.packAddressBool()
	case "withdrawPlatformFunds":
		return pe.packAmount()
	case "transferOwnership":
		return pe.packNewOwner()
	case "pause", "unpause", "renounceOwnership":
		return []any{}, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported method: %s", method)
	}
}

func (pe *paramExtractor) packSubmitProposal() ([]any, *big.Int, error) {
	url, e := pe.str("github_url"); if e != nil { return nil, nil, e }
	target, e := pe.bigInt("target"); if e != nil { return nil, nil, e }
	dur, e := pe.bigInt("duration"); if e != nil { return nil, nil, e }
	descs, e := pe.stringSlice("milestone_descs"); if e != nil { return nil, nil, e }
	return []any{url, target, dur, descs}, nil, nil
}

func (pe *paramExtractor) packFollowOnRound() ([]any, *big.Int, error) {
	pid, e := pe.bigInt("proposal_id"); if e != nil { return nil, nil, e }
	target, e := pe.bigInt("target"); if e != nil { return nil, nil, e }
	dur, e := pe.bigInt("duration"); if e != nil { return nil, nil, e }
	descs, e := pe.stringSlice("milestone_descs"); if e != nil { return nil, nil, e }
	return []any{pid, target, dur, descs}, nil, nil
}

func (pe *paramExtractor) packProposalBool() ([]any, *big.Int, error) {
	pid, e := pe.bigInt("proposal_id"); if e != nil { return nil, nil, e }
	approve, e := pe.boolean("approve"); if e != nil { return nil, nil, e }
	return []any{pid, approve}, nil, nil
}

func (pe *paramExtractor) packProposalOnly() ([]any, *big.Int, error) {
	pid, e := pe.bigInt("proposal_id"); if e != nil { return nil, nil, e }
	return []any{pid}, nil, nil
}

func (pe *paramExtractor) packCampaignPayable() ([]any, *big.Int, error) {
	cid, e := pe.bigInt("campaign_id"); if e != nil { return nil, nil, e }
	val, e := pe.bigInt("value"); if e != nil { return nil, nil, e }
	return []any{cid}, val, nil
}

func (pe *paramExtractor) packPayableNoArgs() ([]any, *big.Int, error) {
	val, e := pe.bigInt("value"); if e != nil { return nil, nil, e }
	return []any{}, val, nil
}

func (pe *paramExtractor) packCampaignOnly() ([]any, *big.Int, error) {
	cid, e := pe.bigInt("campaign_id"); if e != nil { return nil, nil, e }
	return []any{cid}, nil, nil
}

func (pe *paramExtractor) packCampaignAddress() ([]any, *big.Int, error) {
	cid, e := pe.bigInt("campaign_id"); if e != nil { return nil, nil, e }
	addr, e := pe.address("account"); if e != nil { return nil, nil, e }
	return []any{cid, addr}, nil, nil
}

func (pe *paramExtractor) packCampaignMilestone() ([]any, *big.Int, error) {
	cid, e := pe.bigInt("campaign_id"); if e != nil { return nil, nil, e }
	mIdx, e := pe.bigInt("milestone_index"); if e != nil { return nil, nil, e }
	return []any{cid, mIdx}, nil, nil
}

func (pe *paramExtractor) packAddressBool() ([]any, *big.Int, error) {
	addr, e := pe.address("account"); if e != nil { return nil, nil, e }
	allowed, e := pe.boolean("allowed"); if e != nil { return nil, nil, e }
	return []any{addr, allowed}, nil, nil
}

func (pe *paramExtractor) packAmount() ([]any, *big.Int, error) {
	amount, e := pe.bigInt("amount"); if e != nil { return nil, nil, e }
	return []any{amount}, nil, nil
}

func (pe *paramExtractor) packNewOwner() ([]any, *big.Int, error) {
	addr, e := pe.address("new_owner"); if e != nil { return nil, nil, e }
	return []any{addr}, nil, nil
}
