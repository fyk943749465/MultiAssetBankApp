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

COMMENT ON TABLE cp_proposals IS
'Code Pulse 提案主表：链上提案 id 为主键，聚合发起人、GitHub、目标金额与时长、审核与轮次状态；供列表/详情/审核后台使用。';

COMMENT ON COLUMN cp_proposals.proposal_id IS '链上提案合约中的 proposalId，与合约一致。';
COMMENT ON COLUMN cp_proposals.organizer_address IS '发起人钱包地址（0x 前缀 42 字符）。';
COMMENT ON COLUMN cp_proposals.github_url IS '提案关联的 GitHub 仓库或说明链接。';
COMMENT ON COLUMN cp_proposals.github_url_hash IS 'github_url 的哈希，用于去重或校验（可选）。';
COMMENT ON COLUMN cp_proposals.target_wei IS '当前轮或初始目标募集金额（wei）。';
COMMENT ON COLUMN cp_proposals.duration_seconds IS '众筹持续时长（秒）。';
COMMENT ON COLUMN cp_proposals.status IS '提案可读状态文案（与子图/合约枚举映射）。';
COMMENT ON COLUMN cp_proposals.status_code IS '提案状态数值码，便于筛选与排序。';
COMMENT ON COLUMN cp_proposals.last_campaign_id IS '最近一次已创建众筹轮次的 campaign_id（若无则为 NULL）。';
COMMENT ON COLUMN cp_proposals.current_round_count IS '已进行或已规划的轮次数（业务约定由同步逻辑维护）。';
COMMENT ON COLUMN cp_proposals.pending_round_target_wei IS '待审核新轮的目标金额（wei），无待审轮时为 NULL。';
COMMENT ON COLUMN cp_proposals.pending_round_duration_seconds IS '待审核新轮的持续时长（秒）。';
COMMENT ON COLUMN cp_proposals.round_review_state IS '轮次审核可读状态（若有）。';
COMMENT ON COLUMN cp_proposals.round_review_state_code IS '轮次审核状态码。';
COMMENT ON COLUMN cp_proposals.submitted_tx_hash IS '提案提交上链的交易哈希。';
COMMENT ON COLUMN cp_proposals.submitted_block_number IS '提案提交所在区块号。';
COMMENT ON COLUMN cp_proposals.submitted_at IS '提案提交时间（可由区块时间或入库时间推导）。';
COMMENT ON COLUMN cp_proposals.reviewed_at IS '完成审核（通过或拒绝流程节点）的时间。';
COMMENT ON COLUMN cp_proposals.approved_at IS '审核通过时间。';
COMMENT ON COLUMN cp_proposals.rejected_at IS '审核拒绝时间。';
COMMENT ON COLUMN cp_proposals.created_at IS '本行首次写入时间。';
COMMENT ON COLUMN cp_proposals.updated_at IS '本行任意字段最后更新时间。';

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

COMMENT ON TABLE cp_campaigns IS
'众筹轮次表：每次 launchApprovedRound 后一行；记录目标、截止、已筹、提现与退款池、状态及人数统计；一提案可多轮。';

COMMENT ON COLUMN cp_campaigns.campaign_id IS '链上 campaignId，主键。';
COMMENT ON COLUMN cp_campaigns.proposal_id IS '所属提案 id，外键 cp_proposals。';
COMMENT ON COLUMN cp_campaigns.round_index IS '在该提案下的轮次序号（从 0 或 1 起算由业务约定）。';
COMMENT ON COLUMN cp_campaigns.organizer_address IS '本轮展示用组织者地址（通常与提案一致）。';
COMMENT ON COLUMN cp_campaigns.github_url IS '本轮关联的 GitHub 链接快照。';
COMMENT ON COLUMN cp_campaigns.target_wei IS '本轮募集目标（wei）。';
COMMENT ON COLUMN cp_campaigns.deadline_at IS '本轮截止时间（UTC）。';
COMMENT ON COLUMN cp_campaigns.amount_raised_wei IS '本轮已筹集总额（wei），由捐款事件累加。';
COMMENT ON COLUMN cp_campaigns.total_withdrawn_wei IS '组织者已从本轮已提取金额（wei）。';
COMMENT ON COLUMN cp_campaigns.unclaimed_refund_pool_wei IS '未领取退款池余额（wei），由业务/合约语义维护。';
COMMENT ON COLUMN cp_campaigns.state IS '本轮可读状态（进行中/成功/失败等）。';
COMMENT ON COLUMN cp_campaigns.state_code IS '本轮状态数值码。';
COMMENT ON COLUMN cp_campaigns.donor_count IS '独立捐款人数量（去重统计）。';
COMMENT ON COLUMN cp_campaigns.developer_count IS '参与本轮的开发者人数（业务统计）。';
COMMENT ON COLUMN cp_campaigns.finalized_at IS '本轮已结算/终结的时间。';
COMMENT ON COLUMN cp_campaigns.success_at IS '本轮判定成功的时间（若有）。';
COMMENT ON COLUMN cp_campaigns.dormant_funds_swept IS '休眠资金是否已被清扫（合约相关语义）。';
COMMENT ON COLUMN cp_campaigns.launched_tx_hash IS '本轮启动（launch）交易哈希。';
COMMENT ON COLUMN cp_campaigns.launched_block_number IS '本轮启动所在区块号。';
COMMENT ON COLUMN cp_campaigns.launched_at IS '本轮启动时间。';
COMMENT ON COLUMN cp_campaigns.created_at IS '本行首次写入时间。';
COMMENT ON COLUMN cp_campaigns.updated_at IS '本行最后更新时间。';

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

COMMENT ON TABLE cp_contributions IS
'捐款聚合表：按 (campaign, contributor) 唯一一行，累计捐款与已领退款；由 Donated、RefundClaimed 等事件更新。';

COMMENT ON COLUMN cp_contributions.campaign_id IS '众筹轮次 id。';
COMMENT ON COLUMN cp_contributions.contributor_address IS '捐款人钱包地址。';
COMMENT ON COLUMN cp_contributions.total_contributed_wei IS '该地址在本轮累计捐款（wei）。';
COMMENT ON COLUMN cp_contributions.refund_claimed_wei IS '该地址在本轮已领取退款总额（wei）。';
COMMENT ON COLUMN cp_contributions.last_donated_at IS '最近一次捐款时间。';
COMMENT ON COLUMN cp_contributions.last_refund_at IS '最近一次领取退款时间。';
COMMENT ON COLUMN cp_contributions.created_at IS '本行首次写入时间。';
COMMENT ON COLUMN cp_contributions.updated_at IS '本行最后更新时间。';

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

COMMENT ON TABLE cp_proposal_milestones IS
'提案里程碑定义：提交提案及待审核新轮时的阶段划分与描述；launch 时会复制到 cp_campaign_milestones。';

COMMENT ON COLUMN cp_proposal_milestones.id IS '本表自增主键。';
COMMENT ON COLUMN cp_proposal_milestones.proposal_id IS '所属提案 id。';
COMMENT ON COLUMN cp_proposal_milestones.round_ordinal IS '轮次序号，区分初始提案与后续 funding round。';
COMMENT ON COLUMN cp_proposal_milestones.milestone_index IS '该轮内里程碑序号。';
COMMENT ON COLUMN cp_proposal_milestones.description IS '里程碑文字说明。';
COMMENT ON COLUMN cp_proposal_milestones.percentage_raw IS '链上/合约使用的百分比原始值（大整数，非 0–100 小数）。';
COMMENT ON COLUMN cp_proposal_milestones.source_type IS
'proposal_initial=首次提交；pending_round=待审核的新一轮里程碑。';
COMMENT ON COLUMN cp_proposal_milestones.created_at IS '本行首次写入时间。';
COMMENT ON COLUMN cp_proposal_milestones.updated_at IS '本行最后更新时间。';

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

COMMENT ON TABLE cp_campaign_milestones IS
'众筹轮次里程碑快照：launch 时自提案定义复制；记录审批、解锁与是否已 claim，驱动里程碑放款。';

COMMENT ON COLUMN cp_campaign_milestones.campaign_id IS '众筹轮次 id。';
COMMENT ON COLUMN cp_campaign_milestones.milestone_index IS '该轮内里程碑序号，与主键组合唯一。';
COMMENT ON COLUMN cp_campaign_milestones.description IS '里程碑说明快照。';
COMMENT ON COLUMN cp_campaign_milestones.percentage_raw IS '百分比原始值快照。';
COMMENT ON COLUMN cp_campaign_milestones.approved IS '该里程碑是否已通过审批（可放款前置条件）。';
COMMENT ON COLUMN cp_campaign_milestones.claimed IS '该里程碑款项是否已被开发者领取完毕（业务语义）。';
COMMENT ON COLUMN cp_campaign_milestones.approved_at IS '审批通过时间。';
COMMENT ON COLUMN cp_campaign_milestones.unlock_at IS '可领取解锁时间。';
COMMENT ON COLUMN cp_campaign_milestones.created_at IS '本行首次写入时间。';
COMMENT ON COLUMN cp_campaign_milestones.updated_at IS '本行最后更新时间。';

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

COMMENT ON TABLE cp_campaign_developers IS
'众筹轮次开发者名单变更历史：支持同一人多次加入/移出，用 is_active 与 tx 时间追溯。';

COMMENT ON COLUMN cp_campaign_developers.id IS '本表自增主键。';
COMMENT ON COLUMN cp_campaign_developers.campaign_id IS '所属众筹轮次 id。';
COMMENT ON COLUMN cp_campaign_developers.developer_address IS '开发者钱包地址。';
COMMENT ON COLUMN cp_campaign_developers.is_active IS '当前是否仍在名单中（true=在册）。';
COMMENT ON COLUMN cp_campaign_developers.added_tx_hash IS '加入名单时的链上交易哈希。';
COMMENT ON COLUMN cp_campaign_developers.removed_tx_hash IS '移出名单时的链上交易哈希。';
COMMENT ON COLUMN cp_campaign_developers.added_at IS '加入时间。';
COMMENT ON COLUMN cp_campaign_developers.removed_at IS '移出时间。';
COMMENT ON COLUMN cp_campaign_developers.created_at IS '本行首次写入时间。';
COMMENT ON COLUMN cp_campaign_developers.updated_at IS '本行最后更新时间。';

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

COMMENT ON TABLE cp_milestone_claims IS
'里程碑领取记录：每个 (campaign, milestone_index, developer) 至多一条，与链上 claim 一致。';

COMMENT ON COLUMN cp_milestone_claims.campaign_id IS '众筹轮次 id。';
COMMENT ON COLUMN cp_milestone_claims.milestone_index IS '领取所针对的里程碑序号。';
COMMENT ON COLUMN cp_milestone_claims.developer_address IS '领取人钱包地址。';
COMMENT ON COLUMN cp_milestone_claims.claimed_amount_wei IS '实际领取金额（wei）。';
COMMENT ON COLUMN cp_milestone_claims.claimed_tx_hash IS '领取交易哈希。';
COMMENT ON COLUMN cp_milestone_claims.claimed_at IS '领取完成时间。';

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

COMMENT ON TABLE cp_event_log IS
'链上事件统一流水：从子图等增量同步；payload 存原始参数；不对 proposal/campaign 建外键以支持乱序入库与重放。';

COMMENT ON COLUMN cp_event_log.id IS '本表自增主键。';
COMMENT ON COLUMN cp_event_log.chain_id IS 'EVM chain id。';
COMMENT ON COLUMN cp_event_log.contract_address IS '发出事件的合约地址。';
COMMENT ON COLUMN cp_event_log.event_name IS '事件名称（与 ABI/子图一致）。';
COMMENT ON COLUMN cp_event_log.proposal_id IS '关联提案 id（若该事件可解析）。';
COMMENT ON COLUMN cp_event_log.campaign_id IS '关联众筹轮次 id（若可解析）。';
COMMENT ON COLUMN cp_event_log.wallet_address IS '事件主要相关钱包（若有）。';
COMMENT ON COLUMN cp_event_log.entity_key IS '子图或业务侧实体键，用于去重或关联。';
COMMENT ON COLUMN cp_event_log.tx_hash IS '交易哈希。';
COMMENT ON COLUMN cp_event_log.log_index IS '日志在交易内的索引。';
COMMENT ON COLUMN cp_event_log.block_number IS '区块号。';
COMMENT ON COLUMN cp_event_log.block_timestamp IS '区块时间戳。';
COMMENT ON COLUMN cp_event_log.payload IS '事件参数 JSON，完整保留链上字段。';
COMMENT ON COLUMN cp_event_log.source IS '数据来源标识，如 subgraph、direct_indexer。';
COMMENT ON COLUMN cp_event_log.created_at IS '入库时间。';

-- 同步游标表：记录后端从子图增量拉取事件的进度（最后区块、最后事件 ID）。
CREATE TABLE IF NOT EXISTS cp_sync_cursors (
    sync_name            TEXT        PRIMARY KEY,
    last_block_number    BIGINT      NULL,
    last_block_timestamp TIMESTAMPTZ NULL,
    last_event_id        TEXT        NULL,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE cp_sync_cursors IS
'子图/事件同步游标：按 sync_name 区分任务，记录已处理到的块与时间、子图 last_event_id。';

COMMENT ON COLUMN cp_sync_cursors.sync_name IS '同步任务唯一名，如 code_pulse_subgraph_v1。';
COMMENT ON COLUMN cp_sync_cursors.last_block_number IS '已处理到的最高区块号。';
COMMENT ON COLUMN cp_sync_cursors.last_block_timestamp IS '该块对应时间戳（可选）。';
COMMENT ON COLUMN cp_sync_cursors.last_event_id IS '子图侧最后一条事件 id，用于增量游标。';
COMMENT ON COLUMN cp_sync_cursors.updated_at IS '游标最后更新时间。';

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

COMMENT ON TABLE cp_wallet_profiles IS
'钱包链下画像：展示名、GitHub、头像等，与链上地址一对一；非合约状态。';

COMMENT ON COLUMN cp_wallet_profiles.wallet_address IS '钱包地址，主键。';
COMMENT ON COLUMN cp_wallet_profiles.display_name IS '用户展示昵称。';
COMMENT ON COLUMN cp_wallet_profiles.github_username IS 'GitHub 用户名。';
COMMENT ON COLUMN cp_wallet_profiles.avatar_url IS '头像图片 URL。';
COMMENT ON COLUMN cp_wallet_profiles.bio IS '个人简介。';
COMMENT ON COLUMN cp_wallet_profiles.created_at IS '档案创建时间。';
COMMENT ON COLUMN cp_wallet_profiles.updated_at IS '档案最后更新时间。';

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

COMMENT ON TABLE cp_wallet_roles IS
'钱包在系统中的角色：结合 scope 区分全局、某提案或某 campaign；用于工作台入口与权限展示。';

COMMENT ON COLUMN cp_wallet_roles.id IS '本表自增主键。';
COMMENT ON COLUMN cp_wallet_roles.wallet_address IS '钱包地址。';
COMMENT ON COLUMN cp_wallet_roles.role IS '角色标识，如 organizer、contributor、developer。';
COMMENT ON COLUMN cp_wallet_roles.scope_type IS '作用域类型：global、proposal、campaign 等。';
COMMENT ON COLUMN cp_wallet_roles.scope_id IS '作用域 id（如 proposal_id 字符串），全局时可为 NULL。';
COMMENT ON COLUMN cp_wallet_roles.active IS '该条角色绑定是否仍有效。';
COMMENT ON COLUMN cp_wallet_roles.derived_from IS '角色来源说明，如某事件名或 view 同步。';
COMMENT ON COLUMN cp_wallet_roles.created_at IS '本行创建时间。';
COMMENT ON COLUMN cp_wallet_roles.updated_at IS '本行最后更新时间。';

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

COMMENT ON TABLE cp_tx_attempts IS
'dApp 链上操作尝试全链路：模拟、签名、广播、回执；失败时记录 revert 与阶段，便于用户排障。';

COMMENT ON COLUMN cp_tx_attempts.id IS '本表自增主键。';
COMMENT ON COLUMN cp_tx_attempts.wallet_address IS '发起操作的钱包地址。';
COMMENT ON COLUMN cp_tx_attempts.role_snapshot IS '发起时刻的角色快照 JSON。';
COMMENT ON COLUMN cp_tx_attempts.action IS '业务动作标识，如 donate、claimMilestone。';
COMMENT ON COLUMN cp_tx_attempts.proposal_id IS '关联提案 id（若适用）。';
COMMENT ON COLUMN cp_tx_attempts.campaign_id IS '关联众筹轮次 id（若适用）。';
COMMENT ON COLUMN cp_tx_attempts.milestone_index IS '关联里程碑序号（若适用）。';
COMMENT ON COLUMN cp_tx_attempts.request_payload IS '请求参数 JSON（编码前或意图描述）。';
COMMENT ON COLUMN cp_tx_attempts.simulation_ok IS '链上模拟是否通过；NULL 表示未模拟。';
COMMENT ON COLUMN cp_tx_attempts.revert_error_name IS '模拟或执行 revert 的自定义错误名。';
COMMENT ON COLUMN cp_tx_attempts.revert_error_args IS 'revert 错误参数 JSON。';
COMMENT ON COLUMN cp_tx_attempts.tx_hash IS '已广播交易哈希；未广播则为 NULL。';
COMMENT ON COLUMN cp_tx_attempts.tx_status IS
'交易生命周期：simulated_failed / pending_signature / submitted / mined_success / mined_reverted / dropped。';
COMMENT ON COLUMN cp_tx_attempts.receipt_block_number IS '上链后所在区块号。';
COMMENT ON COLUMN cp_tx_attempts.failure_stage IS
'失败阶段：validation / simulation / wallet_signature / broadcast / receipt。';
COMMENT ON COLUMN cp_tx_attempts.failure_message IS '人类可读失败说明。';
COMMENT ON COLUMN cp_tx_attempts.created_at IS '尝试记录创建时间。';
COMMENT ON COLUMN cp_tx_attempts.updated_at IS '尝试记录最后更新时间。';

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

COMMENT ON TABLE cp_platform_fund_movements IS
'平台资金池流水：用户 donateToPlatform 与管理员 withdrawPlatformFunds 等明细，供后台对账。';

COMMENT ON COLUMN cp_platform_fund_movements.id IS '本表自增主键。';
COMMENT ON COLUMN cp_platform_fund_movements.direction IS 'donation=流入；withdrawal=流出。';
COMMENT ON COLUMN cp_platform_fund_movements.wallet_address IS '发起捐赠或接收/操作提现的钱包地址。';
COMMENT ON COLUMN cp_platform_fund_movements.amount_wei IS '金额（wei），方向由 direction 解释。';
COMMENT ON COLUMN cp_platform_fund_movements.tx_hash IS '链上交易哈希。';
COMMENT ON COLUMN cp_platform_fund_movements.log_index IS '事件在交易中的 log_index。';
COMMENT ON COLUMN cp_platform_fund_movements.block_number IS '所在区块号。';
COMMENT ON COLUMN cp_platform_fund_movements.block_timestamp IS '区块时间戳。';
COMMENT ON COLUMN cp_platform_fund_movements.created_at IS '入库时间。';

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

COMMENT ON TABLE cp_snapshots_daily IS
'按日聚合快照：定时任务写入，供首页或 dashboard 展示历史趋势（可选）。';

COMMENT ON COLUMN cp_snapshots_daily.snapshot_date IS '统计日期（UTC 日历日）。';
COMMENT ON COLUMN cp_snapshots_daily.proposal_count IS '截至该日的提案总数或当日累计（由任务定义）。';
COMMENT ON COLUMN cp_snapshots_daily.campaign_count IS '众筹轮次数量指标。';
COMMENT ON COLUMN cp_snapshots_daily.live_campaign_count IS '进行中轮次数量。';
COMMENT ON COLUMN cp_snapshots_daily.successful_campaign_count IS '成功轮次数量。';
COMMENT ON COLUMN cp_snapshots_daily.failed_campaign_count IS '失败轮次数量。';
COMMENT ON COLUMN cp_snapshots_daily.total_raised_wei IS '累计或当日筹资（wei），与任务定义一致。';
COMMENT ON COLUMN cp_snapshots_daily.total_refunded_wei IS '累计或当日退款（wei）。';
COMMENT ON COLUMN cp_snapshots_daily.created_at IS '该快照行写入时间。';
