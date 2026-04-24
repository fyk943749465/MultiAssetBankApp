-- 借贷模块读模型（PostgreSQL）— Base Sepolia Pool 与周边合约事件
-- 合约参考：hardhat-tutorial/contracts/lending（Pool、HybridPriceOracle、ChainlinkPriceOracle、
--           InterestRateStrategyFactory、ReportsVerifier、InterestRateStrategy）。
-- 子图参考：subgraph/lending/schema.graphql（实体字段与本迁移列名对齐，便于「子图优先展示、库兜底」）。
-- 子图后续扩展事件对应的 PG 表见 007_lending.sql（启动时先于 007 执行本文件）。
--
-- 数据原则（与 README 中 Bank / NFT 叙述一致，由应用层实现）：
--   · 列表 / 历史 / 仪表盘：优先读 The Graph；子图无数据或不可用时读本库。
--   · 借款、还款、清算、取款、存款等关键链上事实：以本库由 RPC 扫块写入的表为权威来源；子图仅作展示加速，不得反向覆盖已落库事件行。
--
-- 依赖：001_bank_ledger.sql 中的 chain_indexer_cursors（本文件不修改该表结构）。
--
-- ---------------------------------------------------------------------------
-- chain_indexer_cursors（无需 ALTER）
-- ---------------------------------------------------------------------------
-- 扫块程序连接 Base Sepolia 时，仍使用现有表：每条游标以 name 唯一区分业务与链。
-- 建议命名（Go 侧与运维约定即可）：
--   · Pool 主合约 logs：     lending_pool_rpc_<chainId>        例如 lending_pool_rpc_84532
--   · 仅扫 HybridOracle：    lending_hybrid_oracle_rpc_84532   （若拆进程）
--   · 仅扫 Factory：        lending_ir_strategy_factory_rpc_84532
-- 同一链可多 name 多游标，无需为 chain_id 增加字段；若未来需要「元数据」可另建 lending_indexer_jobs 表，非必需。

-- ---------------------------------------------------------------------------
-- 1. 平台关注的借贷相关合约注册（可选种子；索引器亦可写入）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_contracts (
    id                  BIGSERIAL PRIMARY KEY,
    chain_id            BIGINT        NOT NULL,
    address             VARCHAR(42)   NOT NULL,
    contract_kind       VARCHAR(40)   NOT NULL,
    display_label       VARCHAR(160)  NULL,
    deployed_block      BIGINT        NULL,
    deployed_tx_hash    VARCHAR(66)   NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_contracts_chain_address UNIQUE (chain_id, address),
    CONSTRAINT ck_lending_contracts_kind CHECK (contract_kind IN (
        'lending_pool',
        'hybrid_price_oracle',
        'chainlink_price_oracle',
        'reports_verifier',
        'interest_rate_strategy_factory',
        'interest_rate_strategy',
        'a_token',
        'variable_debt_token'
    ))
);

CREATE INDEX IF NOT EXISTS idx_lending_contracts_chain_kind
    ON lending_contracts (chain_id, contract_kind);

COMMENT ON TABLE lending_contracts IS
'借贷域已跟踪合约：Pool、预言机、验证器、利率工厂/策略实例；地址建议统一小写 0x 前缀 42 字符。';

-- Base Sepolia 默认部署（与 subgraph/lending/networks.json 及前端默认一致）；其它环境请改 chain_id/address 或依赖索引器写入。
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
-- 2. Pool — 核心业务事件（RPC 扫块落库 = 权威事实；列与子图实体对齐）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_supplies (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    user_address     VARCHAR(42)   NOT NULL,
    amount_raw       NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_supplies_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_supplies_block ON lending_supplies (chain_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_lending_supplies_user ON lending_supplies (chain_id, user_address);
CREATE INDEX IF NOT EXISTS idx_lending_supplies_asset ON lending_supplies (chain_id, asset_address);

COMMENT ON TABLE lending_supplies IS 'Pool.Supply；子图实体 Supply；键 (chain_id, tx_hash, log_index) 幂等。';

CREATE TABLE IF NOT EXISTS lending_withdrawals (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    user_address     VARCHAR(42)   NOT NULL,
    amount_raw       NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_withdrawals_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_withdrawals_block ON lending_withdrawals (chain_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_lending_withdrawals_user ON lending_withdrawals (chain_id, user_address);

COMMENT ON TABLE lending_withdrawals IS 'Pool.Withdraw；子图实体 Withdraw。';

CREATE TABLE IF NOT EXISTS lending_borrows (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    user_address     VARCHAR(42)   NOT NULL,
    amount_raw       NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_borrows_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_borrows_block ON lending_borrows (chain_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_lending_borrows_user ON lending_borrows (chain_id, user_address);

COMMENT ON TABLE lending_borrows IS 'Pool.Borrow；子图实体 Borrow。';

CREATE TABLE IF NOT EXISTS lending_repays (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    user_address     VARCHAR(42)   NOT NULL,
    amount_raw       NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_repays_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_repays_block ON lending_repays (chain_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_lending_repays_user ON lending_repays (chain_id, user_address);

COMMENT ON TABLE lending_repays IS 'Pool.Repay；子图实体 Repay。';

CREATE TABLE IF NOT EXISTS lending_liquidations (
    id                         BIGSERIAL PRIMARY KEY,
    chain_id                   BIGINT        NOT NULL,
    pool_address               VARCHAR(42)   NOT NULL,
    tx_hash                    VARCHAR(66)   NOT NULL,
    log_index                  INTEGER       NOT NULL,
    block_number               BIGINT        NOT NULL,
    block_time                 TIMESTAMPTZ   NOT NULL,
    collateral_asset_address   VARCHAR(42)   NOT NULL,
    debt_asset_address         VARCHAR(42)   NOT NULL,
    borrower_address           VARCHAR(42)   NOT NULL,
    liquidator_address         VARCHAR(42)   NOT NULL,
    debt_covered_raw           NUMERIC(78,0) NOT NULL,
    collateral_to_liquidator_raw NUMERIC(78,0) NOT NULL,
    collateral_protocol_fee_raw NUMERIC(78,0) NOT NULL,
    created_at                 TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_liquidations_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_liquidations_block ON lending_liquidations (chain_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_lending_liquidations_borrower ON lending_liquidations (chain_id, borrower_address);

COMMENT ON TABLE lending_liquidations IS 'Pool.LiquidationCall；子图实体 LiquidationCall。';

CREATE TABLE IF NOT EXISTS lending_paused (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    account_address  VARCHAR(42)   NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_paused_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_paused_block ON lending_paused (chain_id, block_number DESC);

COMMENT ON TABLE lending_paused IS 'Pool.Pausable.Paused(address)；子图实体 Paused。';

CREATE TABLE IF NOT EXISTS lending_unpaused (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    account_address  VARCHAR(42)   NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_unpaused_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_unpaused_block ON lending_unpaused (chain_id, block_number DESC);

COMMENT ON TABLE lending_unpaused IS 'Pool.Pausable.Unpaused(address)；子图实体 Unpaused。';

CREATE TABLE IF NOT EXISTS lending_protocol_fee_recipient_updated (
    id                 BIGSERIAL PRIMARY KEY,
    chain_id           BIGINT        NOT NULL,
    pool_address       VARCHAR(42)   NOT NULL,
    tx_hash            VARCHAR(66)   NOT NULL,
    log_index          INTEGER       NOT NULL,
    block_number       BIGINT        NOT NULL,
    block_time         TIMESTAMPTZ   NOT NULL,
    new_recipient_address VARCHAR(42) NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_protocol_fee_recipient_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

COMMENT ON TABLE lending_protocol_fee_recipient_updated IS 'Pool.ProtocolFeeRecipientUpdated；子图同名实体。';

CREATE TABLE IF NOT EXISTS lending_reserve_caps_updated (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    supply_cap_raw   NUMERIC(78,0) NOT NULL,
    borrow_cap_raw   NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_reserve_caps_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_reserve_caps_asset ON lending_reserve_caps_updated (chain_id, asset_address);

COMMENT ON TABLE lending_reserve_caps_updated IS 'Pool.ReserveCapsUpdated；子图同名实体。';

CREATE TABLE IF NOT EXISTS lending_reserve_liquidation_protocol_fee_updated (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    fee_bps_raw      NUMERIC(78,0) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_reserve_liq_fee_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

COMMENT ON TABLE lending_reserve_liquidation_protocol_fee_updated IS
'Pool.ReserveLiquidationProtocolFeeUpdated；子图实体 ReserveLiquidationProtocolFeeUpdated。';

CREATE TABLE IF NOT EXISTS lending_user_emode_set (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    pool_address     VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    user_address     VARCHAR(42)   NOT NULL,
    category_id      SMALLINT      NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_user_emode_set_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_user_emode_set_user ON lending_user_emode_set (chain_id, user_address);

COMMENT ON TABLE lending_user_emode_set IS 'Pool.UserEModeSet；子图实体 UserEModeSet；category_id 对应 uint8。';

-- ---------------------------------------------------------------------------
-- 3. 多合约 — OwnershipTransferred（子图带 emitter；Hybrid / Chainlink / Factory / Verifier 共用事件形态）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_ownership_transferred (
    id                  BIGSERIAL PRIMARY KEY,
    chain_id            BIGINT        NOT NULL,
    emitter_address     VARCHAR(42)   NOT NULL,
    tx_hash             VARCHAR(66)   NOT NULL,
    log_index           INTEGER       NOT NULL,
    block_number        BIGINT        NOT NULL,
    block_time          TIMESTAMPTZ   NOT NULL,
    previous_owner_address VARCHAR(42) NOT NULL,
    new_owner_address   VARCHAR(42)   NOT NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_ownership_transferred_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_ownership_emitter ON lending_ownership_transferred (chain_id, emitter_address);

COMMENT ON TABLE lending_ownership_transferred IS
'OpenZeppelin Ownable.OwnershipTransferred；子图实体 OwnershipTransferred；emitter_address 区分来源合约。';

-- ---------------------------------------------------------------------------
-- 4. HybridPriceOracle — StreamConfigUpdated / StreamPriceFallbackToFeed（Chainlink 侧仅 Ownership，见上表）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_hybrid_stream_config_updated (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    oracle_address   VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    stream_feed_id_hex VARCHAR(66) NOT NULL,
    price_decimals   SMALLINT      NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_hybrid_stream_cfg_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_hybrid_stream_cfg_asset ON lending_hybrid_stream_config_updated (chain_id, asset_address);

COMMENT ON TABLE lending_hybrid_stream_config_updated IS 'HybridPriceOracle.StreamConfigUpdated；子图实体 StreamConfigUpdated；stream_feed_id_hex 为 0x+64hex。';

CREATE TABLE IF NOT EXISTS lending_hybrid_stream_price_fallback_to_feed (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT        NOT NULL,
    oracle_address   VARCHAR(42)   NOT NULL,
    tx_hash          VARCHAR(66)   NOT NULL,
    log_index        INTEGER       NOT NULL,
    block_number     BIGINT        NOT NULL,
    block_time       TIMESTAMPTZ   NOT NULL,
    asset_address    VARCHAR(42)   NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_hybrid_fallback_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

COMMENT ON TABLE lending_hybrid_stream_price_fallback_to_feed IS
'HybridPriceOracle.StreamPriceFallbackToFeed；子图实体 StreamPriceFallbackToFeed。';

-- ---------------------------------------------------------------------------
-- 5. InterestRateStrategyFactory — StrategyCreated（子图 InterestRateStrategyCreated）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_interest_rate_strategy_created (
    id                    BIGSERIAL PRIMARY KEY,
    chain_id              BIGINT        NOT NULL,
    factory_address       VARCHAR(42)   NOT NULL,
    tx_hash               VARCHAR(66)   NOT NULL,
    log_index             INTEGER       NOT NULL,
    block_number          BIGINT        NOT NULL,
    block_time            TIMESTAMPTZ   NOT NULL,
    strategy_address        VARCHAR(42)   NOT NULL,
    strategy_index_raw      NUMERIC(78,0) NOT NULL,
    optimal_utilization_raw NUMERIC(78,0) NOT NULL,
    base_borrow_rate_raw    NUMERIC(78,0) NOT NULL,
    slope1_raw              NUMERIC(78,0) NOT NULL,
    slope2_raw              NUMERIC(78,0) NOT NULL,
    reserve_factor_raw      NUMERIC(78,0) NOT NULL,
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_ir_strategy_created_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_lending_ir_strategy_created_strategy ON lending_interest_rate_strategy_created (chain_id, strategy_address);

COMMENT ON TABLE lending_interest_rate_strategy_created IS
'InterestRateStrategyFactory.StrategyCreated；子图实体 InterestRateStrategyCreated；strategy_index_raw 对应事件 id。';

-- ---------------------------------------------------------------------------
-- 6. InterestRateStrategy — 无链上事件时的不可变参数快照（与子图 InterestRateStrategyImmutableParams 对齐）
--    可由扫块程序在部署高度调用 view 写入一次，或后续由运维脚本插入。
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lending_interest_rate_strategy_immutable_params (
    id                    BIGSERIAL PRIMARY KEY,
    chain_id              BIGINT        NOT NULL,
    strategy_address      VARCHAR(42)   NOT NULL,
    optimal_utilization_raw NUMERIC(78,0) NOT NULL,
    base_borrow_rate_raw    NUMERIC(78,0) NOT NULL,
    slope1_raw              NUMERIC(78,0) NOT NULL,
    slope2_raw              NUMERIC(78,0) NOT NULL,
    reserve_factor_raw      NUMERIC(78,0) NOT NULL,
    source_block_number     BIGINT        NOT NULL,
    source_block_time       TIMESTAMPTZ   NOT NULL,
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_lending_ir_strategy_immutable_chain_addr UNIQUE (chain_id, strategy_address)
);

COMMENT ON TABLE lending_interest_rate_strategy_immutable_params IS
'利率策略构造参数快照；子图实体 InterestRateStrategyImmutableParams；与工厂 StrategyCreated 可交叉校验。';
