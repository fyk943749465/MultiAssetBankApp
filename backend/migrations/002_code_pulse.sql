-- Code Pulse 众筹读模型表（PostgreSQL）
-- 设计来源: backend/docs/code-pulse-design.md
-- 合约: CodePulseAdvanced @ Sepolia 0x3100b1FD5A2180dAc11820106579545D0f1C439b
--
-- 执行前请确认已连接目标库；可与 001_bank_ledger.sql 共存。

-- ---------------------------------------------------------------------------
-- 5. 主实体
-- ---------------------------------------------------------------------------

-- 提案主表：保存每个众筹提案的聚合态，包括发起人、目标金额、审核状态、
-- 待审核轮次信息等。作为提案列表与详情页的数据源。
CREATE TABLE IF NOT EXISTS cp_proposals (
    proposal_id                    BIGINT         PRIMARY KEY,
    organizer_address              VARCHAR(42)    NOT NULL,
    github_url                     TEXT           NOT NULL,
    github_url_hash                TEXT           NULL,
    target_wei                     NUMERIC(78, 0) NOT NULL,
    duration_seconds               BIGINT         NOT NULL,
    status                         TEXT           NOT NULL,
    status_code                    INTEGER        NOT NULL,
    last_campaign_id               BIGINT         NULL,
    current_round_count            INTEGER        NOT NULL DEFAULT 0,
    pending_round_target_wei       NUMERIC(78, 0) NULL,
    pending_round_duration_seconds BIGINT         NULL,
    round_review_state             TEXT           NULL,
    round_review_state_code        INTEGER        NULL,
    submitted_tx_hash              VARCHAR(66)    NULL,
    submitted_block_number         BIGINT         NULL,
    submitted_at                   TIMESTAMPTZ    NULL,
    reviewed_at                    TIMESTAMPTZ    NULL,
    approved_at                    TIMESTAMPTZ    NULL,
    rejected_at                    TIMESTAMPTZ    NULL,
    created_at                     TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at                     TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cp_proposals_status
    ON cp_proposals (status);
CREATE INDEX IF NOT EXISTS idx_cp_proposals_organizer_address
    ON cp_proposals (organizer_address);
CREATE INDEX IF NOT EXISTS idx_cp_proposals_submitted_at_desc
    ON cp_proposals (submitted_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_cp_proposals_last_campaign_id
    ON cp_proposals (last_campaign_id);

-- 众筹轮次表：每次 launchApprovedRound 后创建一条记录，保存该轮的目标金额、
-- 截止时间、已筹金额、结算状态等。一个提案可对应多轮 campaign。
CREATE TABLE IF NOT EXISTS cp_campaigns (
    campaign_id                 BIGINT         PRIMARY KEY,
    proposal_id                 BIGINT         NOT NULL
        REFERENCES cp_proposals (proposal_id) ON DELETE RESTRICT,
    round_index                 INTEGER        NOT NULL,
    organizer_address           VARCHAR(42)    NOT NULL,
    github_url                  TEXT           NOT NULL,
    target_wei                  NUMERIC(78, 0) NOT NULL,
    deadline_at                 TIMESTAMPTZ    NOT NULL,
    amount_raised_wei           NUMERIC(78, 0) NOT NULL DEFAULT 0,
    total_withdrawn_wei         NUMERIC(78, 0) NOT NULL DEFAULT 0,
    unclaimed_refund_pool_wei   NUMERIC(78, 0) NOT NULL DEFAULT 0,
    state                       TEXT           NOT NULL,
    state_code                  INTEGER        NOT NULL,
    donor_count                 INTEGER        NOT NULL DEFAULT 0,
    developer_count             INTEGER        NOT NULL DEFAULT 0,
    finalized_at                TIMESTAMPTZ    NULL,
    success_at                  TIMESTAMPTZ    NULL,
    dormant_funds_swept         BOOLEAN        NOT NULL DEFAULT false,
    launched_tx_hash            VARCHAR(66)    NOT NULL,
    launched_block_number       BIGINT         NOT NULL,
    launched_at                 TIMESTAMPTZ    NOT NULL,
    created_at                  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cp_campaigns_proposal_id
    ON cp_campaigns (proposal_id);
CREATE INDEX IF NOT EXISTS idx_cp_campaigns_state
    ON cp_campaigns (state);
CREATE INDEX IF NOT EXISTS idx_cp_campaigns_organizer_address
    ON cp_campaigns (organizer_address);
CREATE INDEX IF NOT EXISTS idx_cp_campaigns_deadline_at
    ON cp_campaigns (deadline_at);
CREATE INDEX IF NOT EXISTS idx_cp_campaigns_launched_at_desc
    ON cp_campaigns (launched_at DESC);

-- 捐款聚合表：按 (campaign, contributor) 聚合累计捐款与已退款金额。
-- 数据来源为 Donated / RefundClaimed 事件累加，支撑贡献榜与退款查询。
CREATE TABLE IF NOT EXISTS cp_contributions (
    campaign_id           BIGINT         NOT NULL
        REFERENCES cp_campaigns (campaign_id) ON DELETE RESTRICT,
    contributor_address   VARCHAR(42)    NOT NULL,
    total_contributed_wei NUMERIC(78, 0) NOT NULL DEFAULT 0,
    refund_claimed_wei    NUMERIC(78, 0) NOT NULL DEFAULT 0,
    last_donated_at       TIMESTAMPTZ    NULL,
    last_refund_at        TIMESTAMPTZ    NULL,
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, contributor_address)
);

CREATE INDEX IF NOT EXISTS idx_cp_contributions_contributor
    ON cp_contributions (contributor_address);

-- ---------------------------------------------------------------------------
-- 6. 里程碑与成员
-- ---------------------------------------------------------------------------

-- 提案里程碑定义表：保存 submitProposal 时的初始里程碑及后续 funding round
-- 的待审核里程碑。百分比由合约内部计算，通过链上 view 函数读取后写入。
CREATE TABLE IF NOT EXISTS cp_proposal_milestones (
    id               BIGSERIAL PRIMARY KEY,
    proposal_id      BIGINT         NOT NULL
        REFERENCES cp_proposals (proposal_id) ON DELETE RESTRICT,
    round_ordinal    INTEGER        NOT NULL,
    milestone_index  INTEGER        NOT NULL,
    description      TEXT           NOT NULL,
    percentage_raw   NUMERIC(20, 0) NOT NULL,
    source_type      TEXT           NOT NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT uq_cp_proposal_milestones_natural
        UNIQUE (proposal_id, round_ordinal, milestone_index, source_type),
    CONSTRAINT ck_cp_proposal_milestones_source_type
        CHECK (source_type IN ('proposal_initial', 'pending_round'))
);

CREATE INDEX IF NOT EXISTS idx_cp_proposal_milestones_proposal
    ON cp_proposal_milestones (proposal_id);

-- 众筹里程碑快照表：launch 时从 proposal_milestones 复制而来，记录每阶段的
-- 审批状态、解锁时间和是否已结清，是里程碑放款流程的核心表。
CREATE TABLE IF NOT EXISTS cp_campaign_milestones (
    campaign_id      BIGINT         NOT NULL
        REFERENCES cp_campaigns (campaign_id) ON DELETE RESTRICT,
    milestone_index  INTEGER        NOT NULL,
    description      TEXT           NOT NULL,
    percentage_raw   NUMERIC(20, 0) NOT NULL,
    approved         BOOLEAN        NOT NULL DEFAULT false,
    claimed          BOOLEAN        NOT NULL DEFAULT false,
    approved_at      TIMESTAMPTZ    NULL,
    unlock_at        TIMESTAMPTZ    NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, milestone_index)
);

CREATE INDEX IF NOT EXISTS idx_cp_campaign_milestones_pending_approval
    ON cp_campaign_milestones (campaign_id)
    WHERE approved = false;

-- 众筹开发者名单表：记录 campaign 下开发者的添加/移除历史。使用自增主键
-- 以支持同一开发者被 add → remove → re-add 的完整追溯。
CREATE TABLE IF NOT EXISTS cp_campaign_developers (
    id                 BIGSERIAL PRIMARY KEY,
    campaign_id        BIGINT         NOT NULL
        REFERENCES cp_campaigns (campaign_id) ON DELETE RESTRICT,
    developer_address  VARCHAR(42)    NOT NULL,
    is_active          BOOLEAN        NOT NULL DEFAULT true,
    added_tx_hash      VARCHAR(66)    NULL,
    removed_tx_hash    VARCHAR(66)    NULL,
    added_at           TIMESTAMPTZ    NULL,
    removed_at         TIMESTAMPTZ    NULL,
    created_at         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cp_campaign_developers_campaign_developer
    ON cp_campaign_developers (campaign_id, developer_address);
CREATE INDEX IF NOT EXISTS idx_cp_campaign_developers_developer_address
    ON cp_campaign_developers (developer_address);
CREATE INDEX IF NOT EXISTS idx_cp_campaign_developers_active
    ON cp_campaign_developers (campaign_id)
    WHERE is_active = true;

-- 里程碑领取记录表：每位开发者在每个阶段的实际领取金额与交易哈希。
-- 合约保证同一 (campaign, milestone, developer) 只能 claim 一次。
CREATE TABLE IF NOT EXISTS cp_milestone_claims (
    campaign_id         BIGINT         NOT NULL
        REFERENCES cp_campaigns (campaign_id) ON DELETE RESTRICT,
    milestone_index     INTEGER        NOT NULL,
    developer_address   VARCHAR(42)    NOT NULL,
    claimed_amount_wei  NUMERIC(78, 0) NOT NULL,
    claimed_tx_hash     VARCHAR(66)    NOT NULL,
    claimed_at          TIMESTAMPTZ    NOT NULL,
    PRIMARY KEY (campaign_id, milestone_index, developer_address)
);

CREATE INDEX IF NOT EXISTS idx_cp_milestone_claims_developer
    ON cp_milestone_claims (developer_address);
CREATE INDEX IF NOT EXISTS idx_cp_milestone_claims_campaign
    ON cp_milestone_claims (campaign_id);

-- ---------------------------------------------------------------------------
-- 7. 事件流水与同步
-- ---------------------------------------------------------------------------

-- 统一事件流水表：从子图增量同步的所有链上事件，提供时间线、审计与聚合表
-- 重建能力。不对 proposal_id / campaign_id 建外键，便于乱序入库。
CREATE TABLE IF NOT EXISTS cp_event_log (
    id               BIGSERIAL PRIMARY KEY,
    chain_id         BIGINT         NOT NULL,
    contract_address VARCHAR(42)    NOT NULL,
    event_name       TEXT           NOT NULL,
    proposal_id      BIGINT         NULL,
    campaign_id      BIGINT         NULL,
    wallet_address   VARCHAR(42)    NULL,
    entity_key       TEXT           NULL,
    tx_hash          VARCHAR(66)    NOT NULL,
    log_index        INTEGER        NOT NULL,
    block_number     BIGINT         NOT NULL,
    block_timestamp  TIMESTAMPTZ    NOT NULL,
    payload          JSONB          NOT NULL,
    source           TEXT           NOT NULL DEFAULT 'subgraph',
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT uq_cp_event_log_tx_log
        UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_cp_event_log_event_name
    ON cp_event_log (event_name);
CREATE INDEX IF NOT EXISTS idx_cp_event_log_proposal_id
    ON cp_event_log (proposal_id)
    WHERE proposal_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cp_event_log_campaign_id
    ON cp_event_log (campaign_id)
    WHERE campaign_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cp_event_log_wallet_address
    ON cp_event_log (wallet_address)
    WHERE wallet_address IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cp_event_log_block_number_desc
    ON cp_event_log (block_number DESC);

-- 同步游标表：记录后端从子图增量拉取事件的进度（最后区块、最后事件 ID）。
CREATE TABLE IF NOT EXISTS cp_sync_cursors (
    sync_name            TEXT        PRIMARY KEY,
    last_block_number    BIGINT      NULL,
    last_block_timestamp TIMESTAMPTZ NULL,
    last_event_id        TEXT        NULL,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- 8. 角色与画像
-- ---------------------------------------------------------------------------

-- 钱包画像表：保存用户的展示名、GitHub 用户名、头像等扩展信息。
CREATE TABLE IF NOT EXISTS cp_wallet_profiles (
    wallet_address VARCHAR(42) PRIMARY KEY,
    display_name   TEXT        NULL,
    github_username TEXT       NULL,
    avatar_url     TEXT        NULL,
    bio            TEXT        NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 钱包角色表：按 (钱包, 角色, 作用域) 记录身份，支撑前端"我的工作台"入口
-- 导航。角色从链上事件或 view 函数派生，scope 区分全局 / 提案 / campaign。
CREATE TABLE IF NOT EXISTS cp_wallet_roles (
    id             BIGSERIAL PRIMARY KEY,
    wallet_address VARCHAR(42) NOT NULL,
    role           TEXT        NOT NULL,
    scope_type     TEXT        NOT NULL,
    scope_id       TEXT        NULL,
    active         BOOLEAN     NOT NULL DEFAULT true,
    derived_from   TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cp_wallet_roles_logical
    ON cp_wallet_roles (wallet_address, role, scope_type, (COALESCE(scope_id, '')));

CREATE INDEX IF NOT EXISTS idx_cp_wallet_roles_wallet
    ON cp_wallet_roles (wallet_address);
CREATE INDEX IF NOT EXISTS idx_cp_wallet_roles_active
    ON cp_wallet_roles (wallet_address)
    WHERE active = true;

-- ---------------------------------------------------------------------------
-- 9. 交易尝试与错误
-- ---------------------------------------------------------------------------

-- 交易尝试表：记录每次链上动作的完整生命周期——从模拟、签名、广播到上链。
-- 模拟失败时保存解码后的 custom error 名称与参数，是错误展示的关键数据源。
CREATE TABLE IF NOT EXISTS cp_tx_attempts (
    id                   BIGSERIAL PRIMARY KEY,
    wallet_address       VARCHAR(42)    NOT NULL,
    role_snapshot        JSONB          NOT NULL,
    action               TEXT           NOT NULL,
    proposal_id          BIGINT         NULL,
    campaign_id          BIGINT         NULL,
    milestone_index      INTEGER        NULL,
    request_payload      JSONB          NOT NULL,
    simulation_ok        BOOLEAN        NULL,
    revert_error_name    TEXT           NULL,
    revert_error_args    JSONB          NULL,
    tx_hash              VARCHAR(66)    NULL,
    tx_status            TEXT           NOT NULL,
    receipt_block_number BIGINT         NULL,
    failure_stage        TEXT           NULL,
    failure_message      TEXT           NULL,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT ck_cp_tx_attempts_tx_status
        CHECK (tx_status IN (
            'simulated_failed',
            'pending_signature',
            'submitted',
            'mined_success',
            'mined_reverted',
            'dropped'
        )),
    CONSTRAINT ck_cp_tx_attempts_failure_stage
        CHECK (
            failure_stage IS NULL
            OR failure_stage IN (
                'validation',
                'simulation',
                'wallet_signature',
                'broadcast',
                'receipt'
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_cp_tx_attempts_wallet_created
    ON cp_tx_attempts (wallet_address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cp_tx_attempts_tx_hash
    ON cp_tx_attempts (tx_hash)
    WHERE tx_hash IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 10. 平台资金流水
-- ---------------------------------------------------------------------------

-- 平台资金流水表：记录 donateToPlatform / withdrawPlatformFunds 产生的每笔
-- 捐赠与提现明细，为管理员 dashboard 提供余额与流水数据。
CREATE TABLE IF NOT EXISTS cp_platform_fund_movements (
    id               BIGSERIAL PRIMARY KEY,
    direction        TEXT           NOT NULL,
    wallet_address   VARCHAR(42)    NOT NULL,
    amount_wei       NUMERIC(78, 0) NOT NULL,
    tx_hash          VARCHAR(66)    NOT NULL,
    log_index        INTEGER        NOT NULL,
    block_number     BIGINT         NOT NULL,
    block_timestamp  TIMESTAMPTZ    NOT NULL,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT ck_cp_platform_fund_movements_direction
        CHECK (direction IN ('donation', 'withdrawal')),
    CONSTRAINT uq_cp_platform_fund_movements_tx_log
        UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_cp_platform_fund_movements_direction
    ON cp_platform_fund_movements (direction);
CREATE INDEX IF NOT EXISTS idx_cp_platform_fund_movements_wallet
    ON cp_platform_fund_movements (wallet_address);
CREATE INDEX IF NOT EXISTS idx_cp_platform_fund_movements_block_desc
    ON cp_platform_fund_movements (block_number DESC);

-- ---------------------------------------------------------------------------
-- 11. 每日统计快照
-- ---------------------------------------------------------------------------

-- 每日统计快照表（可选）：定时任务写入，为首页 dashboard 提供提案总数、
-- 众筹数量、累计筹资/退款等历史趋势数据。
CREATE TABLE IF NOT EXISTS cp_snapshots_daily (
    snapshot_date               DATE           PRIMARY KEY,
    proposal_count              INTEGER        NOT NULL,
    campaign_count              INTEGER        NOT NULL,
    live_campaign_count         INTEGER        NOT NULL,
    successful_campaign_count   INTEGER        NOT NULL,
    failed_campaign_count       INTEGER        NOT NULL,
    total_raised_wei            NUMERIC(78, 0) NOT NULL,
    total_refunded_wei          NUMERIC(78, 0) NOT NULL,
    created_at                  TIMESTAMPTZ    NOT NULL DEFAULT now()
);
