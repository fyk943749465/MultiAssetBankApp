package codepulse

import (
	"testing"
)

func TestFoldProposalInitiatorUpdatedsToActiveAddrs(t *testing.T) {
	rows := []sgProposalInitiatorUpdatedWire{
		{ID: "a-1", Account: "0x1111111111111111111111111111111111111111", Allowed: true, BlockNumber: "1"},
		{ID: "b-2", Account: "0x2222222222222222222222222222222222222222", Allowed: true, BlockNumber: "2"},
		{ID: "c-3", Account: "0x1111111111111111111111111111111111111111", Allowed: false, BlockNumber: "3"},
	}
	got := foldProposalInitiatorUpdatedsToActiveAddrs(rows)
	want := "0x2222222222222222222222222222222222222222"
	want = normalizeAddress(want)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %#v", got)
	}

	// 乱序输入应仍能折叠为最终状态
	shuffled := []sgProposalInitiatorUpdatedWire{
		{ID: "z", Account: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Allowed: false, BlockNumber: "10"},
		{ID: "y", Account: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Allowed: true, BlockNumber: "5"},
	}
	got2 := foldProposalInitiatorUpdatedsToActiveAddrs(shuffled)
	if len(got2) != 0 {
		t.Fatalf("expected revoked, got %#v", got2)
	}
}
