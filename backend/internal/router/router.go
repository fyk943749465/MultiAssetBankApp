package router

import (
	"go-chain/backend/internal/handlers"
	hbank "go-chain/backend/internal/handlers/bank"
	hchain "go-chain/backend/internal/handlers/chain"
	hcp "go-chain/backend/internal/handlers/codepulse"
	hctr "go-chain/backend/internal/handlers/contract"
	hlending "go-chain/backend/internal/handlers/lending"
	hnft "go-chain/backend/internal/handlers/nft"
	hsys "go-chain/backend/internal/handlers/system"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func devCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func New(h *handlers.Handlers) *gin.Engine {
	r := gin.Default()
	r.Use(devCORS())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/docs", ReDoc)
	r.GET("/scalar", Scalar)

	r.GET("/health", hsys.Health())
	r.GET("/api/info", hsys.APIInfo())
	r.GET("/api/chain/status", hchain.Status(h))
	r.GET("/api/contract/counter/value", hctr.CounterValue(h))
	r.POST("/api/contract/counter/count", hctr.CounterIncrement(h))
	r.GET("/api/bank/deposits", hbank.Deposits(h))
	r.GET("/api/bank/withdrawals", hbank.Withdrawals(h))
	r.GET("/api/bank/subgraph/deposits", hbank.SubgraphDeposits(h))
	r.GET("/api/bank/subgraph/withdrawals", hbank.SubgraphWithdrawals(h))

	hcp.Register(r, h)
	hnft.Register(r, h)
	hlending.Register(r, h)

	return r
}
