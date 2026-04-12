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

// Health Liveness check.
// @Summary      Health
// @Description  Returns JSON `{ "status": "ok" }` when the process is running.
// @Tags         system
// @Produce      json
// @Success      200 {object} HealthResp
// @Router       /health [get]
func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// APIInfo Basic API metadata.
// @Summary      API info
// @Description  Returns API name and version string.
// @Tags         system
// @Produce      json
// @Success      200 {object} APIInfoResp
// @Router       /api/info [get]
func (h *Handlers) APIInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "go-chain API",
		"version": "0.1.0",
	})
}
