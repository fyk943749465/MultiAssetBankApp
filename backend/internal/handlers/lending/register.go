package lending

import (
	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Register 挂载借贷只读 API（PG 权威；supplies 在子图配置且非空时可优先子图）。
func Register(r gin.IRoutes, h *handlers.Handlers) {
	r.GET("/api/lending/contracts", Contracts(h))
	r.GET("/api/lending/chain-status", ChainStatus(h))
	r.GET("/api/lending/native-balance", NativeBalance(h))
	r.GET("/api/lending/sync-status", SyncStatus(h))
	r.GET("/api/lending/subgraph/meta", SubgraphMeta(h))
	r.GET("/api/lending/supplies", Supplies(h))
	r.GET("/api/lending/withdrawals", Withdrawals(h))
	r.GET("/api/lending/borrows", Borrows(h))
	r.GET("/api/lending/repays", Repays(h))
	r.GET("/api/lending/liquidations", Liquidations(h))
	r.GET("/api/lending/reserve-initialized", ReserveInitialized(h))
	r.GET("/api/lending/emode-category-configured", EmodeCategoryConfigured(h))
	r.GET("/api/lending/hybrid-pool-set", HybridPoolSet(h))
	r.GET("/api/lending/reports-authorized-oracle-set", ReportsAuthorizedOracleSet(h))
	r.GET("/api/lending/reports-token-swept", ReportsTokenSwept(h))
	r.GET("/api/lending/reports-native-swept", ReportsNativeSwept(h))
	r.GET("/api/lending/chainlink-feed-set", ChainlinkFeedSet(h))
	r.GET("/api/lending/interest-rate-strategy-deployed", InterestRateStrategyDeployed(h))
	r.GET("/api/lending/a-token-mints", ATokenMints(h))
	r.GET("/api/lending/a-token-burns", ATokenBurns(h))
	r.GET("/api/lending/variable-debt-token-mints", VariableDebtTokenMints(h))
	r.GET("/api/lending/variable-debt-token-burns", VariableDebtTokenBurns(h))
}
