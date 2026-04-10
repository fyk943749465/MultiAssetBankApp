package handlers

import (
	"crypto/ecdsa"
	"net/http"

	"go-chain/backend/internal/chain"
	"go-chain/backend/internal/contracts"
	"go-chain/backend/internal/subgraph"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handlers struct {
	DB       *gorm.DB
	Chain    *chain.Client
	Counter  *contracts.Counter
	TxKey    *ecdsa.PrivateKey
	Subgraph *subgraph.Client
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) APIInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "go-chain API",
		"version": "0.1.0",
	})
}
