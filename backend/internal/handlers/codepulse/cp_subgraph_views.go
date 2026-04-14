package codepulse

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"strings"
	"time"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"
)

// ──────────────────────────────────────────────
// 通用：子图是否可用
// ──────────────────────────────────────────────

func sgAvailable(h *handlers.Handlers) bool {
	return h != nil && h.SubgraphCodePulse != nil && h.SubgraphCodePulse.Configured()
}

// ──────────────────────────────────────────────
// Summary 统计：子图全量
// ──────────────────────────────────────────────

const sgSummaryQuery = `
{
  proposalSubmitteds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId
  }
  proposalRevieweds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId
    approved
  }
  crowdfundingLauncheds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    campaignId
  }
  campaignFinalizeds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    campaignId
    successful
  }
  donateds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    campaignId
    amount
  }
  refundClaimeds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    campaignId
    amount
  }
}
`

type sgSummaryResult struct {
	ProposalTotal   int64
	PendingReview   int64
	Approved        int64
	CampaignTotal   int64
	Fundraising     int64
	Successful      int64
	Failed          int64
	TotalRaisedWei  *big.Int
	TotalRefundWei  *big.Int
	OK              bool
}

func sgQuerySummary(ctx context.Context, h *handlers.Handlers) sgSummaryResult {
	if !sgAvailable(h) {
		return sgSummaryResult{}
	}
	raw, err := h.SubgraphCodePulse.Query(ctx, sgSummaryQuery, nil)
	if err != nil {
		return sgSummaryResult{}
	}
	var data struct {
		ProposalSubmitteds []struct{ ProposalID string `json:"proposalId"` } `json:"proposalSubmitteds"`
		ProposalRevieweds  []struct {
			ProposalID string `json:"proposalId"`
			Approved   bool   `json:"approved"`
		} `json:"proposalRevieweds"`
		CrowdfundingLauncheds []struct{ CampaignID string `json:"campaignId"` } `json:"crowdfundingLauncheds"`
		CampaignFinalizeds    []struct {
			CampaignID string `json:"campaignId"`
			Successful bool   `json:"successful"`
		} `json:"campaignFinalizeds"`
		Donateds      []struct{ CampaignID string `json:"campaignId"`; Amount string `json:"amount"` } `json:"donateds"`
		RefundClaimeds []struct{ CampaignID string `json:"campaignId"`; Amount string `json:"amount"` } `json:"refundClaimeds"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return sgSummaryResult{}
	}

	reviewMap := make(map[string]*bool)
	for _, r := range data.ProposalRevieweds {
		a := r.Approved
		reviewMap[r.ProposalID] = &a
	}

	var pendingReview, approved, rejected int64
	for _, p := range data.ProposalSubmitteds {
		if rev, ok := reviewMap[p.ProposalID]; ok {
			if *rev {
				approved++
			} else {
				rejected++
			}
		} else {
			pendingReview++
		}
	}
	_ = rejected

	finalMap := make(map[string]*bool)
	for _, f := range data.CampaignFinalizeds {
		s := f.Successful
		finalMap[f.CampaignID] = &s
	}
	var fundraising, successful, failed int64
	for _, c := range data.CrowdfundingLauncheds {
		if f, ok := finalMap[c.CampaignID]; ok {
			if *f {
				successful++
			} else {
				failed++
			}
		} else {
			fundraising++
		}
	}

	totalRaised := new(big.Int)
	for _, d := range data.Donateds {
		if n, ok := new(big.Int).SetString(d.Amount, 10); ok {
			totalRaised.Add(totalRaised, n)
		}
	}
	totalRefund := new(big.Int)
	for _, r := range data.RefundClaimeds {
		if n, ok := new(big.Int).SetString(r.Amount, 10); ok {
			totalRefund.Add(totalRefund, n)
		}
	}

	return sgSummaryResult{
		ProposalTotal:  int64(len(data.ProposalSubmitteds)),
		PendingReview:  pendingReview,
		Approved:       approved,
		CampaignTotal:  int64(len(data.CrowdfundingLauncheds)),
		Fundraising:    fundraising,
		Successful:     successful,
		Failed:         failed,
		TotalRaisedWei: totalRaised,
		TotalRefundWei: totalRefund,
		OK:             true,
	}
}

// ──────────────────────────────────────────────
// Proposals 列表：以子图拉全量 + 过滤
// ──────────────────────────────────────────────

const sgProposalListQuery = `
{
  proposalSubmitteds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId organizer githubUrl target duration blockTimestamp transactionHash blockNumber
  }
  proposalRevieweds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId approved blockNumber
  }
  fundingRoundSubmittedForReviews(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId blockNumber
  }
  fundingRoundRevieweds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId approved blockNumber
  }
  crowdfundingLauncheds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId campaignId blockNumber
  }
  campaignFinalizeds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    campaignId successful blockNumber
  }
}
`

func sgQueryAllProposals(ctx context.Context, h *handlers.Handlers) ([]models.CPProposal, bool) {
	if !sgAvailable(h) {
		return nil, false
	}
	raw, err := h.SubgraphCodePulse.Query(ctx, sgProposalListQuery, nil)
	if err != nil {
		return nil, false
	}
	var pipe struct {
		ProposalSubmitteds             []struct {
			ProposalID string `json:"proposalId"`
			Organizer  string `json:"organizer"`
			GithubURL  string `json:"githubUrl"`
			Target     string `json:"target"`
			Duration   string `json:"duration"`
			BlockTS    string `json:"blockTimestamp"`
			TxHash     string `json:"transactionHash"`
			BlockNum   string `json:"blockNumber"`
		} `json:"proposalSubmitteds"`
		ProposalRevieweds              []sgEvPropReview  `json:"proposalRevieweds"`
		FundingRoundSubmittedForReviews []sgEvFRSubmit    `json:"fundingRoundSubmittedForReviews"`
		FundingRoundRevieweds          []sgEvFRReview    `json:"fundingRoundRevieweds"`
		CrowdfundingLauncheds          []sgEvLaunch      `json:"crowdfundingLauncheds"`
		CampaignFinalizeds             []sgEvCampFinalize `json:"campaignFinalizeds"`
	}
	if json.Unmarshal(raw, &pipe) != nil {
		return nil, false
	}

	launchByCID := make(map[uint64]sgEvLaunch)
	for _, l := range pipe.CrowdfundingLauncheds {
		cid, err := parseSubgraphUint(l.CampaignID)
		if err != nil {
			continue
		}
		launchByCID[cid] = l
	}
	finalByCID := make(map[uint64]sgEvCampFinalize)
	for _, f := range pipe.CampaignFinalizeds {
		cid, err := parseSubgraphUint(f.CampaignID)
		if err != nil {
			continue
		}
		finalByCID[cid] = f
	}

	pipeEvents := sgInitiatorPipeEvents{
		ProposalRevieweds:               pipe.ProposalRevieweds,
		FundingRoundSubmittedForReviews: pipe.FundingRoundSubmittedForReviews,
		FundingRoundRevieweds:           pipe.FundingRoundRevieweds,
		CrowdfundingLauncheds:           pipe.CrowdfundingLauncheds,
	}

	pidSeen := make(map[uint64]struct{})
	proposals := make([]models.CPProposal, 0, len(pipe.ProposalSubmitteds))
	for _, s := range pipe.ProposalSubmitteds {
		pid, err := parseSubgraphUint(s.ProposalID)
		if err != nil {
			continue
		}
		if _, ok := pidSeen[pid]; ok {
			continue
		}
		pidSeen[pid] = struct{}{}

		st := simulateProposalStateFromSubgraph(pid, pipeEvents, launchByCID, finalByCID)
		ts := parseSubgraphTime(s.BlockTS)
		txh := s.TxHash
		bn := mustParseBN(s.BlockNum)
		p := models.CPProposal{
			ProposalID:           pid,
			OrganizerAddress:     strings.ToLower(s.Organizer),
			GithubURL:            s.GithubURL,
			TargetWei:            strings.TrimSpace(s.Target),
			DurationSeconds:      mustParseInt64(s.Duration),
			SubmittedTxHash:      &txh,
			SubmittedBlockNumber: &bn,
			SubmittedAt:          ts,
			CreatedAt:            time.Now().UTC(),
			UpdatedAt:            time.Now().UTC(),
		}
		applySimulatedState(&p, st)
		proposals = append(proposals, p)
	}
	return proposals, true
}

// ──────────────────────────────────────────────
// Campaign 全量列表
// ──────────────────────────────────────────────

const sgCampaignListQuery = `
{
  crowdfundingLauncheds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    proposalId campaignId organizer githubUrl target deadline roundIndex blockNumber blockTimestamp transactionHash
  }
  campaignFinalizeds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    campaignId successful blockNumber blockTimestamp
  }
  donateds(first: 5000, orderBy: blockNumber, orderDirection: desc) {
    campaignId contributor amount
  }
}
`

func sgQueryAllCampaigns(ctx context.Context, h *handlers.Handlers) ([]models.CPCampaign, bool) {
	if !sgAvailable(h) {
		return nil, false
	}
	raw, err := h.SubgraphCodePulse.Query(ctx, sgCampaignListQuery, nil)
	if err != nil {
		return nil, false
	}
	var data struct {
		CrowdfundingLauncheds []struct {
			ProposalID string `json:"proposalId"`
			CampaignID string `json:"campaignId"`
			Organizer  string `json:"organizer"`
			GithubURL  string `json:"githubUrl"`
			Target     string `json:"target"`
			Deadline   string `json:"deadline"`
			RoundIndex string `json:"roundIndex"`
			BlockNum   string `json:"blockNumber"`
			BlockTS    string `json:"blockTimestamp"`
			TxHash     string `json:"transactionHash"`
		} `json:"crowdfundingLauncheds"`
		CampaignFinalizeds []struct {
			CampaignID string `json:"campaignId"`
			Successful bool   `json:"successful"`
			BlockNum   string `json:"blockNumber"`
			BlockTS    string `json:"blockTimestamp"`
		} `json:"campaignFinalizeds"`
		Donateds []struct {
			CampaignID  string `json:"campaignId"`
			Contributor string `json:"contributor"`
			Amount      string `json:"amount"`
		} `json:"donateds"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return nil, false
	}

	finalMap := make(map[uint64]struct{ Successful bool; TS string })
	for _, f := range data.CampaignFinalizeds {
		cid, err := parseSubgraphUint(f.CampaignID)
		if err != nil {
			continue
		}
		finalMap[cid] = struct{ Successful bool; TS string }{f.Successful, f.BlockTS}
	}

	raisedMap := make(map[uint64]*big.Int)
	donorMap := make(map[uint64]map[string]struct{})
	for _, d := range data.Donateds {
		cid, err := parseSubgraphUint(d.CampaignID)
		if err != nil {
			continue
		}
		if raisedMap[cid] == nil {
			raisedMap[cid] = new(big.Int)
		}
		if n, ok := new(big.Int).SetString(d.Amount, 10); ok {
			raisedMap[cid].Add(raisedMap[cid], n)
		}
		if donorMap[cid] == nil {
			donorMap[cid] = make(map[string]struct{})
		}
		donorMap[cid][strings.ToLower(d.Contributor)] = struct{}{}
	}

	cidSeen := make(map[uint64]struct{})
	campaigns := make([]models.CPCampaign, 0, len(data.CrowdfundingLauncheds))
	for _, l := range data.CrowdfundingLauncheds {
		cid, err := parseSubgraphUint(l.CampaignID)
		if err != nil {
			continue
		}
		if _, ok := cidSeen[cid]; ok {
			continue
		}
		cidSeen[cid] = struct{}{}

		pid, _ := parseSubgraphUint(l.ProposalID)
		deadlineUnix := mustParseInt64(l.Deadline)
		launchedTS := parseBlockTS(l.BlockTS)
		ri := int(mustParseInt64(l.RoundIndex))

		state := "fundraising"
		stateCode := 1
		var finalizedAt *time.Time
		var successAt *time.Time
		if f, ok := finalMap[cid]; ok {
			if f.Successful {
				state = "successful"
				stateCode = 2
				t := time.Unix(parseBlockTS(f.TS), 0).UTC()
				finalizedAt = &t
				successAt = &t
			} else {
				state = "failed_refundable"
				stateCode = 3
				t := time.Unix(parseBlockTS(f.TS), 0).UTC()
				finalizedAt = &t
			}
		}

		raised := "0"
		if r, ok := raisedMap[cid]; ok {
			raised = r.String()
		}
		donors := 0
		if d, ok := donorMap[cid]; ok {
			donors = len(d)
		}

		c := models.CPCampaign{
			CampaignID:             cid,
			ProposalID:             pid,
			RoundIndex:             ri,
			OrganizerAddress:       strings.ToLower(l.Organizer),
			GithubURL:              l.GithubURL,
			TargetWei:              strings.TrimSpace(l.Target),
			DeadlineAt:             time.Unix(deadlineUnix, 0).UTC(),
			AmountRaisedWei:        raised,
			TotalWithdrawnWei:      "0",
			UnclaimedRefundPoolWei: "0",
			State:                  state,
			StateCode:              stateCode,
			DonorCount:             donors,
			LaunchedTxHash:         l.TxHash,
			LaunchedBlockNumber:    mustParseBN(l.BlockNum),
			LaunchedAt:             time.Unix(launchedTS, 0).UTC(),
			FinalizedAt:            finalizedAt,
			SuccessAt:              successAt,
			CreatedAt:              time.Now().UTC(),
			UpdatedAt:              time.Now().UTC(),
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, true
}

// ──────────────────────────────────────────────
// Wallet overview 的子图补全：捐助数 & 开发者 campaign 数
// ──────────────────────────────────────────────

const sgWalletContributorCountQuery = `
query WalletDonations($addr: Bytes!) {
  donateds(first: 1000, where: { contributor: $addr }) {
    campaignId
  }
}
`

const sgWalletDeveloperCountQuery = `
query WalletDev($addr: Bytes!) {
  developerAddeds(first: 1000, where: { developer: $addr }) {
    campaignId
  }
  developerRemoveds(first: 1000, where: { developer: $addr }) {
    campaignId
  }
}
`

func sgDonationCount(ctx context.Context, h *handlers.Handlers, addr string) int64 {
	if !sgAvailable(h) {
		return 0
	}
	raw, err := h.SubgraphCodePulse.Query(ctx, sgWalletContributorCountQuery, map[string]any{"addr": addr})
	if err != nil {
		return 0
	}
	var data struct {
		Donateds []struct{ CampaignID string `json:"campaignId"` } `json:"donateds"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return 0
	}
	seen := make(map[string]struct{})
	for _, d := range data.Donateds {
		seen[d.CampaignID] = struct{}{}
	}
	return int64(len(seen))
}

func sgDeveloperCampaignCount(ctx context.Context, h *handlers.Handlers, addr string) int64 {
	if !sgAvailable(h) {
		return 0
	}
	raw, err := h.SubgraphCodePulse.Query(ctx, sgWalletDeveloperCountQuery, map[string]any{"addr": addr})
	if err != nil {
		return 0
	}
	var data struct {
		DeveloperAddeds   []struct{ CampaignID string `json:"campaignId"` } `json:"developerAddeds"`
		DeveloperRemoveds []struct{ CampaignID string `json:"campaignId"` } `json:"developerRemoveds"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return 0
	}
	removed := make(map[string]struct{})
	for _, d := range data.DeveloperRemoveds {
		removed[d.CampaignID] = struct{}{}
	}
	active := make(map[string]struct{})
	for _, d := range data.DeveloperAddeds {
		if _, ok := removed[d.CampaignID]; !ok {
			active[d.CampaignID] = struct{}{}
		}
	}
	return int64(len(active))
}

// ──────────────────────────────────────────────
// Contributor dashboard 子图视图
// ──────────────────────────────────────────────

const sgContributorDashboardQuery = `
query CpContributorView($addr: Bytes!) {
  donateds(first: 1000, orderBy: blockNumber, orderDirection: desc, where: { contributor: $addr }) {
    campaignId amount blockTimestamp
  }
  refundClaimeds(first: 1000, where: { contributor: $addr }) {
    campaignId amount
  }
  crowdfundingLauncheds(first: 1000, orderBy: blockNumber, orderDirection: desc) {
    campaignId githubUrl target deadline
  }
  campaignFinalizeds(first: 1000) {
    campaignId successful
  }
}
`

type sgContributorEntry struct {
	CampaignID       uint64
	GithubURL        string
	TotalContributed *big.Int
	TotalRefunded    *big.Int
	CampaignState    string
	LastDonatedAt    *time.Time
}

type sgContributorView struct {
	All          []sgContributorEntry
	Refundable   []sgContributorEntry
	Fundraising  []sgContributorEntry
	Successful   []sgContributorEntry
	TotalDonated *big.Int
	OK           bool
}

func sgQueryContributorDashboard(ctx context.Context, h *handlers.Handlers, addr string) sgContributorView {
	if !sgAvailable(h) {
		return sgContributorView{}
	}
	raw, err := h.SubgraphCodePulse.Query(ctx, sgContributorDashboardQuery, map[string]any{"addr": addr})
	if err != nil {
		return sgContributorView{}
	}
	var data struct {
		Donateds []struct {
			CampaignID string `json:"campaignId"`
			Amount     string `json:"amount"`
			BlockTS    string `json:"blockTimestamp"`
		} `json:"donateds"`
		RefundClaimeds []struct {
			CampaignID string `json:"campaignId"`
			Amount     string `json:"amount"`
		} `json:"refundClaimeds"`
		CrowdfundingLauncheds []struct {
			CampaignID string `json:"campaignId"`
			GithubURL  string `json:"githubUrl"`
			Target     string `json:"target"`
			Deadline   string `json:"deadline"`
		} `json:"crowdfundingLauncheds"`
		CampaignFinalizeds []struct {
			CampaignID string `json:"campaignId"`
			Successful bool   `json:"successful"`
		} `json:"campaignFinalizeds"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return sgContributorView{}
	}

	campURLs := make(map[uint64]string)
	for _, c := range data.CrowdfundingLauncheds {
		cid, err := parseSubgraphUint(c.CampaignID)
		if err != nil {
			continue
		}
		campURLs[cid] = c.GithubURL
	}

	finalMap := make(map[uint64]bool)
	for _, f := range data.CampaignFinalizeds {
		cid, err := parseSubgraphUint(f.CampaignID)
		if err != nil {
			continue
		}
		finalMap[cid] = f.Successful
	}

	type contrib struct {
		total     *big.Int
		refund    *big.Int
		lastTS    *time.Time
	}
	contribs := make(map[uint64]*contrib)
	for _, d := range data.Donateds {
		cid, err := parseSubgraphUint(d.CampaignID)
		if err != nil {
			continue
		}
		e, ok := contribs[cid]
		if !ok {
			e = &contrib{total: new(big.Int), refund: new(big.Int)}
			contribs[cid] = e
		}
		if n, ok2 := new(big.Int).SetString(d.Amount, 10); ok2 {
			e.total.Add(e.total, n)
		}
		t := parseSubgraphTime(d.BlockTS)
		if t != nil && (e.lastTS == nil || t.After(*e.lastTS)) {
			e.lastTS = t
		}
	}
	for _, r := range data.RefundClaimeds {
		cid, err := parseSubgraphUint(r.CampaignID)
		if err != nil {
			continue
		}
		e, ok := contribs[cid]
		if !ok {
			e = &contrib{total: new(big.Int), refund: new(big.Int)}
			contribs[cid] = e
		}
		if n, ok2 := new(big.Int).SetString(r.Amount, 10); ok2 {
			e.refund.Add(e.refund, n)
		}
	}

	totalDonated := new(big.Int)
	var all, refundable, fundraising, successful []sgContributorEntry
	for cid, c := range contribs {
		state := "fundraising"
		if fin, ok := finalMap[cid]; ok {
			if fin {
				state = "successful"
			} else {
				state = "failed_refundable"
			}
		}
		entry := sgContributorEntry{
			CampaignID:       cid,
			GithubURL:        campURLs[cid],
			TotalContributed: c.total,
			TotalRefunded:    c.refund,
			CampaignState:    state,
			LastDonatedAt:    c.lastTS,
		}
		all = append(all, entry)
		totalDonated.Add(totalDonated, c.total)
		switch state {
		case "failed_refundable":
			refundable = append(refundable, entry)
		case "fundraising":
			fundraising = append(fundraising, entry)
		case "successful":
			successful = append(successful, entry)
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].CampaignID > all[j].CampaignID })

	return sgContributorView{
		All:          all,
		Refundable:   refundable,
		Fundraising:  fundraising,
		Successful:   successful,
		TotalDonated: totalDonated,
		OK:           true,
	}
}

// ──────────────────────────────────────────────
// Developer dashboard 子图视图
// ──────────────────────────────────────────────

const sgDeveloperDashboardQuery = `
query CpDeveloperView($addr: Bytes!) {
  developerAddeds(first: 500, where: { developer: $addr }) {
    campaignId blockTimestamp
  }
  developerRemoveds(first: 500, where: { developer: $addr }) {
    campaignId
  }
  milestoneShareClaimeds(first: 500, where: { developer: $addr }) {
    campaignId milestoneIndex amount blockTimestamp transactionHash
  }
  milestoneApproveds(first: 500) {
    campaignId milestoneIndex
  }
  crowdfundingLauncheds(first: 1000) {
    campaignId proposalId githubUrl target deadline organizer roundIndex blockNumber blockTimestamp transactionHash
  }
  campaignFinalizeds(first: 1000) {
    campaignId successful
  }
}
`

type sgDeveloperView struct {
	Campaigns         []models.CPCampaign
	Claims            []models.CPMilestoneClaim
	TotalClaimedWei   string
	PendingMilestones []models.CPCampaignMilestone
	OK                bool
}

func sgQueryDeveloperDashboard(ctx context.Context, h *handlers.Handlers, addr string) sgDeveloperView {
	if !sgAvailable(h) {
		return sgDeveloperView{}
	}
	raw, err := h.SubgraphCodePulse.Query(ctx, sgDeveloperDashboardQuery, map[string]any{"addr": addr})
	if err != nil {
		return sgDeveloperView{}
	}
	var data struct {
		DeveloperAddeds   []struct {
			CampaignID string `json:"campaignId"`
			BlockTS    string `json:"blockTimestamp"`
		} `json:"developerAddeds"`
		DeveloperRemoveds []struct{ CampaignID string `json:"campaignId"` } `json:"developerRemoveds"`
		MilestoneShareClaimeds []struct {
			CampaignID     string `json:"campaignId"`
			MilestoneIndex string `json:"milestoneIndex"`
			Amount         string `json:"amount"`
			BlockTS        string `json:"blockTimestamp"`
			TxHash         string `json:"transactionHash"`
		} `json:"milestoneShareClaimeds"`
		MilestoneApproveds []struct {
			CampaignID     string `json:"campaignId"`
			MilestoneIndex string `json:"milestoneIndex"`
		} `json:"milestoneApproveds"`
		CrowdfundingLauncheds []struct {
			CampaignID string `json:"campaignId"`
			ProposalID string `json:"proposalId"`
			GithubURL  string `json:"githubUrl"`
			Target     string `json:"target"`
			Deadline   string `json:"deadline"`
			Organizer  string `json:"organizer"`
			RoundIndex string `json:"roundIndex"`
			BlockNum   string `json:"blockNumber"`
			BlockTS    string `json:"blockTimestamp"`
			TxHash     string `json:"transactionHash"`
		} `json:"crowdfundingLauncheds"`
		CampaignFinalizeds []struct {
			CampaignID string `json:"campaignId"`
			Successful bool   `json:"successful"`
		} `json:"campaignFinalizeds"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return sgDeveloperView{}
	}

	removed := make(map[string]struct{})
	for _, d := range data.DeveloperRemoveds {
		removed[d.CampaignID] = struct{}{}
	}
	activeCIDs := make(map[uint64]struct{})
	for _, d := range data.DeveloperAddeds {
		if _, ok := removed[d.CampaignID]; ok {
			continue
		}
		cid, err := parseSubgraphUint(d.CampaignID)
		if err != nil {
			continue
		}
		activeCIDs[cid] = struct{}{}
	}

	finalMap := make(map[uint64]bool)
	for _, f := range data.CampaignFinalizeds {
		cid, err := parseSubgraphUint(f.CampaignID)
		if err != nil {
			continue
		}
		finalMap[cid] = f.Successful
	}

	campInfo := make(map[uint64]struct {
		ProposalID  string
		GithubURL   string
		Target      string
		Deadline    string
		Organizer   string
		RoundIndex  string
		BlockNum    string
		BlockTS     string
		TxHash      string
	})
	for _, l := range data.CrowdfundingLauncheds {
		cid, err := parseSubgraphUint(l.CampaignID)
		if err != nil {
			continue
		}
		campInfo[cid] = struct {
			ProposalID  string
			GithubURL   string
			Target      string
			Deadline    string
			Organizer   string
			RoundIndex  string
			BlockNum    string
			BlockTS     string
			TxHash      string
		}{l.ProposalID, l.GithubURL, l.Target, l.Deadline, l.Organizer, l.RoundIndex, l.BlockNum, l.BlockTS, l.TxHash}
	}

	var campaigns []models.CPCampaign
	for cid := range activeCIDs {
		info, ok := campInfo[cid]
		if !ok {
			continue
		}
		pid, _ := parseSubgraphUint(info.ProposalID)
		state := "fundraising"
		stateCode := 1
		if fin, ok := finalMap[cid]; ok {
			if fin {
				state = "successful"
				stateCode = 2
			} else {
				state = "failed_refundable"
				stateCode = 3
			}
		}
		campaigns = append(campaigns, models.CPCampaign{
			CampaignID:       cid,
			ProposalID:       pid,
			OrganizerAddress: strings.ToLower(info.Organizer),
			GithubURL:        info.GithubURL,
			TargetWei:        strings.TrimSpace(info.Target),
			DeadlineAt:       time.Unix(mustParseInt64(info.Deadline), 0).UTC(),
			State:            state,
			StateCode:        stateCode,
			RoundIndex:       int(mustParseInt64(info.RoundIndex)),
			LaunchedTxHash:   info.TxHash,
			LaunchedBlockNumber: mustParseBN(info.BlockNum),
			LaunchedAt:       time.Unix(parseBlockTS(info.BlockTS), 0).UTC(),
			AmountRaisedWei:        "0",
			TotalWithdrawnWei:      "0",
			UnclaimedRefundPoolWei: "0",
			CreatedAt:              time.Now().UTC(),
			UpdatedAt:              time.Now().UTC(),
		})
	}

	approvedSet := make(map[string]struct{})
	for _, m := range data.MilestoneApproveds {
		key := m.CampaignID + "-" + m.MilestoneIndex
		approvedSet[key] = struct{}{}
	}

	totalClaimed := new(big.Int)
	var claims []models.CPMilestoneClaim
	for _, cl := range data.MilestoneShareClaimeds {
		cid, err := parseSubgraphUint(cl.CampaignID)
		if err != nil {
			continue
		}
		mi := int(mustParseInt64(cl.MilestoneIndex))
		t := parseSubgraphTime(cl.BlockTS)
		txh := cl.TxHash
		var claimedAt time.Time
		if t != nil {
			claimedAt = *t
		}
		claims = append(claims, models.CPMilestoneClaim{
			CampaignID:       cid,
			MilestoneIndex:   mi,
			DeveloperAddress: addr,
			ClaimedAmountWei: strings.TrimSpace(cl.Amount),
			ClaimedTxHash:    txh,
			ClaimedAt:        claimedAt,
		})
		if n, ok := new(big.Int).SetString(cl.Amount, 10); ok {
			totalClaimed.Add(totalClaimed, n)
		}
	}

	var pendingMilestones []models.CPCampaignMilestone
	for cid := range activeCIDs {
		if fin, ok := finalMap[cid]; !ok || fin {
			for mi := 0; mi < 10; mi++ {
				key := strings.TrimSpace(big.NewInt(int64(cid)).String()) + "-" + strings.TrimSpace(big.NewInt(int64(mi)).String())
				if _, approved := approvedSet[key]; !approved {
					pendingMilestones = append(pendingMilestones, models.CPCampaignMilestone{
						CampaignID:     cid,
						MilestoneIndex: mi,
						Approved:       false,
					})
				}
			}
		}
	}

	return sgDeveloperView{
		Campaigns:         campaigns,
		Claims:            claims,
		TotalClaimedWei:   totalClaimed.String(),
		PendingMilestones: pendingMilestones,
		OK:                true,
	}
}

// ──────────────────────────────────────────────
// Admin dashboard 子图视图
// ──────────────────────────────────────────────

func sgQueryAdminDashboard(ctx context.Context, h *handlers.Handlers) (pendingProposals []models.CPProposal, pendingRounds []models.CPProposal, liveCampaigns []models.CPCampaign, ok bool) {
	allProposals, pOK := sgQueryAllProposals(ctx, h)
	if !pOK {
		return nil, nil, nil, false
	}
	for _, p := range allProposals {
		if p.Status == "pending_review" {
			pendingProposals = append(pendingProposals, p)
		}
		if p.RoundReviewState != nil && *p.RoundReviewState == "round_review_pending" {
			pendingRounds = append(pendingRounds, p)
		}
	}
	if allCampaigns, cOK := sgQueryAllCampaigns(ctx, h); cOK {
		for _, c := range allCampaigns {
			if c.State == "fundraising" {
				liveCampaigns = append(liveCampaigns, c)
			}
		}
	}
	return pendingProposals, pendingRounds, liveCampaigns, true
}
