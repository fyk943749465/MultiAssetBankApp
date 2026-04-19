package handlers

import (
	"crypto/ecdsa"

	"go-chain/backend/internal/chain"
	"go-chain/backend/internal/contracts"
	"go-chain/backend/internal/subgraph"

	"gorm.io/gorm"
)

// Handlers 聚合各业务 HTTP 依赖；具体路由实现在 handlers/<业务>/ 子包中。
type Handlers struct {
	DB                *gorm.DB
	Chain             *chain.Client
	Counter           *contracts.Counter
	TxKey             *ecdsa.PrivateKey
	Subgraph          *subgraph.Client
	CodePulse         *contracts.CodePulse
	SubgraphCodePulse *subgraph.Client
	SubgraphNft       *subgraph.Client
	// CodePulseServerTx 为 true 时允许 POST /api/code-pulse/tx/submit 使用 ETH_PRIVATE_KEY 代签（默认 false）。
	CodePulseServerTx bool
}
