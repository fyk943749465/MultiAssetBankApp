package contracts

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

//go:embed codepulse.json
var codePulseABIJSON []byte

type CodePulse struct {
	abi     abi.ABI
	addr    common.Address
	bound   *bind.BoundContract
	client  *ethclient.Client
}

// LoadCodePulseABI 解析嵌入的 CodePulse ABI（供测试或独立 ABI 编码使用）。
func LoadCodePulseABI() (abi.ABI, error) {
	return abi.JSON(bytes.NewReader(codePulseABIJSON))
}

func NewCodePulse(client *ethclient.Client, address common.Address) (*CodePulse, error) {
	parsed, err := LoadCodePulseABI()
	if err != nil {
		return nil, fmt.Errorf("parse CodePulse ABI: %w", err)
	}
	b := bind.NewBoundContract(address, parsed, client, client, client)
	return &CodePulse{
		abi:    parsed,
		addr:   address,
		bound:  b,
		client: client,
	}, nil
}

func (cp *CodePulse) Address() common.Address { return cp.addr }
func (cp *CodePulse) ABI() *abi.ABI           { return &cp.abi }

// Owner 读取合约 owner（Ownable）。
func (cp *CodePulse) Owner(ctx context.Context) (common.Address, error) {
	out, err := cp.readCall(ctx, "owner")
	if err != nil {
		return common.Address{}, err
	}
	if len(out) != 1 {
		return common.Address{}, fmt.Errorf("owner: unexpected output length %d", len(out))
	}
	addr, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("owner: unexpected output type %T", out[0])
	}
	return addr, nil
}

// Paused 读取合约暂停状态。
func (cp *CodePulse) Paused(ctx context.Context) (bool, error) {
	out, err := cp.readCall(ctx, "paused")
	if err != nil {
		return false, err
	}
	if len(out) != 1 {
		return false, fmt.Errorf("paused: unexpected output length %d", len(out))
	}
	paused, ok := out[0].(bool)
	if !ok {
		return false, fmt.Errorf("paused: unexpected output type %T", out[0])
	}
	return paused, nil
}

// IsProposalInitiator 读取地址是否在链上 proposal initiator 白名单内。
func (cp *CodePulse) IsProposalInitiator(ctx context.Context, account common.Address) (bool, error) {
	out, err := cp.readCall(ctx, "isProposalInitiator", account)
	if err != nil {
		return false, err
	}
	if len(out) != 1 {
		return false, fmt.Errorf("isProposalInitiator: unexpected output length %d", len(out))
	}
	active, ok := out[0].(bool)
	if !ok {
		return false, fmt.Errorf("isProposalInitiator: unexpected output type %T", out[0])
	}
	return active, nil
}

// PackCalldata 使用 ABI 编码 calldata。
func (cp *CodePulse) PackCalldata(method string, args ...any) ([]byte, error) {
	return cp.abi.Pack(method, args...)
}

// Simulate 使用 eth_call 模拟交易，返回 (成功, revert 数据, error)。
func (cp *CodePulse) Simulate(ctx context.Context, from common.Address, data []byte, value *big.Int) (bool, []byte, error) {
	msg := ethereum.CallMsg{
		From:  from,
		To:    &cp.addr,
		Data:  data,
		Value: value,
	}
	_, err := cp.client.CallContract(ctx, msg, nil)
	if err != nil {
		if revertData := extractCallRevertData(err); len(revertData) > 0 {
			return false, revertData, nil
		}
		return false, nil, err
	}
	return true, nil, nil
}

// extractCallRevertData 从 eth_call 失败错误里取出 revert payload（兼容 Geth、嵌套 JSON-RPC data 等）。
func extractCallRevertData(err error) []byte {
	if err == nil {
		return nil
	}
	if data, ok := ethclient.RevertErrorData(err); ok {
		return data
	}
	var de rpc.DataError
	if errors.As(err, &de) {
		if parsed := parseJSONRPCRevertPayload(de.ErrorData()); len(parsed) > 0 {
			return parsed
		}
	}
	return extractRevertData(err.Error())
}

func parseJSONRPCRevertPayload(v interface{}) []byte {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if b, err := hexutil.Decode(t); err == nil {
			return b
		}
	case []byte:
		return t
	case map[string]interface{}:
		for _, key := range []string{"data", "cause", "revertData", "error", "errorMessage"} {
			if inner, ok := t[key]; ok {
				if parsed := parseJSONRPCRevertPayload(inner); len(parsed) > 0 {
					return parsed
				}
			}
		}
	}
	return nil
}

// HumanRevertMessage 将 revert 字节解码为面向用户的说明（自定义 error、Error(string) 或 selector 提示）。
func (cp *CodePulse) HumanRevertMessage(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	name, _, decErr := cp.DecodeRevertError(data)
	if decErr == nil && name != "" {
		if msg, ok := CustomErrorMessages[name]; ok {
			return msg
		}
		return name
	}
	if s, err := abi.UnpackRevert(data); err == nil && s != "" {
		return s
	}
	return fmt.Sprintf("未识别的合约回退（错误选择器 0x%x）", data[:4])
}

// EstimateGas 估算 gas。
func (cp *CodePulse) EstimateGas(ctx context.Context, from common.Address, data []byte, value *big.Int) (uint64, error) {
	msg := ethereum.CallMsg{
		From:  from,
		To:    &cp.addr,
		Data:  data,
		Value: value,
	}
	return cp.client.EstimateGas(ctx, msg)
}

func (cp *CodePulse) readCall(ctx context.Context, method string, args ...any) ([]any, error) {
	data, err := cp.abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{
		To:   &cp.addr,
		Data: data,
	}
	raw, err := cp.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	return cp.abi.Unpack(method, raw)
}

// CallView exports readCall for handlers to fetch public constants dynamically
func (cp *CodePulse) CallView(ctx context.Context, method string, args ...any) ([]any, error) {
	return cp.readCall(ctx, method, args...)
}

// Transact 使用给定的 auth 签名并发送交易。
func (cp *CodePulse) Transact(ctx context.Context, auth *bind.TransactOpts, method string, args ...any) (*types.Transaction, error) {
	auth.Context = ctx
	return cp.bound.Transact(auth, method, args...)
}

// DecodeRevertError 将 revert data 解码为 custom error 名称和参数。
func (cp *CodePulse) DecodeRevertError(data []byte) (name string, args map[string]any, err error) {
	if len(data) < 4 {
		return "", nil, fmt.Errorf("revert data too short: %d bytes", len(data))
	}

	sel := data[:4]
	for errName, abiErr := range cp.abi.Errors {
		// abiErr.ID 为完整 Keccak-256；revert 前 4 字节为选择器。勿用 BytesToHash(4)，其会把字节右对齐到 Hash 尾部，导致永远对不上。
		if !bytes.Equal(abiErr.ID[:4], sel) {
			continue
		}
		return errName, unpackErrorArgs(abiErr, data[4:]), nil
	}

	return "", nil, fmt.Errorf("unknown error selector: 0x%x", sel)
}

func unpackErrorArgs(abiErr abi.Error, payload []byte) map[string]any {
	if len(abiErr.Inputs) == 0 {
		return nil
	}
	raw, uErr := abiErr.Unpack(payload)
	if uErr != nil {
		return nil
	}
	argMap := make(map[string]any, len(abiErr.Inputs))
	if len(abiErr.Inputs) == 1 {
		argMap[abiErr.Inputs[0].Name] = formatABIValue(raw)
	} else if slice, ok := raw.([]any); ok {
		for i, input := range abiErr.Inputs {
			if i < len(slice) {
				argMap[input.Name] = formatABIValue(slice[i])
			}
		}
	}
	return argMap
}

func extractRevertData(errMsg string) []byte {
	// go-ethereum error format: "execution reverted: ... (0x<hex>)" or just hex in the message
	// Multiple patterns depending on geth version
	patterns := []string{"0x"}
	for _, p := range patterns {
		idx := strings.LastIndex(errMsg, p)
		if idx < 0 {
			continue
		}
		hexStr := errMsg[idx+2:]
		// trim non-hex chars from end
		hexStr = strings.TrimRight(hexStr, ")\n\r\t \"'")
		if len(hexStr) >= 8 {
			if decoded, err := hex.DecodeString(hexStr); err == nil {
				return decoded
			}
		}
	}
	return nil
}

func formatABIValue(v any) any {
	switch val := v.(type) {
	case *big.Int:
		return val.String()
	case common.Address:
		return val.Hex()
	case common.Hash:
		return val.Hex()
	default:
		return val
	}
}

// ---------------------------------------------------------------------------
// Custom error 中文映射
// ---------------------------------------------------------------------------

var CustomErrorMessages = map[string]string{
	"OwnableUnauthorizedAccount":      "当前钱包不是管理员",
	"OwnableInvalidOwner":             "无效的所有者地址",
	"NotProposalInitiator":            "当前钱包不在提案发起人白名单中",
	"OnlyOrganizer":                   "只有提案发起人可执行该操作",
	"NotADeveloper":                   "当前钱包不是该项目开发者",
	"NotSnapshotDeveloper":            "当前钱包不在该阶段快照开发者名单中",
	"InvalidAccount":                  "无效的钱包地址",
	"InvalidProposal":                 "提案不存在",
	"ProposalNotPending":              "当前提案不是待审核状态",
	"ProposalNotApproved":             "当前提案尚未审核通过",
	"EmptyGithubUrl":                  "GitHub URL 不能为空",
	"GithubUrlTooLong":                "GitHub URL 超出长度限制",
	"TargetBelowMinimum":              "目标金额低于最低要求",
	"DurationBelowMinimum":            "众筹持续时间低于最低要求",
	"IncorrectMilestoneCount":         "里程碑数量不正确（必须为 3 个）",
	"EmptyMilestoneDescription":       "里程碑描述不能为空",
	"FirstRoundAlreadyLaunched":       "第一轮已经启动过",
	"FollowOnRequiresPriorRound":      "必须先完成上一轮才能发起后续轮",
	"PriorRoundNotSettled":            "上一轮众筹尚未结算",
	"NoPendingFundingRound":           "没有待审核的 funding round",
	"FundingRoundReviewInProgress":    "当前轮次仍在审核中",
	"FundingRoundNotApprovedForLaunch": "当前轮次尚未审核通过，不能启动众筹",
	"InvalidCampaign":                 "众筹轮次不存在",
	"BadState":                        "当前状态不允许执行此操作",
	"CampaignEnded":                   "众筹已结束，不能继续捐款",
	"CampaignDormant":                 "众筹处于休眠状态",
	"NotInFundraising":                "当前不在众筹阶段",
	"NotReachedDeadline":              "未到截止时间，不能结算",
	"AlreadyFinalized":                "该众筹已经结算过",
	"NotFinalized":                    "该众筹尚未结算",
	"NotSuccessful":                   "众筹未成功",
	"ZeroValue":                       "金额不能为 0",
	"NoETH":                           "未附带 ETH",
	"RefundNotAvailable":              "当前项目暂不可退款",
	"NoContribution":                  "没有捐款记录，无法退款",
	"CannotManageDevelopers":          "当前阶段不允许调整开发者名单",
	"AlreadyDeveloper":                "该地址已经是开发者",
	"TooManyDevelopers":               "开发者人数已达上限",
	"NoDevelopers":                    "没有开发者，无法结算成功",
	"BadMilestoneIndex":               "里程碑索引无效",
	"AlreadyApproved":                 "该里程碑已经审批通过",
	"PreviousMilestoneNotApproved":    "前一个里程碑尚未审批通过",
	"MilestoneLocked":                 "里程碑尚未解锁",
	"TooEarly":                        "时间锁未到期",
	"MilestoneNotApproved":            "该里程碑尚未审核通过",
	"MilestoneSettled":                "该里程碑已经结清",
	"AlreadyClaimed":                  "该阶段份额已领取",
	"NoSnapshot":                      "缺少开发者快照，无法领取",
	"AlreadySwept":                    "沉睡资金已经清扫过",
	"NothingToSweep":                  "没有可清扫的沉睡资金",
	"ExceedsPlatformBalance":          "提现金额超过平台余额",
	"ZeroWithdrawAmount":              "提现金额不能为 0",
	"TransferFailed":                  "ETH 转账失败",
	"EnforcedPause":                   "合约已暂停，无法执行操作",
	"ExpectedPause":                   "合约未处于暂停状态",
	"ReentrancyGuardReentrantCall":    "系统繁忙，请稍后重试",
}
