package codepulse

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
)

// 与 indexer/codepulse_subgraph.go 中字符串常量一致，用于工作台只读展示。
const (
	sgStPendingReview       = "pending_review"
	sgStApproved            = "approved"
	sgStRejected            = "rejected"
	sgStRoundReviewRejected = "round_review_rejected"
	sgStSettled             = "settled"
	sgRsPending             = "round_review_pending"
	sgRsApproved            = "round_review_approved"
	sgRsRejected            = "round_review_rejected"
)

// OrganizerProposalsSubgraphView 发起人工作台只读：以子图事件推导 status / round_review_*（动作预检、tx/build 仍以 PG 为准）。
func OrganizerProposalsSubgraphView(ctx context.Context, h *handlers.Handlers, organizerLower string, fromPG []models.CPProposal) ([]models.CPProposal, string) {
	if h == nil || h.SubgraphCodePulse == nil || !h.SubgraphCodePulse.Configured() {
		return fromPG, ""
	}
	if !common.IsHexAddress(organizerLower) {
		return fromPG, ""
	}
	org := common.HexToAddress(organizerLower)

	raw, err := h.SubgraphCodePulse.Query(ctx, cpSubgraphProposalSubmittedByOrganizer, map[string]any{"org": org.Hex()})
	if err != nil {
		return fromPG, ""
	}
	var sub0 struct {
		ProposalSubmitteds []struct {
			sgPropSubmitted
			BlockNumber string `json:"blockNumber"`
		} `json:"proposalSubmitteds"`
	}
	if json.Unmarshal(raw, &sub0) != nil || len(sub0.ProposalSubmitteds) == 0 {
		return fromPG, ""
	}

	ids := make([]string, 0, len(sub0.ProposalSubmitteds))
	pidSeen := make(map[uint64]struct{})
	for _, s := range sub0.ProposalSubmitteds {
		pid, err := parseSubgraphUint(s.ProposalID)
		if err != nil {
			continue
		}
		if _, ok := pidSeen[pid]; ok {
			continue
		}
		pidSeen[pid] = struct{}{}
		ids = append(ids, strings.TrimSpace(s.ProposalID))
	}
	if len(ids) == 0 {
		return fromPG, ""
	}

	rawPipe, err := h.SubgraphCodePulse.Query(ctx, cpSubgraphInitiatorPipeline, map[string]any{"pids": ids})
	if err != nil {
		return fromPG, ""
	}
	var pipe sgInitiatorPipeEvents
	if json.Unmarshal(rawPipe, &pipe) != nil {
		return fromPG, ""
	}

	launchByCID := make(map[uint64]sgEvLaunch)
	for _, l := range pipe.CrowdfundingLauncheds {
		cid, err := parseSubgraphUint(l.CampaignID)
		if err != nil {
			continue
		}
		launchByCID[cid] = l
	}
	cids := make([]string, 0, len(launchByCID))
	for cid := range launchByCID {
		cids = append(cids, strconv.FormatUint(cid, 10))
	}
	finalByCID := make(map[uint64]sgEvCampFinalize)
	if len(cids) > 0 {
		if rawF, err := h.SubgraphCodePulse.Query(ctx, cpSubgraphCampaignFinalized, map[string]any{"cids": cids}); err == nil {
			var fin struct {
				CampaignFinalizeds []sgEvCampFinalize `json:"campaignFinalizeds"`
			}
			if json.Unmarshal(rawF, &fin) == nil {
				for _, f := range fin.CampaignFinalizeds {
					cid, err := parseSubgraphUint(f.CampaignID)
					if err != nil {
						continue
					}
					finalByCID[cid] = f
				}
			}
		}
	}

	pgByID := make(map[uint64]models.CPProposal, len(fromPG))
	for _, p := range fromPG {
		pgByID[p.ProposalID] = p
	}

	submitByPID := make(map[uint64]sgPropSubmitted)
	for _, row := range sub0.ProposalSubmitteds {
		pid, err := parseSubgraphUint(row.ProposalID)
		if err != nil {
			continue
		}
		if _, ok := submitByPID[pid]; !ok {
			submitByPID[pid] = row.sgPropSubmitted
		}
	}

	note := ""
	out := make([]models.CPProposal, 0, len(pidSeen)+len(fromPG))
	used := make(map[uint64]struct{})

	for _, pid := range sortedUintKeys(pidSeen) {
		st := simulateProposalStateFromSubgraph(pid, pipe, launchByCID, finalByCID)
		base, ok := pgByID[pid]
		if !ok {
			s := submitByPID[pid]
			ts := parseSubgraphTime(s.BlockTimestamp)
			txh := s.TxHash
			base = models.CPProposal{
				ProposalID:       pid,
				OrganizerAddress: organizerLower,
				GithubURL:        s.GithubURL,
				TargetWei:        strings.TrimSpace(s.Target),
				DurationSeconds:  mustParseInt64(s.Duration),
				SubmittedTxHash:  &txh,
				SubmittedAt:      ts,
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
			}
			note = "subgraph_view"
		} else {
			note = "subgraph_view"
		}
		applySimulatedState(&base, st)
		out = append(out, base)
		used[pid] = struct{}{}
	}

	for _, p := range fromPG {
		if _, ok := used[p.ProposalID]; ok {
			continue
		}
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ProposalID > out[j].ProposalID })
	return out, note
}

type propSimState struct {
	status     string
	statusCode int
	round      string // 空表示无轮次审核态
	roundCode  int    // 与 round 同时有效；0 表示未置位（与 PG 一致时用 1/2/3）
	hasRound   bool
}

type sgEvPropReview struct {
	ProposalID  string `json:"proposalId"`
	Approved    bool   `json:"approved"`
	BlockNumber string `json:"blockNumber"`
}

type sgEvFRSubmit struct {
	ProposalID  string `json:"proposalId"`
	BlockNumber string `json:"blockNumber"`
}

type sgEvFRReview struct {
	ProposalID  string `json:"proposalId"`
	Approved    bool   `json:"approved"`
	BlockNumber string `json:"blockNumber"`
}

type sgEvLaunch struct {
	ProposalID  string `json:"proposalId"`
	CampaignID  string `json:"campaignId"`
	BlockNumber string `json:"blockNumber"`
}

type sgEvCampFinalize struct {
	CampaignID  string `json:"campaignId"`
	Successful  bool   `json:"successful"`
	BlockNumber string `json:"blockNumber"`
}

type sgInitiatorPipeEvents struct {
	ProposalRevieweds               []sgEvPropReview `json:"proposalRevieweds"`
	FundingRoundSubmittedForReviews []sgEvFRSubmit   `json:"fundingRoundSubmittedForReviews"`
	FundingRoundRevieweds           []sgEvFRReview   `json:"fundingRoundRevieweds"`
	CrowdfundingLauncheds           []sgEvLaunch     `json:"crowdfundingLauncheds"`
}

type timelineEv struct {
	block uint64
	order int
	kind  int // 1=propR 2=frS 3=frR 4=launch 5=finOk
	appr  bool
	ok    bool
	cid   uint64
}

const cpSubgraphInitiatorPipeline = `
query CpInitiatorPipeline($pids: [BigInt!]!) {
  proposalRevieweds(first: 500, orderBy: blockNumber, orderDirection: asc, where: { proposalId_in: $pids }) {
    proposalId
    approved
    blockNumber
  }
  fundingRoundSubmittedForReviews(first: 500, orderBy: blockNumber, orderDirection: asc, where: { proposalId_in: $pids }) {
    proposalId
    blockNumber
  }
  fundingRoundRevieweds(first: 500, orderBy: blockNumber, orderDirection: asc, where: { proposalId_in: $pids }) {
    proposalId
    approved
    blockNumber
  }
  crowdfundingLauncheds(first: 500, orderBy: blockNumber, orderDirection: asc, where: { proposalId_in: $pids }) {
    proposalId
    campaignId
    blockNumber
  }
}
`

const cpSubgraphCampaignFinalized = `
query CpCampFin($cids: [BigInt!]!) {
  campaignFinalizeds(first: 500, orderBy: blockNumber, orderDirection: asc, where: { campaignId_in: $cids }) {
    campaignId
    successful
    blockNumber
  }
}
`

func simulateProposalStateFromSubgraph(
	pid uint64,
	pipe sgInitiatorPipeEvents,
	launchByCID map[uint64]sgEvLaunch,
	finalByCID map[uint64]sgEvCampFinalize,
) propSimState {
	var evs []timelineEv
	for _, r := range pipe.ProposalRevieweds {
		rpid, err := parseSubgraphUint(r.ProposalID)
		if err != nil || rpid != pid {
			continue
		}
		evs = append(evs, timelineEv{block: mustParseBN(r.BlockNumber), order: 1, kind: 1, appr: r.Approved})
	}
	for _, r := range pipe.FundingRoundSubmittedForReviews {
		rpid, err := parseSubgraphUint(r.ProposalID)
		if err != nil || rpid != pid {
			continue
		}
		evs = append(evs, timelineEv{block: mustParseBN(r.BlockNumber), order: 2, kind: 2})
	}
	for _, r := range pipe.FundingRoundRevieweds {
		rpid, err := parseSubgraphUint(r.ProposalID)
		if err != nil || rpid != pid {
			continue
		}
		evs = append(evs, timelineEv{block: mustParseBN(r.BlockNumber), order: 3, kind: 3, appr: r.Approved})
	}
	for _, r := range pipe.CrowdfundingLauncheds {
		rpid, err := parseSubgraphUint(r.ProposalID)
		if err != nil || rpid != pid {
			continue
		}
		cid, err := parseSubgraphUint(r.CampaignID)
		if err != nil {
			continue
		}
		evs = append(evs, timelineEv{block: mustParseBN(r.BlockNumber), order: 4, kind: 4, cid: cid})
	}
	for cid, l := range launchByCID {
		rpid, err := parseSubgraphUint(l.ProposalID)
		if err != nil || rpid != pid {
			continue
		}
		f, ok := finalByCID[cid]
		if !ok || !f.Successful {
			continue
		}
		fb := mustParseBN(f.BlockNumber)
		lb := mustParseBN(l.BlockNumber)
		if fb < lb {
			continue
		}
		evs = append(evs, timelineEv{block: fb, order: 5, kind: 5, ok: true, cid: cid})
	}

	sort.Slice(evs, func(i, j int) bool {
		if evs[i].block != evs[j].block {
			return evs[i].block < evs[j].block
		}
		return evs[i].order < evs[j].order
	})

	s := propSimState{status: sgStPendingReview, statusCode: 1}

	for _, e := range evs {
		switch e.kind {
		case 1:
			if e.appr {
				s.status = sgStApproved
				s.statusCode = 2
				s.hasRound = false
				s.round = ""
				s.roundCode = 0
			} else {
				s.status = sgStRejected
				s.statusCode = 3
				s.hasRound = false
				s.round = ""
				s.roundCode = 0
			}
		case 2:
			s.status = sgStApproved
			s.statusCode = 2
			s.hasRound = true
			s.round = sgRsPending
			s.roundCode = 1
		case 3:
			if e.appr {
				s.status = sgStApproved
				s.statusCode = 2
				s.hasRound = true
				s.round = sgRsApproved
				s.roundCode = 2
			} else {
				s.status = sgStRoundReviewRejected
				s.statusCode = 6
				s.hasRound = true
				s.round = sgRsRejected
				s.roundCode = 3
			}
		case 4:
			s.status = sgStApproved
			s.statusCode = 2
			s.hasRound = false
			s.round = ""
			s.roundCode = 0
		case 5:
			s.status = sgStSettled
			s.statusCode = 7
			s.hasRound = false
			s.round = ""
			s.roundCode = 0
		}
	}
	return s
}

func applySimulatedState(p *models.CPProposal, st propSimState) {
	p.Status = st.status
	p.StatusCode = st.statusCode
	if !st.hasRound {
		p.RoundReviewState = nil
		p.RoundReviewStateCode = nil
		return
	}
	r := st.round
	c := st.roundCode
	p.RoundReviewState = &r
	p.RoundReviewStateCode = &c
}

func mustParseBN(s string) uint64 {
	n, err := parseSubgraphUint(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func sortedUintKeys(m map[uint64]struct{}) []uint64 {
	out := make([]uint64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// OrganizerFundraisingCampaignsSubgraphView 募资中活动：子图上有 Launch 且尚无 Finalize 的 campaign（只读展示）。
func OrganizerFundraisingCampaignsSubgraphView(ctx context.Context, h *handlers.Handlers, organizerLower string, fromPG []models.CPCampaign) ([]models.CPCampaign, string) {
	if h == nil || h.SubgraphCodePulse == nil || !h.SubgraphCodePulse.Configured() {
		return filterFundraisingPG(fromPG), ""
	}
	if !common.IsHexAddress(organizerLower) {
		return filterFundraisingPG(fromPG), ""
	}
	org := common.HexToAddress(organizerLower)
	raw, err := h.SubgraphCodePulse.Query(ctx, cpSubgraphCrowdfundingLaunchedByOrganizer, map[string]any{"org": org.Hex()})
	if err != nil {
		return filterFundraisingPG(fromPG), ""
	}
	var data struct {
		CrowdfundingLauncheds []struct {
			ProposalID   string `json:"proposalId"`
			CampaignID   string `json:"campaignId"`
			GithubURL    string `json:"githubUrl"`
			Target       string `json:"target"`
			Deadline     string `json:"deadline"`
			RoundIndex   string `json:"roundIndex"`
			BlockNumber  string `json:"blockNumber"`
			BlockTS      string `json:"blockTimestamp"`
			TransactionH string `json:"transactionHash"`
		} `json:"crowdfundingLauncheds"`
	}
	if json.Unmarshal(raw, &data) != nil || len(data.CrowdfundingLauncheds) == 0 {
		return filterFundraisingPG(fromPG), ""
	}

	cids := make([]string, 0, len(data.CrowdfundingLauncheds))
	cidSeen := make(map[uint64]struct{})
	for _, l := range data.CrowdfundingLauncheds {
		cid, err := parseSubgraphUint(l.CampaignID)
		if err != nil {
			continue
		}
		if _, ok := cidSeen[cid]; ok {
			continue
		}
		cidSeen[cid] = struct{}{}
		cids = append(cids, strings.TrimSpace(l.CampaignID))
	}
	finalized := make(map[uint64]struct{})
	if len(cids) > 0 {
		if rawF, err := h.SubgraphCodePulse.Query(ctx, cpSubgraphCampaignFinalized, map[string]any{"cids": cids}); err == nil {
			var fin struct {
				CampaignFinalizeds []struct {
					CampaignID string `json:"campaignId"`
				} `json:"campaignFinalizeds"`
			}
			if json.Unmarshal(rawF, &fin) == nil {
				for _, f := range fin.CampaignFinalizeds {
					cid, err := parseSubgraphUint(f.CampaignID)
					if err != nil {
						continue
					}
					finalized[cid] = struct{}{}
				}
			}
		}
	}

	pgByCID := make(map[uint64]models.CPCampaign, len(fromPG))
	for _, c := range fromPG {
		pgByCID[c.CampaignID] = c
	}

	out := make([]models.CPCampaign, 0)
	note := ""
	seenCamp := make(map[uint64]struct{})
	for _, l := range data.CrowdfundingLauncheds {
		cid, err := parseSubgraphUint(l.CampaignID)
		if err != nil {
			continue
		}
		if _, dup := seenCamp[cid]; dup {
			continue
		}
		seenCamp[cid] = struct{}{}
		if _, fin := finalized[cid]; fin {
			continue
		}
		pid, err := parseSubgraphUint(l.ProposalID)
		if err != nil {
			continue
		}
		note = "subgraph_view"
		if row, ok := pgByCID[cid]; ok {
			cp := row
			cp.State = "fundraising"
			cp.StateCode = 1
			out = append(out, cp)
			continue
		}
		deadlineUnix := mustParseInt64(l.Deadline)
		launchedAt := time.Unix(parseBlockTS(l.BlockTS), 0).UTC()
		ri := int(mustParseInt64(l.RoundIndex))
		txh := l.TransactionH
		out = append(out, models.CPCampaign{
			CampaignID:             cid,
			ProposalID:             pid,
			RoundIndex:             ri,
			OrganizerAddress:       organizerLower,
			GithubURL:              l.GithubURL,
			TargetWei:              strings.TrimSpace(l.Target),
			DeadlineAt:             time.Unix(deadlineUnix, 0).UTC(),
			AmountRaisedWei:        "0",
			TotalWithdrawnWei:      "0",
			UnclaimedRefundPoolWei: "0",
			State:                  "fundraising",
			StateCode:              1,
			DonorCount:             0,
			DeveloperCount:         0,
			LaunchedTxHash:         txh,
			LaunchedBlockNumber:    mustParseBN(l.BlockNumber),
			LaunchedAt:             launchedAt,
			CreatedAt:              time.Now().UTC(),
			UpdatedAt:              time.Now().UTC(),
		})
	}
	return out, note
}

func parseBlockTS(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return sec
}

func filterFundraisingPG(rows []models.CPCampaign) []models.CPCampaign {
	out := make([]models.CPCampaign, 0)
	for _, ca := range rows {
		if ca.State == "fundraising" {
			out = append(out, ca)
		}
	}
	return out
}

const cpSubgraphCrowdfundingLaunchedByOrganizer = `
query CpOrgLaunches($org: Bytes!) {
  crowdfundingLauncheds(
    first: 100
    orderBy: blockNumber
    orderDirection: desc
    where: { organizer: $org }
  ) {
    proposalId
    campaignId
    githubUrl
    target
    deadline
    roundIndex
    blockNumber
    blockTimestamp
    transactionHash
  }
}
`
