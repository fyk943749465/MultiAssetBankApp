package codepulse

import (
	"testing"

	"go-chain/backend/internal/models"
)

func TestProposalAwaitingFirstRoundSubmit(t *testing.T) {
	if !proposalAwaitingFirstRoundSubmit(models.CPProposal{Status: "approved"}) {
		t.Fatal("approved with no round and no campaign should qualify")
	}
	rs := "round_review_pending"
	if proposalAwaitingFirstRoundSubmit(models.CPProposal{Status: "approved", RoundReviewState: &rs}) {
		t.Fatal("pending round should not qualify")
	}
	rs2 := "round_review_approved"
	if proposalAwaitingFirstRoundSubmit(models.CPProposal{Status: "approved", RoundReviewState: &rs2}) {
		t.Fatal("approved round should use other bucket")
	}
	cid := uint64(9)
	if proposalAwaitingFirstRoundSubmit(models.CPProposal{Status: "approved", LastCampaignID: &cid}) {
		t.Fatal("after launch should not qualify")
	}
}

func TestProposalInLaunchQueue(t *testing.T) {
	rs := "round_review_approved"
	if !proposalInLaunchQueue(models.CPProposal{Status: "approved", RoundReviewState: &rs}) {
		t.Fatal("round approved waiting launch should qualify")
	}
	cid := uint64(1)
	if proposalInLaunchQueue(models.CPProposal{Status: "approved", LastCampaignID: &cid}) {
		t.Fatal("launched (last_campaign set) should not qualify even if round cleared in PG")
	}
}
