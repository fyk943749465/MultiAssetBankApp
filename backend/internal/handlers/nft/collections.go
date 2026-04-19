package nft

import (
	"net/http"
	"strconv"
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

const subgraphReorgNote = "子图数据仅供参考，重组后可能与链不一致；未写入数据库。"

type collectionListRow struct {
	models.NFTCollection
	ContractAddress string `json:"contract_address"`
	CreatorAddress  string `json:"creator_address"`
}

// Collections GET /api/nft/collections
// 读策略：子图已配置且本页能拉到数据 → 优先子图（快于扫块入库）；否则用 PostgreSQL。
func Collections(h *handlers.Handlers) gin.HandlerFunc {
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

		if h.SubgraphNft != nil && h.SubgraphNft.Configured() {
			sgRows, hasMore, errSG := fetchSubgraphCollectionsFallback(c.Request.Context(), h.SubgraphNft, page, pageSize)
			if errSG == nil && len(sgRows) > 0 {
				c.JSON(http.StatusOK, gin.H{
					"chain_id":      chainID,
					"data_source":   "subgraph",
					"subgraph_note": subgraphReorgNote,
					"page":          page,
					"page_size":     pageSize,
					"total":         int64(len(sgRows)),
					"total_note":    "本页条数；子图未提供全量 total。",
					"has_more":      hasMore,
					"collections":   sgRows,
				})
				return
			}
		}

		var total int64
		if err := h.DB.Model(&models.NFTCollection{}).Where("chain_id = ?", chainID).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var rows []collectionListRow
		q := h.DB.Table("nft_collections AS col").
			Select("col.*, lower(nc.address) AS contract_address, lower(na.address) AS creator_address").
			Joins("JOIN nft_contracts nc ON nc.id = col.contract_id").
			Joins("JOIN nft_accounts na ON na.id = col.creator_account_id").
			Where("col.chain_id = ?", chainID).
			Order("col.id DESC").
			Limit(pageSize).
			Offset(offset)
		if err := q.Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		out := gin.H{
			"chain_id":    chainID,
			"data_source": "database",
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"collections": rows,
		}
		c.JSON(http.StatusOK, out)
	}
}

// CollectionByID GET /api/nft/collections/:id
// 仍走 PostgreSQL（按库主键 id；子图实体 id 与库 id 不一一对应）。
func CollectionByID(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var row collectionListRow
		err = h.DB.Table("nft_collections AS col").
			Select("col.*, lower(nc.address) AS contract_address, lower(na.address) AS creator_address").
			Joins("JOIN nft_contracts nc ON nc.id = col.contract_id").
			Joins("JOIN nft_accounts na ON na.id = col.creator_account_id").
			Where("col.chain_id = ? AND col.id = ?", chainID, id).
			Scan(&row).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if row.ID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data_source": "database", "collection": row})
	}
}

// CollectionByContractAddress GET /api/nft/collections/by-contract/:contract
// 仅 PostgreSQL：克隆合集合约地址已扫块入库后才可查到；用于前端铸造页准入校验。
func CollectionByContractAddress(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
			return
		}
		chainID, ok := resolveChainID(c, h)
		if !ok {
			return
		}
		raw := strings.TrimSpace(c.Param("contract"))
		if !common.IsHexAddress(raw) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract address"})
			return
		}
		addr := strings.ToLower(common.HexToAddress(raw).Hex())

		var row collectionListRow
		err := h.DB.Table("nft_collections AS col").
			Select("col.*, lower(nc.address) AS contract_address, lower(na.address) AS creator_address").
			Joins("JOIN nft_contracts nc ON nc.id = col.contract_id").
			Joins("JOIN nft_accounts na ON na.id = col.creator_account_id").
			Where("col.chain_id = ? AND lower(nc.address) = ?", chainID, addr).
			Scan(&row).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if row.ID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "collection not found in database"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data_source": "database", "collection": row})
	}
}
