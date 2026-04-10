package router

import (
	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
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

	r.GET("/health", h.Health)
	r.GET("/api/info", h.APIInfo)
	r.GET("/api/chain/status", h.ChainStatus)
	r.GET("/api/contract/counter/value", h.CounterValue)
	r.POST("/api/contract/counter/count", h.CounterIncrement)
	r.GET("/api/bank/deposits", h.BankDeposits)
	r.GET("/api/bank/withdrawals", h.BankWithdrawals)
	r.GET("/api/bank/subgraph/deposits", h.BankSubgraphDeposits)
	r.GET("/api/bank/subgraph/withdrawals", h.BankSubgraphWithdrawals)

	return r
}
