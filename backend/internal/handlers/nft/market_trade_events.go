package nft

import (
	"net/http"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

var allowedMarketEventTypes = map[string]struct{}{
	"ItemListed":          {},
	"ListingPriceUpdated": {},
	"ListingCanceled":     {},
	"ItemSold":            {},
}

type tradeEventRow struct {
	models.NFTMarketTradeEvent
	SellerAddress string `json:"seller_address,omitempty"`
	BuyerAddress  string `json:"buyer_address,omitempty"`
}

// MarketTradeEvents GET /api/nft/market/trade-events
// 挂单 / 改价 / 撤单 / 成交 的链上事件流水（PostgreSQL 扫块写入 nft_market_trade_events）。
// Query: page, page_size, event_type=（可选，单一类型）, involves=0x…（可选，卖家或买家地址匹配）。
func MarketTradeEvents(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		page, pageSize := queryPage(c)
		offset := (page - 1) * pageSize

		eventType := strings.TrimSpace(c.Query("event_type"))
		if eventType != "" {
			if _, ok := allowedMarketEventTypes[eventType]; !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid event_type; use ItemListed | ListingPriceUpdated | ListingCanceled | ItemSold or omit",
				})
				return
			}
		}

		involves := strings.TrimSpace(c.Query("involves"))
		if involves != "" && !common.IsHexAddress(involves) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid involves address"})
			return
		}
		involvesLower := strings.ToLower(involves)

		q := h.DB.Table("nft_market_trade_events AS t").
			Select(`t.*, 
				COALESCE(lower(sa.address), '') AS seller_address,
				COALESCE(lower(ba.address), '') AS buyer_address`).
			Joins("LEFT JOIN nft_accounts sa ON sa.id = t.seller_account_id AND sa.chain_id = ?", chainID).
			Joins("LEFT JOIN nft_accounts ba ON ba.id = t.buyer_account_id AND ba.chain_id = ?", chainID).
			Where("t.chain_id = ?", chainID)

		if eventType != "" {
			q = q.Where("t.event_type = ?", eventType)
		}
		if involvesLower != "" {
			q = q.Where("(lower(sa.address) = ? OR lower(ba.address) = ?)", involvesLower, involvesLower)
		}

		var total int64
		countQ := h.DB.Table("nft_market_trade_events AS t").
			Joins("LEFT JOIN nft_accounts sa ON sa.id = t.seller_account_id AND sa.chain_id = ?", chainID).
			Joins("LEFT JOIN nft_accounts ba ON ba.id = t.buyer_account_id AND ba.chain_id = ?", chainID).
			Where("t.chain_id = ?", chainID)
		if eventType != "" {
			countQ = countQ.Where("t.event_type = ?", eventType)
		}
		if involvesLower != "" {
			countQ = countQ.Where("(lower(sa.address) = ? OR lower(ba.address) = ?)", involvesLower, involvesLower)
		}
		if err := countQ.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var rows []tradeEventRow
		if err := q.Order("t.block_number DESC, t.log_index DESC, t.id DESC").
			Limit(pageSize).
			Offset(offset).
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data_source": "database",
			"chain_id":    chainID,
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"events":      rows,
		})
	}
}
