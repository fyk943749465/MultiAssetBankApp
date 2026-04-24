package models

import "time"

// LendingContract 对应 lending_contracts（006 + 007 迁移）；仅用于只读查询，不加入 AutoMigrate。
// contract_kind 含：lending_pool, hybrid_price_oracle, chainlink_price_oracle, reports_verifier,
// interest_rate_strategy_factory, interest_rate_strategy, a_token, variable_debt_token。
type LendingContract struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID        int64     `gorm:"column:chain_id;not null;uniqueIndex:ux_lending_contracts_chain_address,priority:1" json:"chain_id"`
	Address        string    `gorm:"column:address;size:42;not null;uniqueIndex:ux_lending_contracts_chain_address,priority:2" json:"address"`
	ContractKind   string    `gorm:"size:40;not null" json:"contract_kind"`
	DisplayLabel   *string   `gorm:"size:160" json:"display_label,omitempty"`
	DeployedBlock  *int64    `json:"deployed_block,omitempty"`
	DeployedTxHash *string   `gorm:"size:66" json:"deployed_tx_hash,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (LendingContract) TableName() string { return "lending_contracts" }
