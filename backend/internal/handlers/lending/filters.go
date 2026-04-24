package lending

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func applyPoolAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("pool_address")); p != "" {
		return q.Where("LOWER(pool_address) = LOWER(?)", p)
	}
	return q
}

func applyUserOrParticipantFilter(c *gin.Context, q *gorm.DB, table string) *gorm.DB {
	u := strings.TrimSpace(c.Query("user_address"))
	if u == "" {
		return q
	}
	if table == "lending_liquidations" {
		return q.Where("(LOWER(borrower_address) = LOWER(?) OR LOWER(liquidator_address) = LOWER(?))", u, u)
	}
	return q.Where("LOWER(user_address) = LOWER(?)", u)
}
