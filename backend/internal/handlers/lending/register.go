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
}
