package codepulse

import "testing"

func TestParseSetProposalInitiatorRevoke(t *testing.T) {
	addr, revoke, ok := parseSetProposalInitiatorRevoke(ActionCheckReq{
		Action: "set_proposal_initiator",
		Params: map[string]any{"account": "0xAbCdef0000000000000000000000000000000000", "allowed": false},
	})
	if !ok || !revoke {
		t.Fatalf("expected revoke ok")
	}
	if addr != "0xabcdef0000000000000000000000000000000000" {
		t.Fatalf("addr %q", addr)
	}
	_, revoke2, ok2 := parseSetProposalInitiatorRevoke(ActionCheckReq{
		Action: "set_proposal_initiator",
		Params: map[string]any{"account": "0xAbCdef0000000000000000000000000000000000", "allowed": true},
	})
	if !ok2 || revoke2 {
		t.Fatalf("expected allow, got revoke=%v", revoke2)
	}
}
