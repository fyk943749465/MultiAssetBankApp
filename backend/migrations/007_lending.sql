-- 借贷模块扩展（007）— 与子图 subgraph/lending 新增实体对齐，供 RPC 扫块 / 运维写入。
-- 依赖：006_lending.sql 已执行。
--
-- 1) 扩展 lending_contracts.contract_kind（储备级 aToken / variableDebtToken 可选登记）
-- 2) 移除 006 中已废弃的 Base Sepolia 种子地址，并插入当前默认部署（与 subgraph/lending/networks.json 一致）
-- 3) 新增事件表（列名与 schema.graphql 实体字段语义对齐）

-- ---------------------------------------------------------------------------
-- 1. contract_kind CHECK（兼容曾用旧 006 创建库的 6 元约束）
-- ---------------------------------------------------------------------------

ALTER TABLE lending_contracts DROP CONSTRAINT IF EXISTS ck_lending_contracts_kind;

ALTER TABLE lending_contracts ADD CONSTRAINT ck_lending_contracts_kind CHECK (contract_kind IN (
    'lending_pool',
    'hybrid_price_oracle',
    'chainlink_price_oracle',
    'reports_verifier',
    'interest_rate_strategy_factory',
    'interest_rate_strategy',
    'a_token',
    'variable_debt_token'
));

-- ---------------------------------------------------------------------------
-- 2. Base Sepolia：去掉旧版种子地址，写入当前 Pool 栈（幂等：仅删已知旧 hex）
-- ---------------------------------------------------------------------------

DELETE FROM lending_contracts WHERE chain_id = 84532 AND lower(address) IN (
    '0x65213b004b54dea6cb1096794ca3f1c24066b0ff',
    '0x37a8224bb0ea0828051adf9569967b4e8d0e1f49',
    '0xf48e792dda21f978740df4acb999c22e84a9ae6c',
    '0xdaad54b34d4db3fdb0dddf1ad37316ff862f9ab8',
    '0x7f3d525a1781e295a2ab9aa74c18f28b984dfa74',
    '0x9b91e7fa1e37d32c93f1bd1ecb7be991b53112a3'
);

INSERT INTO lending_contracts (chain_id, address, contract_kind, display_label)
VALUES
    (84532, '0x3f0248e6ff7e414485a146c18d6b72dc9e317e5f', 'lending_pool', 'Pool (Base Sepolia)'),
    (84532, '0xe72ac9c1d557d65094ae92739e409ca56ae12b11', 'hybrid_price_oracle', 'HybridPriceOracle'),
    (84532, '0x3100b1fd5a2180dac11820106579545d0f1c439b', 'chainlink_price_oracle', 'ChainlinkPriceOracle'),
    (84532, '0x960e004f33566d0b56863f54532f1785923d2799', 'reports_verifier', 'ReportsVerifier'),
    (84532, '0xb44d1c69eaf762441d6762e094b18d2614cf1617', 'interest_rate_strategy_factory', 'InterestRateStrategyFactory'),
    (84532, '0x0f4c88d757e370016b5cfc1ac48d013378be4a27', 'interest_rate_strategy', 'InterestRateStrategy')
ON CONFLICT (chain_id, address) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 3. Pool — ReserveInitialized / EModeCategoryConfigured
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_reserve_initialized (
    id                         BIGSERIAL PRIMARY KEY,
    chain_id                   BIGINT        NOT NULL,
    pool_address               VARCHAR(42)   NOT NULL,
    tx_hash                    VARCHAR(66)   NOT NULL,
    log_index                  INTEGER       NOT NULL,
    block_number               BIGINT        NOT NULL,
    block_time                 TIMESTAMPTZ   NOT NULL,
    asset_address              VARCHAR(42)   NOT NULL,
    a_token_address            VARCHAR(42)   NOT NULL,
    debt_token_address         VARCHAR(42)   NOT NULL,
    interest_rate_strategy_address VARCHAR(42) NOT NULL,
    ltv_raw                    NUMERIC(78,0) NOT NULL,
    liquidation_threshold_raw  NUMERIC(78,0) NOT NULL,
    liquidation_bonus_raw    NUMERIC(78,0) NOT NULL,
    supply_cap_raw             NUMERIC(78,0) NOT NULL,
    borrow_cap_raw             NUMERIC(78,0) NOT NULL,
    created_at                 TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_reserve_init_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_reserve_init_asset ON lending_reserve_initialized (chain_id, asset_address);

COMMENT ON TABLE lending_reserve_initialized IS 'Pool.ReserveInitialized；子图实体 ReserveInitialized。';

CREATE TABLE IF NOT EXISTS lending_emode_category_configured (
    id                         BIGSERIAL PRIMARY KEY,
    chain_id                   BIGINT        NOT NULL,
    pool_address               VARCHAR(42)   NOT NULL,
    tx_hash                    VARCHAR(66)   NOT NULL,
    log_index                  INTEGER       NOT NULL,
    block_number               BIGINT        NOT NULL,
    block_time                 TIMESTAMPTZ   NOT NULL,
    category_id                SMALLINT      NOT NULL,
    ltv_raw                    NUMERIC(78,0) NOT NULL,
    liquidation_threshold_raw  NUMERIC(78,0) NOT NULL,
    liquidation_bonus_raw    NUMERIC(78,0) NOT NULL,
    label                      TEXT          NOT NULL,
    created_at                 TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_emode_cat_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_emode_cat_pool ON lending_emode_category_configured (chain_id, pool_address);

COMMENT ON TABLE lending_emode_category_configured IS 'Pool.EModeCategoryConfigured；子图实体 EModeCategoryConfigured。';

-- ---------------------------------------------------------------------------
-- 4. HybridPriceOracle — PoolSet
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_hybrid_pool_set (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    oracle_address   VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_hybrid_pool_set_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

COMMENT ON TABLE lending_hybrid_pool_set IS 'HybridPriceOracle.PoolSet；子图实体 HybridPoolSet。';

-- ---------------------------------------------------------------------------
-- 5. ReportsVerifier — AuthorizedOracleSet / TokenSwept / NativeSwept
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_reports_authorized_oracle_set (
    id                 BIGSERIAL PRIMARY KEY,
    chain_id           BIGINT        NOT NULL,
    verifier_address   VARCHAR(42)   NOT NULL,
    tx_hash            VARCHAR(66)   NOT NULL,
    log_index          INTEGER       NOT NULL,
    block_number       BIGINT        NOT NULL,
    block_time         TIMESTAMPTZ   NOT NULL,
    oracle_address     VARCHAR(42)   NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_reports_auth_oracle_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

COMMENT ON TABLE lending_reports_authorized_oracle_set IS 'ReportsVerifier.AuthorizedOracleSet；子图实体 AuthorizedOracleSet。';

CREATE TABLE IF NOT EXISTS lending_reports_token_swept (
    id                 BIGSERIAL PRIMARY KEY,
    chain_id           BIGINT        NOT NULL,
    verifier_address   VARCHAR(42)   NOT NULL,
    tx_hash            VARCHAR(66)   NOT NULL,
    log_index          INTEGER       NOT NULL,
    block_number       BIGINT        NOT NULL,
    block_time         TIMESTAMPTZ   NOT NULL,
    token_address      VARCHAR(42)   NOT NULL,
    to_address         VARCHAR(42)   NOT NULL,
    amount_raw         NUMERIC(78,0) NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_reports_token_swept_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

COMMENT ON TABLE lending_reports_token_swept IS 'ReportsVerifier.TokenSwept；子图实体 VerifierTokenSwept。';

CREATE TABLE IF NOT EXISTS lending_reports_native_swept (
    id                 BIGSERIAL PRIMARY KEY,
    chain_id           BIGINT        NOT NULL,
    verifier_address   VARCHAR(42)   NOT NULL,
    tx_hash            VARCHAR(66)   NOT NULL,
    log_index          INTEGER       NOT NULL,
    block_number       BIGINT        NOT NULL,
    block_time         TIMESTAMPTZ   NOT NULL,
    to_address         VARCHAR(42)   NOT NULL,
    amount_raw         NUMERIC(78,0) NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_reports_native_swept_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

COMMENT ON TABLE lending_reports_native_swept IS 'ReportsVerifier.NativeSwept；子图实体 VerifierNativeSwept。';

-- ---------------------------------------------------------------------------
-- 6. ChainlinkPriceOracle — FeedSet
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_chainlink_feed_set (
    id                 BIGSERIAL PRIMARY KEY,
    chain_id           BIGINT        NOT NULL,
    oracle_address     VARCHAR(42)   NOT NULL,
    tx_hash            VARCHAR(66)   NOT NULL,
    log_index          INTEGER       NOT NULL,
    block_number       BIGINT        NOT NULL,
    block_time         TIMESTAMPTZ   NOT NULL,
    asset_address      VARCHAR(42)   NOT NULL,
    feed_address       VARCHAR(42)   NOT NULL,
    stale_period_raw   NUMERIC(78,0) NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_chainlink_feed_set_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_chainlink_feed_set_asset ON lending_chainlink_feed_set (chain_id, asset_address);

COMMENT ON TABLE lending_chainlink_feed_set IS 'ChainlinkPriceOracle.FeedSet；子图实体 ChainlinkFeedSet。';

-- ---------------------------------------------------------------------------
-- 7. InterestRateStrategy — InterestRateStrategyDeployed（构造时事件；与 immutable 快照表互补）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_interest_rate_strategy_deployed (
    id                       BIGSERIAL PRIMARY KEY,
    chain_id                 BIGINT        NOT NULL,
    strategy_address         VARCHAR(42)   NOT NULL,
    tx_hash                  VARCHAR(66)   NOT NULL,
    log_index                INTEGER       NOT NULL,
    block_number             BIGINT        NOT NULL,
    block_time               TIMESTAMPTZ   NOT NULL,
    optimal_utilization_raw  NUMERIC(78,0) NOT NULL,
    base_borrow_rate_raw     NUMERIC(78,0) NOT NULL,
    slope1_raw               NUMERIC(78,0) NOT NULL,
    slope2_raw               NUMERIC(78,0) NOT NULL,
    reserve_factor_raw       NUMERIC(78,0) NOT NULL,
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_ir_strategy_deployed_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_ir_strategy_deployed_strategy ON lending_interest_rate_strategy_deployed (chain_id, strategy_address);

COMMENT ON TABLE lending_interest_rate_strategy_deployed IS
'InterestRateStrategy.InterestRateStrategyDeployed；子图实体 InterestRateStrategyDeployed。';

-- ---------------------------------------------------------------------------
-- 8. AToken / VariableDebtToken — Mint / Burn（储备级合约；子图模板实体）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_a_token_mint (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    token_address    VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    to_address       VARCHAR(42)   NOT NULL,
    scaled_amount_raw NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_a_token_mint_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_a_token_mint_token ON lending_a_token_mint (chain_id, token_address);

COMMENT ON TABLE lending_a_token_mint IS 'AToken.Mint；子图实体 ATokenMint。';

CREATE TABLE IF NOT EXISTS lending_a_token_burn (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    token_address    VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    from_address     VARCHAR(42)   NOT NULL,
    scaled_amount_raw NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_a_token_burn_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_a_token_burn_token ON lending_a_token_burn (chain_id, token_address);

COMMENT ON TABLE lending_a_token_burn IS 'AToken.Burn；子图实体 ATokenBurn。';

CREATE TABLE IF NOT EXISTS lending_variable_debt_token_mint (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    token_address    VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    to_address       VARCHAR(42)   NOT NULL,
    scaled_amount_raw NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_v_debt_mint_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_v_debt_mint_token ON lending_variable_debt_token_mint (chain_id, token_address);

COMMENT ON TABLE lending_variable_debt_token_mint IS 'VariableDebtToken.Mint；子图实体 VariableDebtTokenMint。';

CREATE TABLE IF NOT EXISTS lending_variable_debt_token_burn (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    token_address    VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    from_address     VARCHAR(42)   NOT NULL,
    scaled_amount_raw NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_v_debt_burn_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_v_debt_burn_token ON lending_variable_debt_token_burn (chain_id, token_address);

COMMENT ON TABLE lending_variable_debt_token_burn IS 'VariableDebtToken.Burn；子图实体 VariableDebtTokenBurn。';
