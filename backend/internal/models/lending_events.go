package models

import "time"

// 以下结构对应 006_lending.sql 中 Pool 事件表；不加入 AutoMigrate，仅用于查询扫描。

type LendingSupply struct {
	ID            uint64    `gorm:"column:id" json:"id"`
	ChainID       int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress   string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash        string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex      int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber   int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime     time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress  string    `gorm:"column:asset_address" json:"asset_address"`
	UserAddress   string    `gorm:"column:user_address" json:"user_address"`
	AmountRaw     string    `gorm:"column:amount_raw" json:"amount_raw"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingSupply) TableName() string { return "lending_supplies" }

type LendingWithdrawal struct {
	ID           uint64    `gorm:"column:id" json:"id"`
	ChainID      int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress  string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash       string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex     int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber  int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime    time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress string    `gorm:"column:asset_address" json:"asset_address"`
	UserAddress  string    `gorm:"column:user_address" json:"user_address"`
	AmountRaw    string    `gorm:"column:amount_raw" json:"amount_raw"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingWithdrawal) TableName() string { return "lending_withdrawals" }

type LendingBorrow struct {
	ID           uint64    `gorm:"column:id" json:"id"`
	ChainID      int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress  string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash       string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex     int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber  int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime    time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress string    `gorm:"column:asset_address" json:"asset_address"`
	UserAddress  string    `gorm:"column:user_address" json:"user_address"`
	AmountRaw    string    `gorm:"column:amount_raw" json:"amount_raw"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingBorrow) TableName() string { return "lending_borrows" }

type LendingRepay struct {
	ID           uint64    `gorm:"column:id" json:"id"`
	ChainID      int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress  string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash       string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex     int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber  int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime    time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress string    `gorm:"column:asset_address" json:"asset_address"`
	UserAddress  string    `gorm:"column:user_address" json:"user_address"`
	AmountRaw    string    `gorm:"column:amount_raw" json:"amount_raw"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingRepay) TableName() string { return "lending_repays" }

type LendingLiquidation struct {
	ID                            uint64    `gorm:"column:id" json:"id"`
	ChainID                       int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress                   string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash                        string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex                      int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber                   int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime                     time.Time `gorm:"column:block_time" json:"block_time"`
	CollateralAssetAddress        string    `gorm:"column:collateral_asset_address" json:"collateral_asset_address"`
	DebtAssetAddress              string    `gorm:"column:debt_asset_address" json:"debt_asset_address"`
	BorrowerAddress               string    `gorm:"column:borrower_address" json:"borrower_address"`
	LiquidatorAddress             string    `gorm:"column:liquidator_address" json:"liquidator_address"`
	DebtCoveredRaw                string    `gorm:"column:debt_covered_raw" json:"debt_covered_raw"`
	CollateralToLiquidatorRaw     string    `gorm:"column:collateral_to_liquidator_raw" json:"collateral_to_liquidator_raw"`
	CollateralProtocolFeeRaw      string    `gorm:"column:collateral_protocol_fee_raw" json:"collateral_protocol_fee_raw"`
	CreatedAt                     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingLiquidation) TableName() string { return "lending_liquidations" }
