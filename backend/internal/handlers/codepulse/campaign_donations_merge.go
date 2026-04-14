package codepulse

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"gorm.io/gorm"
)

const sgCampaignDonationsListByCampaignQuery = `
query CpCampDonList($cid: BigInt!, $skip: Int!) {
  donateds(first: 1000, skip: $skip, orderBy: blockNumber, orderDirection: asc, where: { campaignId: $cid }) {
    id
    campaignId
    contributor
    amount
    blockNumber
    blockTimestamp
    transactionHash
  }
}
`

// parseSubgraphDonatedEntityID 解析子图 Donated 实体 id（tx.hash.concatI32(logIndex)）为 tx 与 logIndex。
func parseSubgraphDonatedEntityID(idHex string) (txHash string, logIndex int, ok bool) {
	s := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(idHex), "0x"))
	if len(s) < 72 {
		return "", 0, false
	}
	txPart := s[:64]
	if _, err := hex.DecodeString(txPart); err != nil {
		return "", 0, false
	}
	liBytes, err := hex.DecodeString(s[64:72])
	if err != nil || len(liBytes) != 4 {
		return "", 0, false
	}
	u := binary.BigEndian.Uint32(liBytes)
	return "0x" + txPart, int(int32(u)), true
}

func donationDedupKey(txHash string, logIndex int) string {
	return normalizeAddress(txHash) + ":" + strconv.Itoa(logIndex)
}

// sgFetchDonationsForCampaign 分页拉取某活动的全部 Donated 子图实体（单笔）。
func sgFetchDonationsForCampaign(ctx context.Context, h *handlers.Handlers, campaignID uint64) ([]models.CampaignDonationEntry, bool) {
	if !sgAvailable(h) {
		return nil, false
	}
	cidStr := strconv.FormatUint(campaignID, 10)
	skip := 0
	var out []models.CampaignDonationEntry
	for {
		raw, err := h.SubgraphCodePulse.Query(ctx, sgCampaignDonationsListByCampaignQuery, map[string]any{
			"cid":  cidStr,
			"skip": skip,
		})
		if err != nil {
			return nil, false
		}
		var wrap struct {
			Donateds []struct {
				ID              string          `json:"id"`
				CampaignID      json.RawMessage `json:"campaignId"`
				Contributor     json.RawMessage `json:"contributor"`
				Amount          json.RawMessage `json:"amount"`
				BlockTimestamp  json.RawMessage `json:"blockTimestamp"`
				TransactionHash json.RawMessage `json:"transactionHash"`
			} `json:"donateds"`
		}
		if json.Unmarshal(raw, &wrap) != nil {
			return nil, false
		}
		if len(wrap.Donateds) == 0 {
			return out, true
		}
		for _, d := range wrap.Donateds {
			txStr, err := parseGraphQLScalarToString(d.TransactionHash)
			if err != nil || txStr == "" {
				continue
			}
			txNorm := normalizeAddress(txStr)
			if !strings.HasPrefix(txNorm, "0x") {
				txNorm = "0x" + strings.TrimPrefix(txNorm, "0x")
			}
			li := 0
			if _, logIdx, ok := parseSubgraphDonatedEntityID(d.ID); ok {
				li = logIdx
			}
			contribStr, err := parseGraphQLScalarToString(d.Contributor)
			if err != nil || strings.TrimSpace(contribStr) == "" {
				continue
			}
			amt := big.NewInt(0)
			if n, ok := parseWeiFromGraphScalar(d.Amount); ok {
				amt = n
			}
			tsStr, err := parseGraphQLScalarToString(d.BlockTimestamp)
			if err != nil {
				continue
			}
			sec, err := strconv.ParseInt(strings.TrimSpace(tsStr), 10, 64)
			if err != nil {
				continue
			}
			out = append(out, models.CampaignDonationEntry{
				CampaignID:         campaignID,
				ContributorAddress: normalizeAddress(contribStr),
				AmountWei:          amt.String(),
				DonatedAt:          time.Unix(sec, 0).UTC(),
				TxHash:             txNorm,
				LogIndex:           li,
				RefundClaimedWei:   "0",
				Source:             "subgraph",
			})
		}
		skip += len(wrap.Donateds)
		if len(wrap.Donateds) < 1000 {
			return out, true
		}
	}
}

func donationEntryFromCPEventLog(cid uint64, evRow models.CPEventLog) (models.CampaignDonationEntry, error) {
	var p struct {
		Contributor json.RawMessage `json:"contributor"`
		Amount      json.RawMessage `json:"amount"`
	}
	if err := json.Unmarshal(evRow.Payload, &p); err != nil {
		return models.CampaignDonationEntry{}, err
	}
	contribStr, err := parseGraphQLScalarToString(p.Contributor)
	if err != nil || strings.TrimSpace(contribStr) == "" {
		return models.CampaignDonationEntry{}, fmt.Errorf("contributor")
	}
	amt := big.NewInt(0)
	if n, ok := parseWeiFromGraphScalar(p.Amount); ok {
		amt = n
	}
	txNorm := normalizeAddress(evRow.TxHash)
	if !strings.HasPrefix(txNorm, "0x") {
		txNorm = "0x" + strings.TrimPrefix(txNorm, "0x")
	}
	return models.CampaignDonationEntry{
		CampaignID:         cid,
		ContributorAddress: normalizeAddress(contribStr),
		AmountWei:          amt.String(),
		DonatedAt:          evRow.BlockTimestamp.UTC(),
		TxHash:             txNorm,
		LogIndex:           evRow.LogIndex,
		RefundClaimedWei:   "0",
		Source:             "database",
	}, nil
}

// pgFetchDonationsFromEventLog 从 cp_event_log 读取 Donated 单笔（RPC/子图写入均可）。
func pgFetchDonationsFromEventLog(db *gorm.DB, campaignID uint64) ([]models.CampaignDonationEntry, error) {
	var evRows []models.CPEventLog
	if err := db.Where("event_name = ? AND campaign_id = ?", "Donated", campaignID).
		Order("block_number ASC, log_index ASC").
		Find(&evRows).Error; err != nil {
		return nil, err
	}
	out := make([]models.CampaignDonationEntry, 0, len(evRows))
	for _, lg := range evRows {
		e, err := donationEntryFromCPEventLog(campaignID, lg)
		if err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// dedupDonationsByTxLog 同一 (tx_hash, log_index) 只保留一条（单子图内偶发重复时兜底）。
func dedupDonationsByTxLog(rows []models.CampaignDonationEntry) []models.CampaignDonationEntry {
	seen := make(map[string]struct{})
	out := make([]models.CampaignDonationEntry, 0, len(rows))
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

func sortCampaignDonations(rows []models.CampaignDonationEntry, sortKey string) {
	switch sortKey {
	case "latest":
		sort.Slice(rows, func(i, j int) bool {
			if !rows[i].DonatedAt.Equal(rows[j].DonatedAt) {
				return rows[i].DonatedAt.After(rows[j].DonatedAt)
			}
			if rows[i].LogIndex != rows[j].LogIndex {
				return rows[i].LogIndex > rows[j].LogIndex
			}
			return rows[i].TxHash > rows[j].TxHash
		})
	default: // amount_desc
		sort.Slice(rows, func(i, j int) bool {
			ai := mustParseWeiString(rows[i].AmountWei)
			aj := mustParseWeiString(rows[j].AmountWei)
			if ai.Cmp(aj) != 0 {
				return ai.Cmp(aj) > 0
			}
			if !rows[i].DonatedAt.Equal(rows[j].DonatedAt) {
				return rows[i].DonatedAt.After(rows[j].DonatedAt)
			}
			return rows[i].TxHash > rows[j].TxHash
		})
	}
}

// contributionsListDataSource 列表来源：子图成功则 subgraph；子图失败时用 PG 则为 database，无库或无数据为 empty。
func contributionsListDataSource(sgOK bool, n int) string {
	if sgOK {
		return "subgraph"
	}
	if n == 0 {
		return "empty"
	}
	return "database"
}
