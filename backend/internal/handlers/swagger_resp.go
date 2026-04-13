package handlers

import "go-chain/backend/internal/models"

// Types below are used only for Swagger / OpenAPI generation (@Success comments).

type HealthResp struct {
	Status string `json:"status" example:"ok"`
}

type APIInfoResp struct {
	Name    string `json:"name" example:"go-chain API"`
	Version string `json:"version" example:"0.1.0"`
}

type ChainStatusResp struct {
	Configured bool   `json:"configured"`
	ChainID    uint64 `json:"chain_id,omitempty"`
	Message    string `json:"message,omitempty"`
}

type CounterValueResp struct {
	Value string `json:"value"`
}

type CounterIncrementResp struct {
	TxHash string `json:"tx_hash"`
}

type BankDepositsResp struct {
	Deposits []models.BankDeposit `json:"deposits"`
}

type BankWithdrawalsResp struct {
	Withdrawals []models.BankWithdrawal `json:"withdrawals"`
}

// SubgraphEventRow 银行子图单条事件（Swagger 与 bank 子图接口共用）。
type SubgraphEventRow struct {
	SubgraphEntityID string `json:"subgraph_entity_id"`
	TokenAddress     string `json:"token_address"`
	UserAddress      string `json:"user_address"`
	AmountRaw        string `json:"amount_raw"`
	BlockNumber      uint64 `json:"block_number"`
	BlockTime        string `json:"block_time"`
	TxHash           string `json:"tx_hash"`
}

type SubgraphDepositsResp struct {
	Deposits []SubgraphEventRow `json:"deposits"`
	Source   string             `json:"source" example:"subgraph"`
}

type SubgraphWithdrawalsResp struct {
	Withdrawals []SubgraphEventRow `json:"withdrawals"`
	Source      string             `json:"source" example:"subgraph"`
}

type ErrorJSON struct {
	Error string `json:"error"`
}
