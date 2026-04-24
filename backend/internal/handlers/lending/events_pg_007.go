package lending

import (
	"strings"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ReserveInitialized GET /api/lending/reserve-initialized
// @Summary      Pool.ReserveInitialized 列表
// @Description  PostgreSQL（007 表）。支持 pool_address、asset_address、分页。
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        pool_address query string false "Pool 地址"
// @Param        asset_address query string false "底层资产地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/reserve-initialized [get]
func ReserveInitialized(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "reserve_initialized", fetchReserveInitializedPG)
}

// EmodeCategoryConfigured GET /api/lending/emode-category-configured
// @Summary      Pool.EModeCategoryConfigured 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        pool_address query string false "Pool 地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/emode-category-configured [get]
func EmodeCategoryConfigured(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "emode_category_configured", fetchEmodeCategoryConfiguredPG)
}

// HybridPoolSet GET /api/lending/hybrid-pool-set
// @Summary      HybridPriceOracle.PoolSet 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        oracle_address query string false "Hybrid 预言机合约地址"
// @Param        pool_address query string false "Pool 地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/hybrid-pool-set [get]
func HybridPoolSet(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "hybrid_pool_set", fetchHybridPoolSetPG)
}

// ReportsAuthorizedOracleSet GET /api/lending/reports-authorized-oracle-set
// @Summary      ReportsVerifier.AuthorizedOracleSet 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        verifier_address query string false "ReportsVerifier 合约地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/reports-authorized-oracle-set [get]
func ReportsAuthorizedOracleSet(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "reports_authorized_oracle_set", fetchReportsAuthorizedOracleSetPG)
}

// ReportsTokenSwept GET /api/lending/reports-token-swept
// @Summary      ReportsVerifier.TokenSwept 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        verifier_address query string false "ReportsVerifier 合约地址"
// @Param        token_address query string false "被 sweep 的 token"
// @Param        user_address query string false "收款地址 to 过滤"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/reports-token-swept [get]
func ReportsTokenSwept(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "reports_token_swept", fetchReportsTokenSweptPG)
}

// ReportsNativeSwept GET /api/lending/reports-native-swept
// @Summary      ReportsVerifier.NativeSwept 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        verifier_address query string false "ReportsVerifier 合约地址"
// @Param        user_address query string false "收款地址 to 过滤"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/reports-native-swept [get]
func ReportsNativeSwept(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "reports_native_swept", fetchReportsNativeSweptPG)
}

// ChainlinkFeedSet GET /api/lending/chainlink-feed-set
// @Summary      ChainlinkPriceOracle.FeedSet 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        oracle_address query string false "ChainlinkPriceOracle 合约地址"
// @Param        asset_address query string false "底层资产地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/chainlink-feed-set [get]
func ChainlinkFeedSet(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "chainlink_feed_set", fetchChainlinkFeedSetPG)
}

// InterestRateStrategyDeployed GET /api/lending/interest-rate-strategy-deployed
// @Summary      InterestRateStrategy.InterestRateStrategyDeployed 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        strategy_address query string false "策略合约地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/interest-rate-strategy-deployed [get]
func InterestRateStrategyDeployed(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "interest_rate_strategy_deployed", fetchInterestRateStrategyDeployedPG)
}

// ATokenMints GET /api/lending/a-token-mints
// @Summary      AToken.Mint 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        token_address query string false "aToken 合约地址"
// @Param        user_address query string false "接收方 to（与 to_address 二选一，user_address 优先）"
// @Param        to_address query string false "接收方地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/a-token-mints [get]
func ATokenMints(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "a_token_mints", fetchATokenMintsPG)
}

// ATokenBurns GET /api/lending/a-token-burns
// @Summary      AToken.Burn 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        token_address query string false "aToken 合约地址"
// @Param        user_address query string false "销毁方 from（与 from_address 二选一）"
// @Param        from_address query string false "from 地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/a-token-burns [get]
func ATokenBurns(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "a_token_burns", fetchATokenBurnsPG)
}

// VariableDebtTokenMints GET /api/lending/variable-debt-token-mints
// @Summary      VariableDebtToken.Mint 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        token_address query string false "debt token 合约地址"
// @Param        user_address query string false "接收方 to"
// @Param        to_address query string false "接收方地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/variable-debt-token-mints [get]
func VariableDebtTokenMints(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "variable_debt_token_mints", fetchVariableDebtTokenMintsPG)
}

// VariableDebtTokenBurns GET /api/lending/variable-debt-token-burns
// @Summary      VariableDebtToken.Burn 列表
// @Tags         lending
// @Produce      json
// @Param        chain_id query int false "默认 84532 或 LENDING_CHAIN_ID"
// @Param        token_address query string false "debt token 合约地址"
// @Param        user_address query string false "销毁方 from"
// @Param        from_address query string false "from 地址"
// @Param        page query int false "页码，默认 1"
// @Param        page_size query int false "每页条数，默认 20，最大 100"
// @Success      200 {object} map[string]interface{}
// @Router       /api/lending/variable-debt-token-burns [get]
func VariableDebtTokenBurns(h *handlers.Handlers) gin.HandlerFunc {
	return listPGOnly(h, "variable_debt_token_burns", fetchVariableDebtTokenBurnsPG)
}

func fetchReserveInitializedPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingReserveInitialized{}).Where("chain_id = ?", chainID)
	q = applyPoolAddressFilter(c, q)
	q = applyAssetAddressFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingReserveInitialized{}).Where("chain_id = ?", chainID)
	q2 = applyPoolAddressFilter(c, q2)
	q2 = applyAssetAddressFilter(c, q2)
	var rows []models.LendingReserveInitialized
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchEmodeCategoryConfiguredPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingEmodeCategoryConfigured{}).Where("chain_id = ?", chainID)
	q = applyPoolAddressFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingEmodeCategoryConfigured{}).Where("chain_id = ?", chainID)
	q2 = applyPoolAddressFilter(c, q2)
	var rows []models.LendingEmodeCategoryConfigured
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchHybridPoolSetPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingHybridPoolSet{}).Where("chain_id = ?", chainID)
	q = applyOracleAddressFilter(c, q)
	q = applyPoolAddressFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingHybridPoolSet{}).Where("chain_id = ?", chainID)
	q2 = applyOracleAddressFilter(c, q2)
	q2 = applyPoolAddressFilter(c, q2)
	var rows []models.LendingHybridPoolSet
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchReportsAuthorizedOracleSetPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingReportsAuthorizedOracleSet{}).Where("chain_id = ?", chainID)
	q = applyVerifierAddressFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingReportsAuthorizedOracleSet{}).Where("chain_id = ?", chainID)
	q2 = applyVerifierAddressFilter(c, q2)
	var rows []models.LendingReportsAuthorizedOracleSet
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchReportsTokenSweptPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingReportsTokenSwept{}).Where("chain_id = ?", chainID)
	q = applyVerifierAddressFilter(c, q)
	q = applyTokenAddressFilter(c, q)
	q = applyReportsRecipientFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingReportsTokenSwept{}).Where("chain_id = ?", chainID)
	q2 = applyVerifierAddressFilter(c, q2)
	q2 = applyTokenAddressFilter(c, q2)
	q2 = applyReportsRecipientFilter(c, q2)
	var rows []models.LendingReportsTokenSwept
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchReportsNativeSweptPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingReportsNativeSwept{}).Where("chain_id = ?", chainID)
	q = applyVerifierAddressFilter(c, q)
	q = applyReportsRecipientFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingReportsNativeSwept{}).Where("chain_id = ?", chainID)
	q2 = applyVerifierAddressFilter(c, q2)
	q2 = applyReportsRecipientFilter(c, q2)
	var rows []models.LendingReportsNativeSwept
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

// applyReportsRecipientFilter: user_address 或 to_address 匹配收款方
func applyReportsRecipientFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("to_address")); p != "" {
		return q.Where("LOWER(to_address) = LOWER(?)", p)
	}
	if u := strings.TrimSpace(c.Query("user_address")); u != "" {
		return q.Where("LOWER(to_address) = LOWER(?)", u)
	}
	return q
}

func fetchChainlinkFeedSetPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingChainlinkFeedSet{}).Where("chain_id = ?", chainID)
	q = applyOracleAddressFilter(c, q)
	q = applyAssetAddressFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingChainlinkFeedSet{}).Where("chain_id = ?", chainID)
	q2 = applyOracleAddressFilter(c, q2)
	q2 = applyAssetAddressFilter(c, q2)
	var rows []models.LendingChainlinkFeedSet
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchInterestRateStrategyDeployedPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingInterestRateStrategyDeployed{}).Where("chain_id = ?", chainID)
	q = applyStrategyAddressFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingInterestRateStrategyDeployed{}).Where("chain_id = ?", chainID)
	q2 = applyStrategyAddressFilter(c, q2)
	var rows []models.LendingInterestRateStrategyDeployed
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchATokenMintsPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingATokenMint{}).Where("chain_id = ?", chainID)
	q = applyTokenAddressFilter(c, q)
	q = applyMintToFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingATokenMint{}).Where("chain_id = ?", chainID)
	q2 = applyTokenAddressFilter(c, q2)
	q2 = applyMintToFilter(c, q2)
	var rows []models.LendingATokenMint
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchATokenBurnsPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingATokenBurn{}).Where("chain_id = ?", chainID)
	q = applyTokenAddressFilter(c, q)
	q = applyBurnFromFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingATokenBurn{}).Where("chain_id = ?", chainID)
	q2 = applyTokenAddressFilter(c, q2)
	q2 = applyBurnFromFilter(c, q2)
	var rows []models.LendingATokenBurn
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchVariableDebtTokenMintsPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingVariableDebtTokenMint{}).Where("chain_id = ?", chainID)
	q = applyTokenAddressFilter(c, q)
	q = applyMintToFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingVariableDebtTokenMint{}).Where("chain_id = ?", chainID)
	q2 = applyTokenAddressFilter(c, q2)
	q2 = applyMintToFilter(c, q2)
	var rows []models.LendingVariableDebtTokenMint
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func fetchVariableDebtTokenBurnsPG(h *handlers.Handlers, c *gin.Context, chainID int64, page, pageSize int) (int64, any, error) {
	offset := (page - 1) * pageSize
	q := h.DB.Model(&models.LendingVariableDebtTokenBurn{}).Where("chain_id = ?", chainID)
	q = applyTokenAddressFilter(c, q)
	q = applyBurnFromFilter(c, q)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	q2 := h.DB.Model(&models.LendingVariableDebtTokenBurn{}).Where("chain_id = ?", chainID)
	q2 = applyTokenAddressFilter(c, q2)
	q2 = applyBurnFromFilter(c, q2)
	var rows []models.LendingVariableDebtTokenBurn
	if err := q2.Order("block_number DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func applyMintToFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("to_address")); p != "" {
		return q.Where("LOWER(to_address) = LOWER(?)", p)
	}
	if u := strings.TrimSpace(c.Query("user_address")); u != "" {
		return q.Where("LOWER(to_address) = LOWER(?)", u)
	}
	return q
}

func applyBurnFromFilter(c *gin.Context, q *gorm.DB) *gorm.DB {
	if p := strings.TrimSpace(c.Query("from_address")); p != "" {
		return q.Where("LOWER(from_address) = LOWER(?)", p)
	}
	if u := strings.TrimSpace(c.Query("user_address")); u != "" {
		return q.Where("LOWER(from_address) = LOWER(?)", u)
	}
	return q
}
