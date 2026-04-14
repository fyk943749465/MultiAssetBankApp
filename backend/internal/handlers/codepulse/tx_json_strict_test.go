package codepulse

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseTxBuildBodyRejectsDuplicateKeysInParams(t *testing.T) {
	body := []byte(`{"action":"submit_proposal","wallet":"0xabc","params":{"duration":"604800","duration":9676800}}`)
	_, err := parseTxBuildBody(body)
	if err == nil {
		t.Fatal("expected error for duplicate duration key")
	}
	if !strings.Contains(err.Error(), "重复 JSON 键") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseTxBuildBodyPreservesStringDuration(t *testing.T) {
	body := []byte(`{"action":"submit_proposal","wallet":"0xabc","params":{"github_url":"https://example.com/repo","target":"100000000000000000","duration":"604800","milestone_descs":["a","b","c"]}}`)
	req, err := parseTxBuildBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if req.Params["duration"] != "604800" {
		t.Fatalf("duration: got %#v (%T)", req.Params["duration"], req.Params["duration"])
	}
}

func TestParseTxBuildBodyRejectsDuplicateRootParams(t *testing.T) {
	body := []byte(`{"action":"submit_proposal","wallet":"0xabc","params":{"duration":"604800"},"params":{"duration":9676800}}`)
	_, err := parseTxBuildBody(body)
	if err == nil {
		t.Fatal("expected error for duplicate root params key")
	}
	if !strings.Contains(err.Error(), "重复 JSON 键") || !strings.Contains(err.Error(), "params") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseTxBuildBodyNumericDurationUsesJSONNumber(t *testing.T) {
	body := []byte(`{"action":"submit_proposal","wallet":"0xabc","params":{"duration":604800}}`)
	req, err := parseTxBuildBody(body)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := req.Params["duration"].(json.Number)
	if !ok {
		t.Fatalf("expected json.Number, got %T", req.Params["duration"])
	}
	if d.String() != "604800" {
		t.Fatalf("got %q", d.String())
	}
}
