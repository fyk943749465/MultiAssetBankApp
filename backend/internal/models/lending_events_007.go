package models

import "time"

// 以下结构对应 007_lending.sql；不加入 AutoMigrate，仅用于查询。

type LendingReserveInitialized struct {
	ID                          uint64    `gorm:"column:id" json:"id"`
	ChainID                     int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress                 string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash                      string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex                    int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber                 int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime                   time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress                string    `gorm:"column:asset_address" json:"asset_address"`
	ATokenAddress               string    `gorm:"column:a_token_address" json:"a_token_address"`
	DebtTokenAddress            string    `gorm:"column:debt_token_address" json:"debt_token_address"`
	InterestRateStrategyAddress string    `gorm:"column:interest_rate_strategy_address" json:"interest_rate_strategy_address"`
	LtvRaw                      string    `gorm:"column:ltv_raw" json:"ltv_raw"`
	LiquidationThresholdRaw     string    `gorm:"column:liquidation_threshold_raw" json:"liquidation_threshold_raw"`
	LiquidationBonusRaw         string    `gorm:"column:liquidation_bonus_raw" json:"liquidation_bonus_raw"`
	SupplyCapRaw                string    `gorm:"column:supply_cap_raw" json:"supply_cap_raw"`
	BorrowCapRaw                string    `gorm:"column:borrow_cap_raw" json:"borrow_cap_raw"`
	CreatedAt                   time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingReserveInitialized) TableName() string { return "lending_reserve_initialized" }

type LendingEmodeCategoryConfigured struct {
	ID                      uint64    `gorm:"column:id" json:"id"`
	ChainID                 int64     `gorm:"column:chain_id" json:"chain_id"`
	PoolAddress             string    `gorm:"column:pool_address" json:"pool_address"`
	TxHash                  string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex                int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber             int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime               time.Time `gorm:"column:block_time" json:"block_time"`
	CategoryID              int       `gorm:"column:category_id" json:"category_id"`
	LtvRaw                  string    `gorm:"column:ltv_raw" json:"ltv_raw"`
	LiquidationThresholdRaw string    `gorm:"column:liquidation_threshold_raw" json:"liquidation_threshold_raw"`
	LiquidationBonusRaw     string    `gorm:"column:liquidation_bonus_raw" json:"liquidation_bonus_raw"`
	Label                   string    `gorm:"column:label" json:"label"`
	CreatedAt               time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingEmodeCategoryConfigured) TableName() string { return "lending_emode_category_configured" }

type LendingHybridPoolSet struct {
	ID            uint64    `gorm:"column:id" json:"id"`
	ChainID       int64     `gorm:"column:chain_id" json:"chain_id"`
	OracleAddress string    `gorm:"column:oracle_address" json:"oracle_address"`
	TxHash        string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex      int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber   int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime     time.Time `gorm:"column:block_time" json:"block_time"`
	PoolAddress   string    `gorm:"column:pool_address" json:"pool_address"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingHybridPoolSet) TableName() string { return "lending_hybrid_pool_set" }

type LendingReportsAuthorizedOracleSet struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ChainID         int64     `gorm:"column:chain_id" json:"chain_id"`
	VerifierAddress string    `gorm:"column:verifier_address" json:"verifier_address"`
	TxHash          string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex        int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber     int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime       time.Time `gorm:"column:block_time" json:"block_time"`
	OracleAddress   string    `gorm:"column:oracle_address" json:"oracle_address"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingReportsAuthorizedOracleSet) TableName() string {
	return "lending_reports_authorized_oracle_set"
}

type LendingReportsTokenSwept struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ChainID         int64     `gorm:"column:chain_id" json:"chain_id"`
	VerifierAddress string    `gorm:"column:verifier_address" json:"verifier_address"`
	TxHash          string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex        int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber     int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime       time.Time `gorm:"column:block_time" json:"block_time"`
	TokenAddress    string    `gorm:"column:token_address" json:"token_address"`
	ToAddress       string    `gorm:"column:to_address" json:"to_address"`
	AmountRaw       string    `gorm:"column:amount_raw" json:"amount_raw"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingReportsTokenSwept) TableName() string { return "lending_reports_token_swept" }

type LendingReportsNativeSwept struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ChainID         int64     `gorm:"column:chain_id" json:"chain_id"`
	VerifierAddress string    `gorm:"column:verifier_address" json:"verifier_address"`
	TxHash          string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex        int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber     int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime       time.Time `gorm:"column:block_time" json:"block_time"`
	ToAddress       string    `gorm:"column:to_address" json:"to_address"`
	AmountRaw       string    `gorm:"column:amount_raw" json:"amount_raw"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingReportsNativeSwept) TableName() string { return "lending_reports_native_swept" }

type LendingChainlinkFeedSet struct {
	ID             uint64    `gorm:"column:id" json:"id"`
	ChainID        int64     `gorm:"column:chain_id" json:"chain_id"`
	OracleAddress  string    `gorm:"column:oracle_address" json:"oracle_address"`
	TxHash         string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex       int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber    int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime      time.Time `gorm:"column:block_time" json:"block_time"`
	AssetAddress   string    `gorm:"column:asset_address" json:"asset_address"`
	FeedAddress    string    `gorm:"column:feed_address" json:"feed_address"`
	StalePeriodRaw string    `gorm:"column:stale_period_raw" json:"stale_period_raw"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingChainlinkFeedSet) TableName() string { return "lending_chainlink_feed_set" }

type LendingInterestRateStrategyDeployed struct {
	ID                    uint64    `gorm:"column:id" json:"id"`
	ChainID               int64     `gorm:"column:chain_id" json:"chain_id"`
	StrategyAddress       string    `gorm:"column:strategy_address" json:"strategy_address"`
	TxHash                string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex              int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber           int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime             time.Time `gorm:"column:block_time" json:"block_time"`
	OptimalUtilizationRaw string    `gorm:"column:optimal_utilization_raw" json:"optimal_utilization_raw"`
	BaseBorrowRateRaw     string    `gorm:"column:base_borrow_rate_raw" json:"base_borrow_rate_raw"`
	Slope1Raw             string    `gorm:"column:slope1_raw" json:"slope1_raw"`
	Slope2Raw             string    `gorm:"column:slope2_raw" json:"slope2_raw"`
	ReserveFactorRaw      string    `gorm:"column:reserve_factor_raw" json:"reserve_factor_raw"`
	CreatedAt             time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingInterestRateStrategyDeployed) TableName() string {
	return "lending_interest_rate_strategy_deployed"
}

type LendingATokenMint struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ChainID         int64     `gorm:"column:chain_id" json:"chain_id"`
	TokenAddress    string    `gorm:"column:token_address" json:"token_address"`
	TxHash          string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex        int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber     int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime       time.Time `gorm:"column:block_time" json:"block_time"`
	ToAddress       string    `gorm:"column:to_address" json:"to_address"`
	ScaledAmountRaw string    `gorm:"column:scaled_amount_raw" json:"scaled_amount_raw"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingATokenMint) TableName() string { return "lending_a_token_mint" }

type LendingATokenBurn struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ChainID         int64     `gorm:"column:chain_id" json:"chain_id"`
	TokenAddress    string    `gorm:"column:token_address" json:"token_address"`
	TxHash          string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex        int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber     int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime       time.Time `gorm:"column:block_time" json:"block_time"`
	FromAddress     string    `gorm:"column:from_address" json:"from_address"`
	ScaledAmountRaw string    `gorm:"column:scaled_amount_raw" json:"scaled_amount_raw"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingATokenBurn) TableName() string { return "lending_a_token_burn" }

type LendingVariableDebtTokenMint struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ChainID         int64     `gorm:"column:chain_id" json:"chain_id"`
	TokenAddress    string    `gorm:"column:token_address" json:"token_address"`
	TxHash          string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex        int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber     int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime       time.Time `gorm:"column:block_time" json:"block_time"`
	ToAddress       string    `gorm:"column:to_address" json:"to_address"`
	ScaledAmountRaw string    `gorm:"column:scaled_amount_raw" json:"scaled_amount_raw"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingVariableDebtTokenMint) TableName() string { return "lending_variable_debt_token_mint" }

type LendingVariableDebtTokenBurn struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ChainID         int64     `gorm:"column:chain_id" json:"chain_id"`
	TokenAddress    string    `gorm:"column:token_address" json:"token_address"`
	TxHash          string    `gorm:"column:tx_hash" json:"tx_hash"`
	LogIndex        int       `gorm:"column:log_index" json:"log_index"`
	BlockNumber     int64     `gorm:"column:block_number" json:"block_number"`
	BlockTime       time.Time `gorm:"column:block_time" json:"block_time"`
	FromAddress     string    `gorm:"column:from_address" json:"from_address"`
	ScaledAmountRaw string    `gorm:"column:scaled_amount_raw" json:"scaled_amount_raw"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LendingVariableDebtTokenBurn) TableName() string { return "lending_variable_debt_token_burn" }
