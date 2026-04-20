package nft

import (
	"net/http"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type holdingRow struct {
	models.NFTToken
	OwnerAddress                string  `json:"owner_address"`
	CollectionContractAddress   string  `json:"collection_contract_address"`
	CollectionName              *string `json:"collection_name,omitempty"`
}

// HoldingsByOwner GET /api/nft/holdings?owner=0x…
// 按 PostgreSQL 扫块索引返回该地址在本链上持有的 NFT（与合集 Token 列表同源，非链上实时 balanceOf）。
func HoldingsByOwner(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		raw := strings.TrimSpace(c.Query("owner"))
		if raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing query: owner (0x + 40 hex)"})
			return
		}
		if !common.IsHexAddress(raw) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner address"})
			return
		}
		ownerLower := strings.ToLower(raw)
		page, pageSize := queryPage(c)
		offset := (page - 1) * pageSize

		countDB := h.DB.Table("nft_tokens AS t").
			Joins("JOIN nft_accounts na ON na.id = t.owner_account_id AND na.chain_id = ?", chainID).
			Joins("JOIN nft_collections nc ON nc.id = t.collection_id AND nc.chain_id = ?", chainID).
			Joins("JOIN nft_contracts nctr ON nctr.id = nc.contract_id AND nctr.chain_id = ?", chainID).
			Where("t.chain_id = ? AND lower(na.address) = ?", chainID, ownerLower)

		var total int64
		if err := countDB.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var rows []holdingRow
		if err := h.DB.Table("nft_tokens AS t").
			Select("t.*, lower(na.address) AS owner_address, lower(nctr.address) AS collection_contract_address, nc.collection_name").
			Joins("JOIN nft_accounts na ON na.id = t.owner_account_id AND na.chain_id = ?", chainID).
			Joins("JOIN nft_collections nc ON nc.id = t.collection_id AND nc.chain_id = ?", chainID).
			Joins("JOIN nft_contracts nctr ON nctr.id = nc.contract_id AND nctr.chain_id = ?", chainID).
			Where("t.chain_id = ? AND lower(na.address) = ?", chainID, ownerLower).
			Order("t.collection_id ASC, t.token_id ASC").
			Limit(pageSize).
			Offset(offset).
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data_source": "database",
			"chain_id":    chainID,
			"owner":       common.HexToAddress(raw).Hex(),
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"holdings":    rows,
		})
	}
}
