-- MultiAssetBank 链上充值 / 提现索引表（PostgreSQL）
-- 事件来源（Sepolia 已验证合约 0x668A7A8372C41EE0be46a4eA34e6eafeaA4E9748）:
--   event Deposited(address indexed token, address indexed user, uint256 amount);
--   event Withdrawn(address indexed token, address indexed user, uint256 amount);
--
-- The Graph 方案说明（可选，不必执行本文件即可用后端直连索引）:
-- 1) 在 https://thegraph.com/studio 创建 Subgraph，选 Sepolia。
-- 2) 用 graph-cli 初始化，在 schema.graphql 定义 Deposit/Withdraw 实体，与下面字段对齐。
-- 3) 在 subgraph.yaml 指向 MultiAssetBank 地址与起始区块；handlers 里处理 Deposited / Withdrawn。
-- 4) 部署后得到 GraphQL endpoint；你的 Go 服务可定时拉 GraphQL 写入本库，或用 webhook（需自建桥）。
-- 当前仓库已实现 go-ethereum FilterLogs 索引，无需 The Graph 即可落库。

CREATE TABLE IF NOT EXISTS bank_deposits (
    id              BIGSERIAL PRIMARY KEY,
    chain_id        BIGINT        NOT NULL,
    tx_hash         VARCHAR(66)   NOT NULL,
    log_index       INTEGER       NOT NULL,
    block_number    BIGINT        NOT NULL,
    block_time      TIMESTAMPTZ   NOT NULL,
    token_address   VARCHAR(42)   NOT NULL,
    user_address    VARCHAR(42)   NOT NULL,
    amount_raw      NUMERIC(78,0) NOT NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_bank_deposits_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_bank_deposits_block ON bank_deposits (block_number DESC);
CREATE INDEX IF NOT EXISTS idx_bank_deposits_user ON bank_deposits (user_address);

CREATE TABLE IF NOT EXISTS bank_withdrawals (
    id              BIGSERIAL PRIMARY KEY,
    chain_id        BIGINT        NOT NULL,
    tx_hash         VARCHAR(66)   NOT NULL,
    log_index       INTEGER       NOT NULL,
    block_number    BIGINT        NOT NULL,
    block_time      TIMESTAMPTZ   NOT NULL,
    token_address   VARCHAR(42)   NOT NULL,
    user_address    VARCHAR(42)   NOT NULL,
    amount_raw      NUMERIC(78,0) NOT NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_bank_withdrawals_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_bank_withdrawals_block ON bank_withdrawals (block_number DESC);
CREATE INDEX IF NOT EXISTS idx_bank_withdrawals_user ON bank_withdrawals (user_address);

-- 索引器进度（每个合约+链一条；name 由后端写入）
CREATE TABLE IF NOT EXISTS chain_indexer_cursors (
    id                   BIGSERIAL PRIMARY KEY,
    name                 VARCHAR(128) NOT NULL UNIQUE,
    last_scanned_block   BIGINT       NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now()
);
