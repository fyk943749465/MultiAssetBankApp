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

func applyAssetAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("asset_address")); p != "" {
		return q.Where("LOWER(asset_address) = LOWER(?)", p)
	}
	return q
}

func applyOracleAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("oracle_address")); p != "" {
		return q.Where("LOWER(oracle_address) = LOWER(?)", p)
	}
	return q
}

func applyVerifierAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("verifier_address")); p != "" {
		return q.Where("LOWER(verifier_address) = LOWER(?)", p)
	}
	return q
}

func applyTokenAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("token_address")); p != "" {
		return q.Where("LOWER(token_address) = LOWER(?)", p)
	}
	return q
}

func applyStrategyAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("strategy_address")); p != "" {
		return q.Where("LOWER(strategy_address) = LOWER(?)", p)
	}
	return q
}

func applyToAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("to_address")); p != "" {
		return q.Where("LOWER(to_address) = LOWER(?)", p)
	}
	return q
}

func applyFromAddressFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("from_address")); p != "" {
		return q.Where("LOWER(from_address) = LOWER(?)", p)
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
