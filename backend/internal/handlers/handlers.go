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
	DB                           *gorm.DB
	Chain                        *chain.Client
	Counter                      *contracts.Counter
	TxKey                        *ecdsa.PrivateKey
	Subgraph                     *subgraph.Client
	CodePulse                    *contracts.CodePulse
	SubgraphCodePulse            *subgraph.Client
	SubgraphNft                  *subgraph.Client
	SubgraphLending              *subgraph.Client // optional: lending list subgraph (supplies when non-empty); Bearer from lending-only Studio key
	LendingChain                 *chain.Client    // optional: lending-isolated JSON-RPC (LENDING_ETH_RPC_URL / BASE_ETH_RPC_URL), never ETH_RPC_URL
	LendingChainID               int64            // optional: default chain_id for lending tables (query overrides); 0 → 84532 in handlers
	LendingSubgraphAPIKeySource  string           // which env supplied Bearer for lending subgraph: SUBGRAPH_LENDING_API_KEY | SUBGRAPH_API_SECOND_KEY | ""
	LendingSubgraphAPIKeyPresent bool             // true if Bearer non-empty (Studio auth)
	// CodePulseServerTx 为 true 时允许 POST /api/code-pulse/tx/submit 使用 ETH_PRIVATE_KEY 代签（默认 false）。
	CodePulseServerTx bool
}
