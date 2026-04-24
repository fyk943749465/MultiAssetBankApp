package models

import "time"

// 以下结构对应 006_lending.sql 中尚未在 lending_events.go 定义的事件表。

type LendingPaused struct {
	ID             uint64    `gorm:"column:id" json:"id"`
	ChainID        int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress    string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash         string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex       int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber    int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime      time.Time `gorm:"column:block_time" json:"block_time"`
	AccountAddress string    `gorm:"column:account_address" json:"account_address"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingPaused) TableName() string { return "lending_paused" }

type LendingUnpaused struct {
	ID             uint64    `gorm:"column:id" json:"id"`
	ChainID        int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress    string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash         string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex       int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber    int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime      time.Time `gorm:"column:block_time" json:"block_time"`
	AccountAddress string    `gorm:"column:account_address" json:"account_address"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingUnpaused) TableName() string { return "lending_unpaused" }

type LendingProtocolFeeRecipientUpdated struct {
	ID                  uint64    `gorm:"column:id" json:"id"`
	ChainID             int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress         string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash              string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex            int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber         int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime           time.Time `gorm:"column:block_time" json:"block_time"`
	NewRecipientAddress string    `gorm:"column:new_recipient_address" json:"new_recipient_address"`
	CreatedAt           time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingProtocolFeeRecipientUpdated) TableName() string {
	return "lending_protocol_fee_recipient_updated"
}

type LendingReserveCapsUpdated struct {
	ID            uint64    `gorm:"column:id" json:"id"`
	ChainID       int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress   string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash        string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex      int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber   int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime     time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress  string    `gorm:"column:asset_address" json:"asset_address"`
	SupplyCapRaw  string    `gorm:"column:supply_cap_raw" json:"supply_cap_raw"`
	BorrowCapRaw  string    `gorm:"column:borrow_cap_raw" json:"borrow_cap_raw"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingReserveCapsUpdated) TableName() string { return "lending_reserve_caps_updated" }

type LendingReserveLiquidationProtocolFeeUpdated struct {
	ID           uint64    `gorm:"column:id" json:"id"`
	ChainID      int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress  string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash       string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex     int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber  int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime    time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress string    `gorm:"column:asset_address" json:"asset_address"`
	FeeBpsRaw    string    `gorm:"column:fee_bps_raw" json:"fee_bps_raw"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingReserveLiquidationProtocolFeeUpdated) TableName() string {
	return "lending_reserve_liquidation_protocol_fee_updated"
}

type LendingUserEModeSet struct {
	ID           uint64    `gorm:"column:id" json:"id"`
	ChainID      int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress  string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash       string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex     int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber  int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime    time.Time `gorm:"column:block_time" json:"block_time"`
	UserAddress  string    `gorm:"column:user_address" json:"user_address"`
	CategoryID   int       `gorm:"column:category_id" json:"category_id"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingUserEModeSet) TableName() string { return "lending_user_emode_set" }

type LendingOwnershipTransferred struct {
	ID                   uint64    `gorm:"column:id" json:"id"`
	ChainID              int64     `gorm:"column:chain_id" json:"chain_id"`
	EmitterAddress       string    `gorm:"column:emitter_address" json:"emitter_address"`
	TxHash               string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex             int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber          int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime            time.Time `gorm:"column:block_time" json:"block_time"`
	PreviousOwnerAddress string    `gorm:"column:previous_owner_address" json:"previous_owner_address"`
	NewOwnerAddress      string    `gorm:"column:new_owner_address" json:"new_owner_address"`
	CreatedAt            time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingOwnershipTransferred) TableName() string { return "lending_ownership_transferred" }

type LendingHybridStreamConfigUpdated struct {
	ID               uint64    `gorm:"column:id" json:"id"`
	ChainID          int64     `gorm:"column:chain_id" json:"chain_id"`
	OracleAddress    string    `gorm:"column:oracle_address" json:"oracle_address"`
	TxHash           string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex         int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber      int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime        time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress     string    `gorm:"column:asset_address" json:"asset_address"`
	StreamFeedIDHex  string    `gorm:"column:stream_feed_id_hex" json:"stream_feed_id_hex"`
	PriceDecimals    int       `gorm:"column:price_decimals" json:"price_decimals"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingHybridStreamConfigUpdated) TableName() string { return "lending_hybrid_stream_config_updated" }

type LendingHybridStreamPriceFallbackToFeed struct {
	ID            uint64    `gorm:"column:id" json:"id"`
	ChainID       int64     `gorm:"column:chain_id" json:"chain_id"`
	OracleAddress string    `gorm:"column:oracle_address" json:"oracle_address"`
	TxHash        string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex      int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber   int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime     time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress  string    `gorm:"column:asset_address" json:"asset_address"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingHybridStreamPriceFallbackToFeed) TableName() string {
	return "lending_hybrid_stream_price_fallback_to_feed"
}

type LendingInterestRateStrategyCreated struct {
	ID                    uint64    `gorm:"column:id" json:"id"`
	ChainID               int64     `gorm:"column:chain_id" json:"chain_id"`
	FactoryAddress        string    `gorm:"column:factory_address" json:"factory_address"`
	TxHash                string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex              int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber           int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime             time.Time `gorm:"column:block_time" json:"block_time"`
	StrategyAddress       string    `gorm:"column:strategy_address" json:"strategy_address"`
	StrategyIndexRaw      string    `gorm:"column:strategy_index_raw" json:"strategy_index_raw"`
	OptimalUtilizationRaw string    `gorm:"column:optimal_utilization_raw" json:"optimal_utilization_raw"`
	BaseBorrowRateRaw     string    `gorm:"column:base_borrow_rate_raw" json:"base_borrow_rate_raw"`
	Slope1Raw             string    `gorm:"column:slope1_raw" json:"slope1_raw"`
	Slope2Raw             string    `gorm:"column:slope2_raw" json:"slope2_raw"`
	ReserveFactorRaw      string    `gorm:"column:reserve_factor_raw" json:"reserve_factor_raw"`
	CreatedAt             time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingInterestRateStrategyCreated) TableName() string { return "lending_interest_rate_strategy_created" }

type LendingInterestRateStrategyImmutableParams struct {
	ID                    uint64    `gorm:"column:id" json:"id"`
	ChainID               int64     `gorm:"column:chain_id" json:"chain_id"`
	StrategyAddress       string    `gorm:"column:strategy_address" json:"strategy_address"`
	OptimalUtilizationRaw string    `gorm:"column:optimal_utilization_raw" json:"optimal_utilization_raw"`
	BaseBorrowRateRaw     string    `gorm:"column:base_borrow_rate_raw" json:"base_borrow_rate_raw"`
	Slope1Raw             string    `gorm:"column:slope1_raw" json:"slope1_raw"`
	Slope2Raw             string    `gorm:"column:slope2_raw" json:"slope2_raw"`
	ReserveFactorRaw      string    `gorm:"column:reserve_factor_raw" json:"reserve_factor_raw"`
	SourceBlockNumber     int64     `gorm:"column:source_block_number" json:"source_block_number"`
	SourceBlockTime       time.Time `gorm:"column:source_block_time" json:"source_block_time"`
	CreatedAt             time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingInterestRateStrategyImmutableParams) TableName() string {
	return "lending_interest_rate_strategy_immutable_params"
}
