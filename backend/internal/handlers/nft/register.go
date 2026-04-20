package nft

import (
	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Register NFT 平台只读 API。
func Register(r gin.IRoutes, h *handlers.Handlers) {
	r.GET("/api/nft/contracts", Contracts(h))
	r.GET("/api/nft/collections", Collections(h))
	// 必须在 :id 之前注册，避免 "by-contract" 被当成数字 id。
	r.GET("/api/nft/collections/by-contract/:contract", CollectionByContractAddress(h))
	r.GET("/api/nft/collections/:id", CollectionByID(h))
	r.GET("/api/nft/collections/:id/tokens", CollectionTokens(h))
	r.GET("/api/nft/listings/verify-active", VerifyActiveListing(h))
	r.GET("/api/nft/listings/active", ActiveListings(h))
	r.GET("/api/nft/market/trade-events", MarketTradeEvents(h))
	r.GET("/api/nft/holdings", HoldingsByOwner(h))
	r.GET("/api/nft/sync-status", SyncStatus(h))
	r.GET("/api/nft/subgraph/meta", SubgraphMeta(h))
	r.GET("/api/nft/subgraph/collection", SubgraphCollectionByAddress(h))
}
