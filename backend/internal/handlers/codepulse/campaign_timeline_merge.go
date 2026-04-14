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

	"gorm.io/gorm"
)

// sgTimelineCampMetaQuery 单笔/低频事件：Launch + Finalize 一次拉齐。
const sgTimelineCampMetaQuery = `
query CpTimelineCampMeta($cid: BigInt!) {
  crowdfundingLauncheds(where: { campaignId: $cid }, first: 100, orderBy: blockNumber, orderDirection: asc) {
    id proposalId campaignId organizer githubUrl target deadline roundIndex
    blockNumber blockTimestamp transactionHash
  }
  campaignFinalizeds(where: { campaignId: $cid }, first: 100, orderBy: blockNumber, orderDirection: asc) {
    id campaignId successful blockNumber blockTimestamp transactionHash
  }
}
`

const sgTimelineDonatedPage = `
query CpTimelineDons($cid: BigInt!, $skip: Int!) {
  donateds(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id campaignId contributor amount blockNumber blockTimestamp transactionHash
  }
}
`

const sgTimelineRefundPage = `
query CpTimelineRef($cid: BigInt!, $skip: Int!) {
  refundClaimeds(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id campaignId contributor amount blockNumber blockTimestamp transactionHash
  }
}
`

const sgTimelineDevAddedPage = `
query CpTimelineDA($cid: BigInt!, $skip: Int!) {
  developerAddeds(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id campaignId developer blockNumber blockTimestamp transactionHash
  }
}
`

const sgTimelineDevRemovedPage = `
query CpTimelineDR($cid: BigInt!, $skip: Int!) {
  developerRemoveds(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id campaignId developer blockNumber blockTimestamp transactionHash
  }
}
`

const sgTimelineMilestoneApprovedPage = `
query CpTimelineMA($cid: BigInt!, $skip: Int!) {
  milestoneApproveds(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id campaignId milestoneIndex blockNumber blockTimestamp transactionHash
  }
}
`

const sgTimelineMilestoneSharePage = `
query CpTimelineMSC($cid: BigInt!, $skip: Int!) {
  milestoneShareClaimeds(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id campaignId milestoneIndex developer amount blockNumber blockTimestamp transactionHash
  }
}
`

const sgTimelineStaleFundsPage = `
query CpTimelineSFS($cid: BigInt!, $skip: Int!) {
  staleFundsSwepts(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id campaignId amount blockNumber blockTimestamp transactionHash
  }
}
`

func resolveTimelineMeta(ctx context.Context, h *handlers.Handlers) (chainID uint64, contract string) {
	if h != nil && h.CodePulse != nil {
		contract = strings.ToLower(h.CodePulse.Address().Hex())
	}
	if h != nil && h.Chain != nil {
		if cid, err := h.Chain.Eth().ChainID(ctx); err == nil {
			chainID = cid.Uint64()
		}
	}
	return chainID, contract
}

func timelineScalarUint64(raw json.RawMessage) (uint64, bool) {
	s, err := parseGraphQLScalarToString(raw)
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n, err == nil
}

func timelineScalarUnixUTC(raw json.RawMessage) (time.Time, bool) {
	s, err := parseGraphQLScalarToString(raw)
	if err != nil {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(sec, 0).UTC(), true
}

func timelineNormTx(raw json.RawMessage) (string, bool) {
	txStr, err := parseGraphQLScalarToString(raw)
	if err != nil || strings.TrimSpace(txStr) == "" {
		return "", false
	}
	txNorm := normalizeAddress(txStr)
	if !strings.HasPrefix(txNorm, "0x") {
		txNorm = "0x" + strings.TrimPrefix(txNorm, "0x")
	}
	return txNorm, true
}

func timelineEntityKey(evName, txHash string, logIndex int) string {
	return evName + ":" + strings.TrimPrefix(strings.ToLower(strings.TrimSpace(txHash)), "0x") + ":" + strconv.Itoa(logIndex)
}

func buildSyntheticCPEventLog(
	chainID uint64,
	contract string,
	eventName string,
	idHex string,
	txNorm string,
	blockNum uint64,
	ts time.Time,
	payload map[string]any,
	proposalID *uint64,
	campaignID *uint64,
	wallet *string,
) (models.CPEventLog, bool) {
	_, logIdx, ok := parseSubgraphDonatedEntityID(idHex)
	if !ok {
		return models.CPEventLog{}, false
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return models.CPEventLog{}, false
	}
	ek := timelineEntityKey(eventName, txNorm, logIdx)
	return models.CPEventLog{
		ID:              0,
		ChainID:         chainID,
		ContractAddress: contract,
		EventName:       eventName,
		ProposalID:      proposalID,
		CampaignID:      campaignID,
		WalletAddress:   wallet,
		EntityKey:       &ek,
		TxHash:          txNorm,
		LogIndex:        logIdx,
		BlockNumber:     blockNum,
		BlockTimestamp:  ts.UTC(),
		Payload:         models.JSONB(rawPayload),
		Source:          "subgraph",
		CreatedAt:       ts.UTC(),
	}, true
}

func scalarString(raw json.RawMessage) (string, bool) {
	s, err := parseGraphQLScalarToString(raw)
	if err != nil {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

// dedupEventLogsByTxLog 同一 (tx_hash, log_index) 只保留一条。
func dedupEventLogsByTxLog(rows []models.CPEventLog) []models.CPEventLog {
	seen := make(map[string]struct{})
	out := make([]models.CPEventLog, 0, len(rows))
	for _, e := range rows {
		k := donationDedupKey(e.TxHash, e.LogIndex)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, e)
	}
	return out
}

func sortTimelineEventsDesc(rows []models.CPEventLog) {
	for i := range rows {
		if !strings.HasPrefix(rows[i].ContractAddress, "0x") && rows[i].ContractAddress != "" {
			rows[i].ContractAddress = "0x" + strings.TrimPrefix(rows[i].ContractAddress, "0x")
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].BlockNumber != rows[j].BlockNumber {
			return rows[i].BlockNumber > rows[j].BlockNumber
		}
		return rows[i].LogIndex > rows[j].LogIndex
	})
}

// sgFetchCampaignTimelineFromSubgraph 拉取与子图 schema 中 campaignId 相关的活动事件，并规范为 CPEventLog 形态。
func sgFetchCampaignTimelineFromSubgraph(ctx context.Context, h *handlers.Handlers, campaignID uint64) ([]models.CPEventLog, bool) {
	if !sgAvailable(h) {
		return nil, false
	}
	cidStr := strconv.FormatUint(campaignID, 10)
	chainID, contract := resolveTimelineMeta(ctx, h)
	cidPtr := campaignID

	rawMeta, err := h.SubgraphCodePulse.Query(ctx, sgTimelineCampMetaQuery, map[string]any{"cid": cidStr})
	if err != nil {
		return nil, false
	}
	var metaWrap struct {
		CrowdfundingLauncheds []struct {
			ID               string          `json:"id"`
			ProposalID       json.RawMessage `json:"proposalId"`
			CampaignID       json.RawMessage `json:"campaignId"`
			Organizer        json.RawMessage `json:"organizer"`
			GithubURL        string          `json:"githubUrl"`
			Target           json.RawMessage `json:"target"`
			Deadline         json.RawMessage `json:"deadline"`
			RoundIndex       json.RawMessage `json:"roundIndex"`
			BlockNumber      json.RawMessage `json:"blockNumber"`
			BlockTimestamp   json.RawMessage `json:"blockTimestamp"`
			TransactionHash  json.RawMessage `json:"transactionHash"`
		} `json:"crowdfundingLauncheds"`
		CampaignFinalizeds []struct {
			ID              string          `json:"id"`
			CampaignID      json.RawMessage `json:"campaignId"`
			Successful      bool            `json:"successful"`
			BlockNumber     json.RawMessage `json:"blockNumber"`
			BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
			TransactionHash json.RawMessage `json:"transactionHash"`
		} `json:"campaignFinalizeds"`
	}
	if json.Unmarshal(rawMeta, &metaWrap) != nil {
		return nil, false
	}

	var all []models.CPEventLog

	for _, r := range metaWrap.CrowdfundingLauncheds {
		txNorm, ok := timelineNormTx(r.TransactionHash)
		if !ok {
			continue
		}
		bn, ok := timelineScalarUint64(r.BlockNumber)
		if !ok {
			continue
		}
		ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
		if !ok {
			continue
		}
		pidStr, ok := scalarString(r.ProposalID)
		if !ok {
			continue
		}
		pid64, err := strconv.ParseUint(pidStr, 10, 64)
		if err != nil {
			continue
		}
		orgStr, ok := scalarString(r.Organizer)
		if !ok {
			continue
		}
		w := normalizeAddress(orgStr)
		tgt, _ := scalarString(r.Target)
		dl, _ := scalarString(r.Deadline)
		ri, _ := scalarString(r.RoundIndex)
		cidS, _ := scalarString(r.CampaignID)
		payload := map[string]any{
			"id": r.ID, "proposalId": pidStr, "campaignId": cidS, "organizer": w,
			"githubUrl": r.GithubURL, "target": tgt, "deadline": dl, "roundIndex": ri,
			"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
		}
		ev, ok := buildSyntheticCPEventLog(chainID, contract, "CrowdfundingLaunched", r.ID, txNorm, bn, ts, payload, &pid64, &cidPtr, &w)
		if ok {
			all = append(all, ev)
		}
	}

	for _, r := range metaWrap.CampaignFinalizeds {
		txNorm, ok := timelineNormTx(r.TransactionHash)
		if !ok {
			continue
		}
		bn, ok := timelineScalarUint64(r.BlockNumber)
		if !ok {
			continue
		}
		ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
		if !ok {
			continue
		}
		cidS, _ := scalarString(r.CampaignID)
		payload := map[string]any{
			"id": r.ID, "campaignId": cidS, "successful": r.Successful,
			"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
		}
		ev, ok := buildSyntheticCPEventLog(chainID, contract, "CampaignFinalized", r.ID, txNorm, bn, ts, payload, nil, &cidPtr, nil)
		if ok {
			all = append(all, ev)
		}
	}

	appendPagedDonations := func(gql string) bool {
		skip := 0
		for {
			raw, err := h.SubgraphCodePulse.Query(ctx, gql, map[string]any{"cid": cidStr, "skip": skip})
			if err != nil {
				return false
			}
			var wrap struct {
				Donateds []struct {
					ID              string          `json:"id"`
					CampaignID      json.RawMessage `json:"campaignId"`
					Contributor     json.RawMessage `json:"contributor"`
					Amount          json.RawMessage `json:"amount"`
					BlockNumber     json.RawMessage `json:"blockNumber"`
					BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
					TransactionHash json.RawMessage `json:"transactionHash"`
				} `json:"donateds"`
			}
			if json.Unmarshal(raw, &wrap) != nil {
				return false
			}
			if len(wrap.Donateds) == 0 {
				return true
			}
			for _, r := range wrap.Donateds {
				txNorm, ok := timelineNormTx(r.TransactionHash)
				if !ok {
					continue
				}
				bn, ok := timelineScalarUint64(r.BlockNumber)
				if !ok {
					continue
				}
				ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
				if !ok {
					continue
				}
				contrib, ok := scalarString(r.Contributor)
				if !ok {
					continue
				}
				w := normalizeAddress(contrib)
				amt, _ := scalarString(r.Amount)
				cidS, _ := scalarString(r.CampaignID)
				payload := map[string]any{
					"id": r.ID, "campaignId": cidS, "contributor": w, "amount": amt,
					"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
				}
				ev, ok := buildSyntheticCPEventLog(chainID, contract, "Donated", r.ID, txNorm, bn, ts, payload, nil, &cidPtr, &w)
				if ok {
					all = append(all, ev)
				}
			}
			skip += len(wrap.Donateds)
			if len(wrap.Donateds) < 1000 {
				return true
			}
		}
	}
	if !appendPagedDonations(sgTimelineDonatedPage) {
		return nil, false
	}

	appendPagedRefunds := func() bool {
		skip := 0
		for {
			raw, err := h.SubgraphCodePulse.Query(ctx, sgTimelineRefundPage, map[string]any{"cid": cidStr, "skip": skip})
			if err != nil {
				return false
			}
			var wrap struct {
				Rows []struct {
					ID              string          `json:"id"`
					CampaignID      json.RawMessage `json:"campaignId"`
					Contributor     json.RawMessage `json:"contributor"`
					Amount          json.RawMessage `json:"amount"`
					BlockNumber     json.RawMessage `json:"blockNumber"`
					BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
					TransactionHash json.RawMessage `json:"transactionHash"`
				} `json:"refundClaimeds"`
			}
			if json.Unmarshal(raw, &wrap) != nil {
				return false
			}
			if len(wrap.Rows) == 0 {
				return true
			}
			for _, r := range wrap.Rows {
				txNorm, ok := timelineNormTx(r.TransactionHash)
				if !ok {
					continue
				}
				bn, ok := timelineScalarUint64(r.BlockNumber)
				if !ok {
					continue
				}
				ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
				if !ok {
					continue
				}
				contrib, ok := scalarString(r.Contributor)
				if !ok {
					continue
				}
				w := normalizeAddress(contrib)
				amt, _ := scalarString(r.Amount)
				cidS, _ := scalarString(r.CampaignID)
				payload := map[string]any{
					"id": r.ID, "campaignId": cidS, "contributor": w, "amount": amt,
					"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
				}
				ev, ok := buildSyntheticCPEventLog(chainID, contract, "RefundClaimed", r.ID, txNorm, bn, ts, payload, nil, &cidPtr, &w)
				if ok {
					all = append(all, ev)
				}
			}
			skip += len(wrap.Rows)
			if len(wrap.Rows) < 1000 {
				return true
			}
		}
	}
	if !appendPagedRefunds() {
		return nil, false
	}

	appendDevPaged := func(gql, root, evName string) bool {
		skip := 0
		for {
			raw, err := h.SubgraphCodePulse.Query(ctx, gql, map[string]any{"cid": cidStr, "skip": skip})
			if err != nil {
				return false
			}
			var wrap map[string]json.RawMessage
			if json.Unmarshal(raw, &wrap) != nil {
				return false
			}
			arrRaw, ok := wrap[root]
			if !ok {
				return false
			}
			var rows []struct {
				ID              string          `json:"id"`
				CampaignID      json.RawMessage `json:"campaignId"`
				Developer       json.RawMessage `json:"developer"`
				BlockNumber     json.RawMessage `json:"blockNumber"`
				BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
				TransactionHash json.RawMessage `json:"transactionHash"`
			}
			if json.Unmarshal(arrRaw, &rows) != nil {
				return false
			}
			if len(rows) == 0 {
				return true
			}
			for _, r := range rows {
				txNorm, ok := timelineNormTx(r.TransactionHash)
				if !ok {
					continue
				}
				bn, ok := timelineScalarUint64(r.BlockNumber)
				if !ok {
					continue
				}
				ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
				if !ok {
					continue
				}
				dev, ok := scalarString(r.Developer)
				if !ok {
					continue
				}
				w := normalizeAddress(dev)
				cidS, _ := scalarString(r.CampaignID)
				payload := map[string]any{
					"id": r.ID, "campaignId": cidS, "developer": w,
					"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
				}
				ev, ok := buildSyntheticCPEventLog(chainID, contract, evName, r.ID, txNorm, bn, ts, payload, nil, &cidPtr, &w)
				if ok {
					all = append(all, ev)
				}
			}
			skip += len(rows)
			if len(rows) < 1000 {
				return true
			}
		}
	}
	if !appendDevPaged(sgTimelineDevAddedPage, "developerAddeds", "DeveloperAdded") {
		return nil, false
	}
	if !appendDevPaged(sgTimelineDevRemovedPage, "developerRemoveds", "DeveloperRemoved") {
		return nil, false
	}

	appendMilestoneApproved := func() bool {
		skip := 0
		for {
			raw, err := h.SubgraphCodePulse.Query(ctx, sgTimelineMilestoneApprovedPage, map[string]any{"cid": cidStr, "skip": skip})
			if err != nil {
				return false
			}
			var wrap struct {
				Rows []struct {
					ID              string          `json:"id"`
					CampaignID      json.RawMessage `json:"campaignId"`
					MilestoneIndex  json.RawMessage `json:"milestoneIndex"`
					BlockNumber     json.RawMessage `json:"blockNumber"`
					BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
					TransactionHash json.RawMessage `json:"transactionHash"`
				} `json:"milestoneApproveds"`
			}
			if json.Unmarshal(raw, &wrap) != nil {
				return false
			}
			if len(wrap.Rows) == 0 {
				return true
			}
			for _, r := range wrap.Rows {
				txNorm, ok := timelineNormTx(r.TransactionHash)
				if !ok {
					continue
				}
				bn, ok := timelineScalarUint64(r.BlockNumber)
				if !ok {
					continue
				}
				ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
				if !ok {
					continue
				}
				mi, _ := scalarString(r.MilestoneIndex)
				cidS, _ := scalarString(r.CampaignID)
				payload := map[string]any{
					"id": r.ID, "campaignId": cidS, "milestoneIndex": mi,
					"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
				}
				ev, ok := buildSyntheticCPEventLog(chainID, contract, "MilestoneApproved", r.ID, txNorm, bn, ts, payload, nil, &cidPtr, nil)
				if ok {
					all = append(all, ev)
				}
			}
			skip += len(wrap.Rows)
			if len(wrap.Rows) < 1000 {
				return true
			}
		}
	}
	if !appendMilestoneApproved() {
		return nil, false
	}

	appendMilestoneShare := func() bool {
		skip := 0
		for {
			raw, err := h.SubgraphCodePulse.Query(ctx, sgTimelineMilestoneSharePage, map[string]any{"cid": cidStr, "skip": skip})
			if err != nil {
				return false
			}
			var wrap struct {
				Rows []struct {
					ID              string          `json:"id"`
					CampaignID      json.RawMessage `json:"campaignId"`
					MilestoneIndex  json.RawMessage `json:"milestoneIndex"`
					Developer       json.RawMessage `json:"developer"`
					Amount          json.RawMessage `json:"amount"`
					BlockNumber     json.RawMessage `json:"blockNumber"`
					BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
					TransactionHash json.RawMessage `json:"transactionHash"`
				} `json:"milestoneShareClaimeds"`
			}
			if json.Unmarshal(raw, &wrap) != nil {
				return false
			}
			if len(wrap.Rows) == 0 {
				return true
			}
			for _, r := range wrap.Rows {
				txNorm, ok := timelineNormTx(r.TransactionHash)
				if !ok {
					continue
				}
				bn, ok := timelineScalarUint64(r.BlockNumber)
				if !ok {
					continue
				}
				ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
				if !ok {
					continue
				}
				dev, ok := scalarString(r.Developer)
				if !ok {
					continue
				}
				w := normalizeAddress(dev)
				amt, _ := scalarString(r.Amount)
				mi, _ := scalarString(r.MilestoneIndex)
				cidS, _ := scalarString(r.CampaignID)
				payload := map[string]any{
					"id": r.ID, "campaignId": cidS, "milestoneIndex": mi, "developer": w, "amount": amt,
					"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
				}
				ev, ok := buildSyntheticCPEventLog(chainID, contract, "MilestoneShareClaimed", r.ID, txNorm, bn, ts, payload, nil, &cidPtr, &w)
				if ok {
					all = append(all, ev)
				}
			}
			skip += len(wrap.Rows)
			if len(wrap.Rows) < 1000 {
				return true
			}
		}
	}
	if !appendMilestoneShare() {
		return nil, false
	}

	appendStale := func() bool {
		skip := 0
		for {
			raw, err := h.SubgraphCodePulse.Query(ctx, sgTimelineStaleFundsPage, map[string]any{"cid": cidStr, "skip": skip})
			if err != nil {
				return false
			}
			var wrap struct {
				Rows []struct {
					ID              string          `json:"id"`
					CampaignID      json.RawMessage `json:"campaignId"`
					Amount          json.RawMessage `json:"amount"`
					BlockNumber     json.RawMessage `json:"blockNumber"`
					BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
					TransactionHash json.RawMessage `json:"transactionHash"`
				} `json:"staleFundsSwepts"`
			}
			if json.Unmarshal(raw, &wrap) != nil {
				return false
			}
			if len(wrap.Rows) == 0 {
				return true
			}
			for _, r := range wrap.Rows {
				txNorm, ok := timelineNormTx(r.TransactionHash)
				if !ok {
					continue
				}
				bn, ok := timelineScalarUint64(r.BlockNumber)
				if !ok {
					continue
				}
				ts, ok := timelineScalarUnixUTC(r.BlockTimestamp)
				if !ok {
					continue
				}
				amt, _ := scalarString(r.Amount)
				cidS, _ := scalarString(r.CampaignID)
				payload := map[string]any{
					"id": r.ID, "campaignId": cidS, "amount": amt,
					"blockNumber": bn, "blockTimestamp": ts.Unix(), "transactionHash": txNorm,
				}
				ev, ok := buildSyntheticCPEventLog(chainID, contract, "StaleFundsSwept", r.ID, txNorm, bn, ts, payload, nil, &cidPtr, nil)
				if ok {
					all = append(all, ev)
				}
			}
			skip += len(wrap.Rows)
			if len(wrap.Rows) < 1000 {
				return true
			}
		}
	}
	if !appendStale() {
		return nil, false
	}

	return all, true
}

func pgFetchCampaignTimelineEvents(db *gorm.DB, campaignID uint64, eventNames []string) ([]models.CPEventLog, error) {
	var events []models.CPEventLog
	err := db.Model(&models.CPEventLog{}).
		Where("campaign_id = ? AND event_name IN ?", campaignID, eventNames).
		Order("block_number ASC, log_index ASC").
		Find(&events).Error
	return events, err
}
