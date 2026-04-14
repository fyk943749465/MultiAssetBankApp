package indexer

import (
	"context"
	"fmt"
	"strings"

	"go-chain/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// cpIndexerSource 写入 cp_event_log / 钱包角色等时的来源标识；扫块索引用 "rpc"，子图用 "subgraph"。
func cpIndexerSource(ev normalizedEvent) string {
	if ev.IndexerSource != "" {
		return ev.IndexerSource
	}
	return "subgraph"
}

// applyCodePulseEventTx 在已有事务内写入 cp_event_log 并执行 Apply（供子图同步与 RPC 扫块复用）。
func applyCodePulseEventTx(tx *gorm.DB, chainID uint64, contract string, ev normalizedEvent) error {
	src := cpIndexerSource(ev)
	payload := models.JSONB(ev.Raw)
	logRow := models.CPEventLog{
		ChainID:         chainID,
		ContractAddress: contract,
		EventName:       ev.Name,
		TxHash:          ev.TxHash,
		LogIndex:        ev.LogIndex,
		BlockNumber:     ev.Block,
		BlockTimestamp:  ev.TS,
		Payload:         payload,
		Source:          src,
	}
	ek := ev.Name + ":" + strings.TrimPrefix(ev.TxHash, "0x") + fmt.Sprintf(":%d", ev.LogIndex)
	logRow.EntityKey = &ek

	res := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tx_hash"}, {Name: "log_index"}},
		DoNothing: true,
	}).Create(&logRow)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}

	if err := ev.Apply(tx, contract, ev); err != nil {
		return err
	}

	pid, cid, w := inferProposalCampaignWallet(ev)
	updates := map[string]any{}
	if pid != nil {
		updates["proposal_id"] = *pid
	}
	if cid != nil {
		updates["campaign_id"] = *cid
	}
	if w != nil {
		updates["wallet_address"] = *w
	}
	if len(updates) > 0 {
		return tx.Model(&models.CPEventLog{}).Where("tx_hash = ? AND log_index = ?", ev.TxHash, ev.LogIndex).Updates(updates).Error
	}
	return nil
}

func (x *CodePulseSubgraph) applyEvent(ctx context.Context, contract string, ev normalizedEvent) error {
	return x.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return applyCodePulseEventTx(tx, x.ChainID, contract, ev)
	})
}
