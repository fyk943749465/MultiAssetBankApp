package nft

import (
	"errors"
	"math/big"
	"net/http"
	"strings"

	"go-chain/backend/internal/handlers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VerifyActiveListing GET /api/nft/listings/verify-active?collection=0x&token_id=
// 购买前校验：PostgreSQL 中是否存在该 collection + token 的 listing_status=active 记录（扫块入库）。
func VerifyActiveListing(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		rawColl := strings.TrimSpace(c.Query("collection"))
		rawTid := strings.TrimSpace(c.Query("token_id"))
		if rawColl == "" || rawTid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "collection and token_id required"})
			return
		}
		if !common.IsHexAddress(rawColl) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid collection"})
			return
		}
		tok := new(big.Int)
		if _, ok := tok.SetString(rawTid, 10); !ok || tok.Sign() < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		tidStr := tok.String()
		coll := strings.ToLower(common.HexToAddress(rawColl).Hex())

		var row activeListingRow
		err := h.DB.Table("nft_active_listings AS l").
			Select("l.*, lower(na.address) AS seller_address").
			Joins("JOIN nft_accounts na ON na.id = l.seller_account_id").
			Where("l.chain_id = ? AND l.listing_status = ? AND lower(l.collection_address) = ? AND l.token_id = ?",
				chainID, "active", coll, tidStr).
			Take(&row).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusOK, gin.H{
					"chain_id": chainID,
					"active":   false,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"chain_id": chainID,
			"active":   true,
			"listing":  row,
		})
	}
}
