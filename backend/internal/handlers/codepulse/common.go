package codepulse

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	whereProposalID    = "proposal_id = ?"
	whereCampaignID    = "campaign_id = ?"
	errInvalidCampaign = "invalid campaign_id"
	chainDataTTL       = time.Minute
)

// Pagination 列表分页元数据。
type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func parsePagination(c *gin.Context) (page, pageSize, offset int) {
	page = 1
	pageSize = 20
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}
	offset = (page - 1) * pageSize
	return
}

func requireDB(h *handlers.Handlers, c *gin.Context) bool {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return false
	}
	return true
}

func normalizeAddress(addr string) string {
	return strings.ToLower(strings.TrimSpace(addr))
}

func isContractOwner(h *handlers.Handlers, addr string) bool {
	active, err := resolveGlobalRole(h, addr, "admin", false)
	return err == nil && active
}

func resolveGlobalRole(h *handlers.Handlers, addr, role string, forceRefresh bool) (bool, error) {
	addr = normalizeAddress(addr)
	if h == nil || h.DB == nil {
		return false, nil
	}

	switch role {
	case "admin":
		state, err := resolveSystemState(h, forceRefresh)
		if err == nil && state != nil {
			return normalizeAddress(state.OwnerAddress) == addr, nil
		}
		if cached, ok := loadCachedGlobalRole(h, addr, role); ok {
			return cached.Active, nil
		}
		return false, err
	case "proposal_initiator":
		if cached, ok := loadCachedGlobalRole(h, addr, role); ok && !forceRefresh && cachedFresh(cached.SyncedAt) {
			return cached.Active, nil
		}
		active, err := syncProposalInitiatorRole(h, addr)
		if err == nil {
			return active, nil
		}
		if cached, ok := loadCachedGlobalRole(h, addr, role); ok {
			return cached.Active, nil
		}
		return false, err
	default:
		if cached, ok := loadCachedGlobalRole(h, addr, role); ok {
			return cached.Active, nil
		}
		return false, nil
	}
}

func resolveSystemState(h *handlers.Handlers, forceRefresh bool) (*models.CPSystemState, error) {
	if h == nil || h.DB == nil || h.CodePulse == nil {
		return nil, nil
	}

	contractAddr := normalizeAddress(h.CodePulse.Address().Hex())
	var state models.CPSystemState
	err := h.DB.Where("LOWER(contract_address) = ?", contractAddr).First(&state).Error
	if err == nil && !forceRefresh && cachedFresh(state.SyncedAt) {
		return &state, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return syncSystemStateFromChain(h, contractAddr)
}

func syncSystemStateFromChain(h *handlers.Handlers, contractAddr string) (*models.CPSystemState, error) {
	if h == nil || h.DB == nil || h.CodePulse == nil {
		return nil, nil
	}

	ctx := context.Background()
	owner, err := h.CodePulse.Owner(ctx)
	if err != nil {
		return nil, err
	}
	paused, err := h.CodePulse.Paused(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	blockNumber := currentBlockNumber(h, ctx)
	ownerAddr := normalizeAddress(owner.Hex())

	state := models.CPSystemState{ContractAddress: contractAddr}
	assign := map[string]any{
		"owner_address":        ownerAddr,
		"paused":               paused,
		"source":               "chain",
		"source_block_number":  blockNumber,
		"synced_at":            &now,
		"updated_at":           now,
	}
	if err := h.DB.Where("LOWER(contract_address) = ?", contractAddr).
		Assign(assign).
		FirstOrCreate(&state, models.CPSystemState{
			ContractAddress: contractAddr,
			OwnerAddress:    ownerAddr,
			Paused:          paused,
			Source:          "chain",
			SyncedAt:        &now,
		}).Error; err != nil {
		return nil, err
	}
	if err := syncAdminRoleCache(h, ownerAddr, blockNumber, &now); err != nil {
		return nil, err
	}
	if err := h.DB.Where("LOWER(contract_address) = ?", contractAddr).First(&state).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func syncProposalInitiatorRole(h *handlers.Handlers, addr string) (bool, error) {
	if h == nil || h.DB == nil || h.CodePulse == nil || !common.IsHexAddress(addr) {
		if cached, ok := loadCachedGlobalRole(h, addr, "proposal_initiator"); ok {
			return cached.Active, nil
		}
		return false, nil
	}

	ctx := context.Background()
	active, err := h.CodePulse.IsProposalInitiator(ctx, common.HexToAddress(addr))
	if err != nil {
		return false, err
	}
	now := time.Now()
	blockNumber := currentBlockNumber(h, ctx)
	if err := upsertGlobalRoleCache(h, addr, "proposal_initiator", active, "chain_view", "chain", blockNumber, &now); err != nil {
		return false, err
	}
	return active, nil
}

func syncAdminRoleCache(h *handlers.Handlers, ownerAddr string, blockNumber *uint64, syncedAt *time.Time) error {
	if h == nil || h.DB == nil {
		return nil
	}
	now := time.Now()
	if err := h.DB.Model(&models.CPWalletRole{}).
		Where("role = ? AND scope_type = ? AND source = ? AND LOWER(wallet_address) <> ?", "admin", "global", "chain", ownerAddr).
		Updates(map[string]any{
			"active":              false,
			"source_block_number": blockNumber,
			"synced_at":           syncedAt,
			"updated_at":          now,
		}).Error; err != nil {
		return err
	}
	return upsertGlobalRoleCache(h, ownerAddr, "admin", true, "contract_owner", "chain", blockNumber, syncedAt)
}

func upsertGlobalRoleCache(
	h *handlers.Handlers,
	addr, role string,
	active bool,
	derivedFrom, source string,
	blockNumber *uint64,
	syncedAt *time.Time,
) error {
	if h == nil || h.DB == nil {
		return nil
	}
	addr = normalizeAddress(addr)
	now := time.Now()
	lookup := models.CPWalletRole{}
	err := h.DB.Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ?", addr, role, "global").
		First(&lookup).Error
	switch err {
	case nil:
		return h.DB.Model(&lookup).Updates(map[string]any{
			"active":              active,
			"derived_from":        derivedFrom,
			"source":              source,
			"source_block_number": blockNumber,
			"synced_at":           syncedAt,
			"updated_at":          now,
		}).Error
	case gorm.ErrRecordNotFound:
		roleRow := models.CPWalletRole{
			WalletAddress:     addr,
			Role:              role,
			ScopeType:         "global",
			Active:            active,
			DerivedFrom:       derivedFrom,
			Source:            source,
			SourceBlockNumber: blockNumber,
			SyncedAt:          syncedAt,
		}
		return h.DB.Create(&roleRow).Error
	default:
		return err
	}
}

func loadCachedGlobalRole(h *handlers.Handlers, addr, role string) (models.CPWalletRole, bool) {
	if h == nil || h.DB == nil {
		return models.CPWalletRole{}, false
	}
	var cached models.CPWalletRole
	err := h.DB.Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ?", normalizeAddress(addr), role, "global").
		First(&cached).Error
	return cached, err == nil
}

func cachedFresh(syncedAt *time.Time) bool {
	return syncedAt != nil && time.Since(*syncedAt) <= chainDataTTL
}

func currentBlockNumber(h *handlers.Handlers, ctx context.Context) *uint64 {
	if h == nil || h.Chain == nil || h.Chain.Eth() == nil {
		return nil
	}
	blockNumber, err := h.Chain.Eth().BlockNumber(ctx)
	if err != nil {
		return nil
	}
	return &blockNumber
}
