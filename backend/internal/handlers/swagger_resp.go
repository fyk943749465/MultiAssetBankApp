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
