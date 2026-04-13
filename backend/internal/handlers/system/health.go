package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health Liveness check.
// @Summary      Health
// @Description  Returns JSON `{ "status": "ok" }` when the process is running.
// @Tags         system
// @Produce      json
// @Success      200 {object} handlers.HealthResp
// @Router       /health [get]
func Health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// APIInfo Basic API metadata.
// @Summary      API info
// @Description  Returns API name and version string.
// @Tags         system
// @Produce      json
// @Success      200 {object} handlers.APIInfoResp
// @Router       /api/info [get]
func APIInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "go-chain API",
			"version": "0.1.0",
		})
	}
}
