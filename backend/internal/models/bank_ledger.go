package models

import "time"

// BankDeposit 对应链上 Deposited 事件的一条记录。
type BankDeposit struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID       uint64    `gorm:"not null;uniqueIndex:ux_bank_deposits_chain_tx_log" json:"chain_id"`
	TxHash        string    `gorm:"size:66;not null;uniqueIndex:ux_bank_deposits_chain_tx_log" json:"tx_hash"`
	LogIndex      uint      `gorm:"not null;uniqueIndex:ux_bank_deposits_chain_tx_log" json:"log_index"`
	BlockNumber   uint64    `gorm:"not null;index" json:"block_number"`
	BlockTime     time.Time `gorm:"not null" json:"block_time"`
	TokenAddress  string    `gorm:"size:42;not null" json:"token_address"`
	UserAddress   string    `gorm:"size:42;not null;index" json:"user_address"`
	AmountRaw     string    `gorm:"type:numeric(78,0);not null" json:"amount_raw"`
	CreatedAt     time.Time `json:"created_at"`
}

func (BankDeposit) TableName() string { return "bank_deposits" }

// BankWithdrawal 对应链上 Withdrawn 事件的一条记录。
type BankWithdrawal struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID       uint64    `gorm:"not null;uniqueIndex:ux_bank_withdrawals_chain_tx_log" json:"chain_id"`
	TxHash        string    `gorm:"size:66;not null;uniqueIndex:ux_bank_withdrawals_chain_tx_log" json:"tx_hash"`
	LogIndex      uint      `gorm:"not null;uniqueIndex:ux_bank_withdrawals_chain_tx_log" json:"log_index"`
	BlockNumber   uint64    `gorm:"not null;index" json:"block_number"`
	BlockTime     time.Time `gorm:"not null" json:"block_time"`
	TokenAddress  string    `gorm:"size:42;not null" json:"token_address"`
	UserAddress   string    `gorm:"size:42;not null;index" json:"user_address"`
	AmountRaw     string    `gorm:"type:numeric(78,0);not null" json:"amount_raw"`
	CreatedAt     time.Time `json:"created_at"`
}

func (BankWithdrawal) TableName() string { return "bank_withdrawals" }

// ChainIndexerCursor 记录各索引任务已处理到的区块高度（含该块）。
type ChainIndexerCursor struct {
	ID                 uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	LastScannedBlock   uint64    `gorm:"not null" json:"last_scanned_block"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (ChainIndexerCursor) TableName() string { return "chain_indexer_cursors" }
