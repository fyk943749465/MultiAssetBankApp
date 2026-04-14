package codepulse

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// parseTxBuildBody 解析 tx/build、tx/submit 请求体；根对象与 params 均拒绝重复 JSON 键，
// 避免 encoding/json 对重复键静默保留最后一个值，导致错误 calldata（例如顶层出现两个 params）。
func parseTxBuildBody(data []byte) (TxBuildReq, error) {
	root, err := decodeUniqueJSONObject(bytes.TrimSpace(data), "请求体根对象")
	if err != nil {
		return TxBuildReq{}, err
	}
	action, err := rawJSONStringFromRoot(root, "action", true)
	if err != nil {
		return TxBuildReq{}, err
	}
	wallet, err := rawJSONStringFromRoot(root, "wallet", true)
	if err != nil {
		return TxBuildReq{}, err
	}
	paramsRaw, ok := root["params"]
	if !ok {
		return TxBuildReq{}, fmt.Errorf("params is required")
	}
	params, err := paramsJSONToMapStrict(paramsRaw)
	if err != nil {
		return TxBuildReq{}, err
	}
	return TxBuildReq{Action: action, Wallet: wallet, Params: params}, nil
}

func rawJSONStringFromRoot(root map[string]json.RawMessage, key string, required bool) (string, error) {
	raw, ok := root[key]
	if !ok {
		if required {
			return "", fmt.Errorf("%s is required", key)
		}
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	if required && s == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return s, nil
}

func paramsJSONToMapStrict(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	uniq, err := decodeUniqueJSONObject(raw, "params")
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(uniq))
	for k, vraw := range uniq {
		var v any
		dec := json.NewDecoder(bytes.NewReader(vraw))
		dec.UseNumber()
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("params.%s: %w", k, err)
		}
		out[k] = v
	}
	return out, nil
}

func decodeUniqueJSONObject(data []byte, ctx string) (map[string]json.RawMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return map[string]json.RawMessage{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, ok := t.(json.Delim)
	if !ok || d != '{' {
		return nil, fmt.Errorf("%s：应为 JSON 对象", ctx)
	}
	seen := make(map[string]struct{})
	out := make(map[string]json.RawMessage)
	for dec.More() {
		tk, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tk.(string)
		if !ok {
			return nil, fmt.Errorf("%s：期望字符串键", ctx)
		}
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("重复 JSON 键 %q（%s）。标准解析会静默采用最后一个值，易导致错误参数", key, ctx)
		}
		seen[key] = struct{}{}
		var vr json.RawMessage
		if err := dec.Decode(&vr); err != nil {
			return nil, err
		}
		out[key] = vr
	}
	end, err := dec.Token()
	if err != nil {
		return nil, err
	}
	ed, ok := end.(json.Delim)
	if !ok || ed != '}' {
		return nil, fmt.Errorf("%s：对象未正确闭合", ctx)
	}
	return out, nil
}
