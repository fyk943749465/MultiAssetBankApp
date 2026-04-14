package codepulse

import (
	"encoding/json"
	"math/big"
	"testing"

	"go-chain/backend/internal/contracts"

	"github.com/gin-gonic/gin/binding"
)

func TestSubmitProposalPackDurationFromJSONParams(t *testing.T) {
	raw := []byte(`{"action":"submit_proposal","wallet":"0xd324bB66a39d4efDEf1C6FAef569643507977C30","params":{"github_url":"https://github.com/NousResearch/hermes-agent","target":"100000000000000000","duration":"604800","milestone_descs":["完整需求分析，并给出需求文档","核心功能开发完成，并上线部署","项目线上稳定运行7个工作日"]}}`)

	var req TxBuildReq
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}

	args, _, err := newParamExtractor(req.Params).packForMethod("submitProposal")
	if err != nil {
		t.Fatal(err)
	}
	dur := args[2].(*big.Int)
	if dur.Cmp(big.NewInt(604800)) != 0 {
		t.Fatalf("duration big.Int: got %s want 604800", dur.String())
	}

	cpABI, err := contracts.LoadCodePulseABI()
	if err != nil {
		t.Fatal(err)
	}
	data, err := cpABI.Pack("submitProposal", args...)
	if err != nil {
		t.Fatal(err)
	}
	// Static head: [off str][target][duration][off arr] → duration is bytes [68:100]
	if len(data) < 100 {
		t.Fatalf("short calldata: %d", len(data))
	}
	wordDur := new(big.Int).SetBytes(data[68:100])
	if wordDur.Cmp(big.NewInt(604800)) != 0 {
		t.Fatalf("duration word in calldata: got %s (0x%x) want 604800", wordDur.String(), wordDur)
	}
}

// 与浏览器/curl 实际请求一致：严格解析 + 目标 wei 与里程碑文案与生产环境示例相同。
func TestSubmitProposalParseTxBuildBodyCalldataMatchesDurationHint(t *testing.T) {
	raw := []byte(`{"action":"submit_proposal","wallet":"0xd324bB66a39d4efDEf1C6FAef569643507977C30","params":{"github_url":"https://github.com/NousResearch/hermes-agent","target":"150000000000000000","duration":"604800","milestone_descs":["完成核心功能1","完成核心功能2","部署上线并稳定运行7个工作日"]}}`)

	req, err := parseTxBuildBody(raw)
	if err != nil {
		t.Fatal(err)
	}
	args, _, err := newParamExtractor(req.Params).packForMethod("submitProposal")
	if err != nil {
		t.Fatal(err)
	}
	durArg := args[2].(*big.Int)
	if durArg.Cmp(big.NewInt(604800)) != 0 {
		t.Fatalf("args duration: got %s", durArg.String())
	}

	cpABI, err := contracts.LoadCodePulseABI()
	if err != nil {
		t.Fatal(err)
	}
	data, err := cpABI.Pack("submitProposal", args...)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 100 {
		t.Fatalf("short calldata: %d", len(data))
	}
	wordDur := new(big.Int).SetBytes(data[68:100])
	if wordDur.Cmp(durArg) != 0 {
		t.Fatalf("calldata duration word %s != args[2] %s (若线上出现此失败说明运行中的二进制与仓库不一致或 calldata 被篡改)", wordDur.String(), durArg.String())
	}
	if wordDur.Cmp(big.NewInt(604800)) != 0 {
		t.Fatalf("duration word: got %s want 604800", wordDur.String())
	}
}

func TestSubmitProposalPackDurationViaGinJSONBinding(t *testing.T) {
	raw := []byte(`{"action":"submit_proposal","wallet":"0xd324bB66a39d4efDEf1C6FAef569643507977C30","params":{"github_url":"https://github.com/NousResearch/hermes-agent","target":"100000000000000000","duration":"604800","milestone_descs":["a","b","c"]}}`)

	var req TxBuildReq
	if err := binding.JSON.BindBody(raw, &req); err != nil {
		t.Fatal(err)
	}

	args, _, err := newParamExtractor(req.Params).packForMethod("submitProposal")
	if err != nil {
		t.Fatal(err)
	}
	dur := args[2].(*big.Int)
	if dur.Cmp(big.NewInt(604800)) != 0 {
		t.Fatalf("duration after gin binding: got %s want 604800 (params[duration] type %T value %#v)",
			dur.String(), req.Params["duration"], req.Params["duration"])
	}
}
