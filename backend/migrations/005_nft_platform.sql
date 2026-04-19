-- NFT 平台读模型（PostgreSQL）
-- 业务覆盖：NFTFactory（克隆合集）、NFTTemplate 实现/克隆合集、NFTMarketPlace（挂单与成交）。
-- 角色维度：合约创建者（平台配置/资金）、NFT 创作者（部署合集）、卖家（挂单）、买家（成交）。
-- 依赖：001_bank_ledger.sql 中的 chain_indexer_cursors（索引器游标名由 Go 侧约定，本文件不修改该表结构）。
--
-- 执行前请确认已连接目标库；可与 001～004 共存。

-- ---------------------------------------------------------------------------
-- 1. 链上地址归一表（钱包 / 合约参与者）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_accounts (
    id              BIGSERIAL PRIMARY KEY,
    chain_id        BIGINT        NOT NULL,
    address         VARCHAR(42)   NOT NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_accounts_chain_address UNIQUE (chain_id, address)
);

CREATE INDEX IF NOT EXISTS idx_nft_accounts_address
    ON nft_accounts (address);

COMMENT ON TABLE nft_accounts IS
'链上参与地址归一表：钱包或合约地址按 (chain_id, address) 唯一存储，供创作者/买卖家/合约外键引用；索引器写入前应规范为小写 0x 前缀 42 字符。';

COMMENT ON COLUMN nft_accounts.id IS '内部主键，供其它表作为 account_id 引用。';
COMMENT ON COLUMN nft_accounts.chain_id IS 'EVM chain id，例如 Sepolia 为 11155111。';
COMMENT ON COLUMN nft_accounts.address IS '以太坊地址，42 字符含 0x；建议统一小写。';
COMMENT ON COLUMN nft_accounts.created_at IS '该地址在本库首次出现时间。';

-- ---------------------------------------------------------------------------
-- 2. 已跟踪合约注册表（工厂 / 实现母体 / 市场 / 各克隆合集）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_contracts (
    id                  BIGSERIAL PRIMARY KEY,
    chain_id            BIGINT        NOT NULL,
    address             VARCHAR(42)   NOT NULL,
    contract_kind       VARCHAR(32)   NOT NULL,
    display_label       VARCHAR(128)  NULL,
    deployed_block      BIGINT        NULL,
    deployed_tx_hash    VARCHAR(66)   NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_contracts_chain_address UNIQUE (chain_id, address),
    CONSTRAINT ck_nft_contracts_kind CHECK (contract_kind IN (
        'nft_factory',
        'nft_template',
        'nft_marketplace',
        'nft_collection'
    ))
);

CREATE INDEX IF NOT EXISTS idx_nft_contracts_chain_kind
    ON nft_contracts (chain_id, contract_kind);

COMMENT ON TABLE nft_contracts IS
'平台关注的链上合约注册表：区分工厂、ERC721 逻辑实现母体、二级市场合约、以及工厂部署出的每个 NFT 合集（克隆代理）。';

COMMENT ON COLUMN nft_contracts.id IS '内部主键，供 nft_collections 等表引用。';
COMMENT ON COLUMN nft_contracts.chain_id IS 'EVM chain id。';
COMMENT ON COLUMN nft_contracts.address IS '合约部署地址；与 chain_id 组合唯一。';
COMMENT ON COLUMN nft_contracts.contract_kind IS
'nft_factory=工厂合约；nft_template=模板/逻辑实现合约；nft_marketplace=二级市场合约；nft_collection=创作者克隆出的 ERC721 合集代理。';
COMMENT ON COLUMN nft_contracts.display_label IS '运营侧可读短名，可选。';
COMMENT ON COLUMN nft_contracts.deployed_block IS '合约创建所在区块号（若已知）。';
COMMENT ON COLUMN nft_contracts.deployed_tx_hash IS '合约创建交易哈希（若已知）。';
COMMENT ON COLUMN nft_contracts.created_at IS '本表写入时间。';

-- contract_kind 取值（索引器 / 应用查询请与此一致）：nft_factory | nft_template | nft_marketplace | nft_collection
-- ---------------------------------------------------------------------------
-- Sepolia 根合约种子（NftPlatformModule 部署产物）
-- 若主网或其它链另有一套地址，可复制本段并改 chain_id / address；
-- 克隆合集合约由索引器在 CollectionCreated 时写入，勿在此硬编码。
-- deployed_tx_hash / deployed_block：合约创建交易哈希与所在区块（Sepolia）。
-- ---------------------------------------------------------------------------

INSERT INTO nft_contracts (chain_id, address, contract_kind, display_label, deployed_block, deployed_tx_hash)
VALUES
    (
        11155111,
        '0xcbbf6cd8d652289a91dc560944108ad962a69599',
        'nft_template',
        'NFTTemplate (Sepolia)',
        10683724,
        '0xf512c6ad7714c6a74e40dc95e373f58598059827e291257e27d9f0c643d82b0f'
    ),
    (
        11155111,
        '0x32ccf2565f382519d6f8ca2fe12ba7a5ac1f8c90',
        'nft_factory',
        'NFTFactory (Sepolia)',
        10683729,
        '0x7a8d7b9d86187eb74b3041064ab4395bcd664161fbaf5519e3ec23d0f6a2f81c'
    ),
    (
        11155111,
        '0x7bf717d5c1262756e80b99f7eb6c8838cea5c9f6',
        'nft_marketplace',
        'NFTMarketPlace (Sepolia)',
        10683734,
        '0xdd30ea4165bc1b95b59a58a1994eb4ff2cc931e46b688b4f34f7a10a357f64fa'
    )
ON CONFLICT (chain_id, address) DO NOTHING;


-- ---------------------------------------------------------------------------
-- 3. 合集主表（工厂 CollectionCreated → 每个克隆合集一行）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_collections (
    id                   BIGSERIAL PRIMARY KEY,
    chain_id             BIGINT        NOT NULL,
    contract_id          BIGINT        NOT NULL
        REFERENCES nft_contracts (id) ON DELETE RESTRICT,
    creator_account_id   BIGINT        NOT NULL
        REFERENCES nft_accounts (id) ON DELETE RESTRICT,
    collection_name      TEXT          NULL,
    collection_symbol    TEXT          NULL,
    base_uri             TEXT          NULL,
    deploy_salt_hex      VARCHAR(66)   NULL,
    fee_paid_wei         NUMERIC(78, 0) NULL,
    created_block_number BIGINT        NOT NULL,
    created_tx_hash      VARCHAR(66)   NOT NULL,
    created_log_index    INTEGER       NOT NULL,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_collections_chain_contract UNIQUE (chain_id, contract_id),
    CONSTRAINT ux_nft_collections_chain_tx_log UNIQUE (chain_id, created_tx_hash, created_log_index)
);

CREATE INDEX IF NOT EXISTS idx_nft_collections_creator
    ON nft_collections (creator_account_id);
CREATE INDEX IF NOT EXISTS idx_nft_collections_created_block
    ON nft_collections (created_block_number DESC);

COMMENT ON TABLE nft_collections IS
'NFT 合集（克隆代理）主表：对应工厂事件 CollectionCreated；创作者维度统计与「我的合集」列表的核心数据源。';

COMMENT ON COLUMN nft_collections.id IS '合集业务主键。';
COMMENT ON COLUMN nft_collections.chain_id IS '链 id。';
COMMENT ON COLUMN nft_collections.contract_id IS '指向 nft_contracts 且 contract_kind=nft_collection 的合约行。';
COMMENT ON COLUMN nft_collections.creator_account_id IS '链上 msg.sender 作为创建者写入工厂事件的 creator 地址所对应的 nft_accounts.id。';
COMMENT ON COLUMN nft_collections.collection_name IS 'ERC721 name（若索引器从链上或初始化参数解析）。';
COMMENT ON COLUMN nft_collections.collection_symbol IS 'ERC721 symbol（若已解析）。';
COMMENT ON COLUMN nft_collections.base_uri IS '最近一次 BaseURIUpdated 或 initialize 所见的 token 元数据前缀（可选缓存）。';
COMMENT ON COLUMN nft_collections.deploy_salt_hex IS 'CREATE2 时 salt，bytes32 以 0x 前缀 66 字符存储；CREATE 则为 NULL。';
COMMENT ON COLUMN nft_collections.fee_paid_wei IS 'CollectionCreated 中的 feePaid（创建时支付的 wei）。';
COMMENT ON COLUMN nft_collections.created_block_number IS '合集创建交易所在区块。';
COMMENT ON COLUMN nft_collections.created_tx_hash IS '合集创建交易哈希。';
COMMENT ON COLUMN nft_collections.created_log_index IS 'CollectionCreated 日志在交易内的索引。';
COMMENT ON COLUMN nft_collections.created_at IS '本行首次插入时间。';
COMMENT ON COLUMN nft_collections.updated_at IS '本行任意字段更新时刷新。';

-- ---------------------------------------------------------------------------
-- 4. Token 当前态（所有权等；由 Transfer / mint 索引维护）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_tokens (
    id                   BIGSERIAL PRIMARY KEY,
    chain_id             BIGINT        NOT NULL,
    collection_id        BIGINT        NOT NULL
        REFERENCES nft_collections (id) ON DELETE CASCADE,
    token_id             NUMERIC(78, 0) NOT NULL,
    owner_account_id     BIGINT        NOT NULL
        REFERENCES nft_accounts (id) ON DELETE RESTRICT,
    mint_tx_hash         VARCHAR(66)   NULL,
    mint_block_number    BIGINT        NULL,
    last_transfer_tx_hash VARCHAR(66)  NULL,
    last_transfer_block  BIGINT        NULL,
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_tokens_collection_token UNIQUE (collection_id, token_id)
);

CREATE INDEX IF NOT EXISTS idx_nft_tokens_owner
    ON nft_tokens (owner_account_id);
CREATE INDEX IF NOT EXISTS idx_nft_tokens_chain_collection
    ON nft_tokens (chain_id, collection_id);

COMMENT ON TABLE nft_tokens IS
'每个 (合集, tokenId) 的当前读模型：持有人、铸造交易等；由 ERC721 Transfer 与 mint 事件增量更新。';

COMMENT ON COLUMN nft_tokens.id IS '内部主键。';
COMMENT ON COLUMN nft_tokens.chain_id IS '链 id（冗余自合集，便于分片查询）。';
COMMENT ON COLUMN nft_tokens.collection_id IS '所属合集，对应 nft_collections.id。';
COMMENT ON COLUMN nft_tokens.token_id IS '链上 uint256 tokenId，使用 NUMERIC 避免超大整数精度问题。';
COMMENT ON COLUMN nft_tokens.owner_account_id IS '当前 owner 地址对应的 nft_accounts.id。';
COMMENT ON COLUMN nft_tokens.mint_tx_hash IS '首次铸造该 token 的交易哈希（若可区分）。';
COMMENT ON COLUMN nft_tokens.mint_block_number IS '首次铸造所在区块。';
COMMENT ON COLUMN nft_tokens.last_transfer_tx_hash IS '最近一次 Transfer 所在交易。';
COMMENT ON COLUMN nft_tokens.last_transfer_block IS '最近一次 Transfer 所在区块。';
COMMENT ON COLUMN nft_tokens.updated_at IS '最近一次链上状态同步时间。';

-- ---------------------------------------------------------------------------
-- 5. Transfer 事件流水（append-only，幂等）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_transfers (
    id                   BIGSERIAL PRIMARY KEY,
    chain_id             BIGINT        NOT NULL,
    collection_id        BIGINT        NOT NULL
        REFERENCES nft_collections (id) ON DELETE CASCADE,
    token_id             NUMERIC(78, 0) NOT NULL,
    from_account_id      BIGINT        NULL
        REFERENCES nft_accounts (id) ON DELETE RESTRICT,
    to_account_id        BIGINT        NOT NULL
        REFERENCES nft_accounts (id) ON DELETE RESTRICT,
    block_number         BIGINT        NOT NULL,
    block_time           TIMESTAMPTZ   NOT NULL,
    tx_hash              VARCHAR(66)   NOT NULL,
    log_index            INTEGER       NOT NULL,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_transfers_chain_tx_log UNIQUE (chain_id, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_nft_transfers_collection_token_block
    ON nft_transfers (collection_id, token_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_nft_transfers_to_account
    ON nft_transfers (to_account_id);

COMMENT ON TABLE nft_transfers IS
'ERC721 Transfer 事件追加表：用于持有人历史、审计与重扫；与 (chain_id, tx_hash, log_index) 唯一约束保证幂等。';

COMMENT ON COLUMN nft_transfers.id IS '内部主键。';
COMMENT ON COLUMN nft_transfers.chain_id IS '链 id。';
COMMENT ON COLUMN nft_transfers.collection_id IS '发生转账的合集。';
COMMENT ON COLUMN nft_transfers.token_id IS '被转移的 tokenId。';
COMMENT ON COLUMN nft_transfers.from_account_id IS '转出方；mint 时可为零地址对应账户或 NULL（由索引器约定）。';
COMMENT ON COLUMN nft_transfers.to_account_id IS '转入方。';
COMMENT ON COLUMN nft_transfers.block_number IS '日志所在区块号。';
COMMENT ON COLUMN nft_transfers.block_time IS '区块时间（来自节点或链配置）。';
COMMENT ON COLUMN nft_transfers.tx_hash IS '交易哈希。';
COMMENT ON COLUMN nft_transfers.log_index IS '日志在交易内的索引。';
COMMENT ON COLUMN nft_transfers.created_at IS '入库时间。';

-- ---------------------------------------------------------------------------
-- 6. 工厂侧非「新合集」类事件（费用、暂停、提现、退款等；append-only）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_factory_events (
    id                   BIGSERIAL PRIMARY KEY,
    chain_id             BIGINT        NOT NULL,
    factory_contract_id  BIGINT        NOT NULL
        REFERENCES nft_contracts (id) ON DELETE RESTRICT,
    event_type           VARCHAR(48)   NOT NULL,
    block_number         BIGINT        NOT NULL,
    block_time           TIMESTAMPTZ   NOT NULL,
    tx_hash              VARCHAR(66)   NOT NULL,
    log_index            INTEGER       NOT NULL,
    payload_json         JSONB         NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_factory_events_chain_tx_log UNIQUE (chain_id, tx_hash, log_index),
    CONSTRAINT ck_nft_factory_events_type CHECK (event_type IN (
        'CollectionCreated',
        'CreationFeeUpdated',
        'Paused',
        'Unpaused',
        'Withdrawal',
        'EthReceived',
        'RefundSent',
        'OwnershipTransferred'
    ))
);

CREATE INDEX IF NOT EXISTS idx_nft_factory_events_factory_block
    ON nft_factory_events (factory_contract_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_nft_factory_events_type
    ON nft_factory_events (event_type);

COMMENT ON TABLE nft_factory_events IS
'NFTFactory 合约事件流水：含 CollectionCreated（可与 nft_collections 重复存储便于审计）、费用/暂停/提现等；合约创建者（平台）运营视图的数据源之一。';

COMMENT ON COLUMN nft_factory_events.id IS '内部主键。';
COMMENT ON COLUMN nft_factory_events.chain_id IS '链 id。';
COMMENT ON COLUMN nft_factory_events.factory_contract_id IS '指向 nft_contracts 中 contract_kind=nft_factory 的行。';
COMMENT ON COLUMN nft_factory_events.event_type IS '事件名称，与 ABI 中事件名一致。';
COMMENT ON COLUMN nft_factory_events.block_number IS '日志所在区块。';
COMMENT ON COLUMN nft_factory_events.block_time IS '区块时间。';
COMMENT ON COLUMN nft_factory_events.tx_hash IS '交易哈希。';
COMMENT ON COLUMN nft_factory_events.log_index IS '日志索引。';
COMMENT ON COLUMN nft_factory_events.payload_json IS
'事件参数 JSON：如 CollectionCreated 的 collection/creator/feePaid/salt；Withdrawal 的 to/amount 等，便于扩展字段。';
COMMENT ON COLUMN nft_factory_events.created_at IS '入库时间。';

-- ---------------------------------------------------------------------------
-- 7. 市场侧「协议/运营」类事件（费率、暂停、提现等；append-only）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_marketplace_admin_events (
    id                       BIGSERIAL PRIMARY KEY,
    chain_id                 BIGINT        NOT NULL,
    marketplace_contract_id BIGINT       NOT NULL
        REFERENCES nft_contracts (id) ON DELETE RESTRICT,
    event_type               VARCHAR(48)   NOT NULL,
    block_number             BIGINT        NOT NULL,
    block_time               TIMESTAMPTZ   NOT NULL,
    tx_hash                  VARCHAR(66)   NOT NULL,
    log_index                INTEGER       NOT NULL,
    payload_json             JSONB         NOT NULL DEFAULT '{}'::jsonb,
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_marketplace_admin_events_chain_tx_log UNIQUE (chain_id, tx_hash, log_index),
    CONSTRAINT ck_nft_marketplace_admin_events_type CHECK (event_type IN (
        'PlatformFeeUpdated',
        'MaxRoyaltyBpsUpdated',
        'Paused',
        'Unpaused',
        'PlatformFeesWithdrawn',
        'UntrackedEthWithdrawn',
        'OwnershipTransferred'
    ))
);

CREATE INDEX IF NOT EXISTS idx_nft_marketplace_admin_events_contract_block
    ON nft_marketplace_admin_events (marketplace_contract_id, block_number DESC);

COMMENT ON TABLE nft_marketplace_admin_events IS
'NFTMarketPlace 上与「平台配置与资金」相关的事件流水：费率、暂停、提现等；合约创建者管理市场参数与对账的数据源。';

COMMENT ON COLUMN nft_marketplace_admin_events.id IS '内部主键。';
COMMENT ON COLUMN nft_marketplace_admin_events.chain_id IS '链 id。';
COMMENT ON COLUMN nft_marketplace_admin_events.marketplace_contract_id IS '指向 nft_contracts 中 contract_kind=nft_marketplace 的行。';
COMMENT ON COLUMN nft_marketplace_admin_events.event_type IS '事件名称，与 ABI 一致。';
COMMENT ON COLUMN nft_marketplace_admin_events.block_number IS '日志所在区块。';
COMMENT ON COLUMN nft_marketplace_admin_events.block_time IS '区块时间。';
COMMENT ON COLUMN nft_marketplace_admin_events.tx_hash IS '交易哈希。';
COMMENT ON COLUMN nft_marketplace_admin_events.log_index IS '日志索引。';
COMMENT ON COLUMN nft_marketplace_admin_events.payload_json IS '事件参数 JSON，如 oldBps/newBps、to/amount 等。';
COMMENT ON COLUMN nft_marketplace_admin_events.created_at IS '入库时间。';

-- ---------------------------------------------------------------------------
-- 8. 市场交易事件（挂单、改价、取消、成交；append-only）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_market_trade_events (
    id                       BIGSERIAL PRIMARY KEY,
    chain_id                 BIGINT        NOT NULL,
    marketplace_contract_id BIGINT       NOT NULL
        REFERENCES nft_contracts (id) ON DELETE RESTRICT,
    event_type               VARCHAR(32)   NOT NULL,
    collection_address       VARCHAR(42)   NOT NULL,
    token_id                 NUMERIC(78, 0) NOT NULL,
    seller_account_id        BIGINT        NULL
        REFERENCES nft_accounts (id) ON DELETE RESTRICT,
    buyer_account_id         BIGINT        NULL
        REFERENCES nft_accounts (id) ON DELETE RESTRICT,
    price_wei                NUMERIC(78, 0) NULL,
    old_price_wei            NUMERIC(78, 0) NULL,
    new_price_wei            NUMERIC(78, 0) NULL,
    platform_fee_wei         NUMERIC(78, 0) NULL,
    royalty_amount_wei       NUMERIC(78, 0) NULL,
    fee_bps_snapshot         NUMERIC(78, 0) NULL,
    block_number             BIGINT        NOT NULL,
    block_time               TIMESTAMPTZ   NOT NULL,
    tx_hash                  VARCHAR(66)   NOT NULL,
    log_index                INTEGER       NOT NULL,
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_market_trade_events_chain_tx_log UNIQUE (chain_id, tx_hash, log_index),
    CONSTRAINT ck_nft_market_trade_events_type CHECK (event_type IN (
        'ItemListed',
        'ListingPriceUpdated',
        'ListingCanceled',
        'ItemSold'
    ))
);

CREATE INDEX IF NOT EXISTS idx_nft_market_trade_events_collection_token
    ON nft_market_trade_events (collection_address, token_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_nft_market_trade_events_seller
    ON nft_market_trade_events (seller_account_id);
CREATE INDEX IF NOT EXISTS idx_nft_market_trade_events_buyer
    ON nft_market_trade_events (buyer_account_id) WHERE buyer_account_id IS NOT NULL;

COMMENT ON TABLE nft_market_trade_events IS
'二级市场用户行为事件：挂单、改价、撤单、成交；卖家/买家维度报表与成交明细的数据源；与子图实体可对齐。';

COMMENT ON COLUMN nft_market_trade_events.id IS '内部主键。';
COMMENT ON COLUMN nft_market_trade_events.chain_id IS '链 id。';
COMMENT ON COLUMN nft_market_trade_events.marketplace_contract_id IS '市场合约在 nft_contracts 中的 id。';
COMMENT ON COLUMN nft_market_trade_events.event_type IS 'ItemListed / ListingPriceUpdated / ListingCanceled / ItemSold。';
COMMENT ON COLUMN nft_market_trade_events.collection_address IS 'NFT 合集合约地址（事件中的 collection），便于 join 前快速过滤。';
COMMENT ON COLUMN nft_market_trade_events.token_id IS 'tokenId。';
COMMENT ON COLUMN nft_market_trade_events.seller_account_id IS '卖家账户；ItemSold 等场景可为空视解析而定。';
COMMENT ON COLUMN nft_market_trade_events.buyer_account_id IS '买家账户；仅 ItemSold 等成交类事件有值。';
COMMENT ON COLUMN nft_market_trade_events.price_wei IS '挂单或成交价格（wei），视事件类型使用。';
COMMENT ON COLUMN nft_market_trade_events.old_price_wei IS 'ListingPriceUpdated 旧价。';
COMMENT ON COLUMN nft_market_trade_events.new_price_wei IS 'ListingPriceUpdated 新价。';
COMMENT ON COLUMN nft_market_trade_events.platform_fee_wei IS 'ItemSold 平台费。';
COMMENT ON COLUMN nft_market_trade_events.royalty_amount_wei IS 'ItemSold 版税金额。';
COMMENT ON COLUMN nft_market_trade_events.fee_bps_snapshot IS 'ItemSold 时 feeBpsSnapshot。';
COMMENT ON COLUMN nft_market_trade_events.block_number IS '日志所在区块。';
COMMENT ON COLUMN nft_market_trade_events.block_time IS '区块时间。';
COMMENT ON COLUMN nft_market_trade_events.tx_hash IS '交易哈希。';
COMMENT ON COLUMN nft_market_trade_events.log_index IS '日志索引。';
COMMENT ON COLUMN nft_market_trade_events.created_at IS '入库时间。';

-- ---------------------------------------------------------------------------
-- 9. 挂单快照（含软删除；便于列表页与历史；由市场事件维护）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_active_listings (
    id                    BIGSERIAL PRIMARY KEY,
    chain_id              BIGINT        NOT NULL,
    marketplace_contract_id BIGINT      NOT NULL
        REFERENCES nft_contracts (id) ON DELETE RESTRICT,
    collection_address    VARCHAR(42)   NOT NULL,
    token_id              NUMERIC(78, 0) NOT NULL,
    seller_account_id     BIGINT        NOT NULL
        REFERENCES nft_accounts (id) ON DELETE RESTRICT,
    price_wei             NUMERIC(78, 0) NOT NULL,
    listed_block_number   BIGINT        NOT NULL,
    listed_tx_hash        VARCHAR(66)   NOT NULL,
    listing_status        VARCHAR(16)   NOT NULL DEFAULT 'active',
    closed_at             TIMESTAMPTZ   NULL,
    close_tx_hash         VARCHAR(66)   NULL,
    close_log_index       INTEGER       NULL,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ck_nft_active_listings_status CHECK (listing_status IN (
        'active',
        'cancelled',
        'sold'
    )),
    CONSTRAINT ck_nft_active_listings_active_vs_closed CHECK (
        (listing_status = 'active' AND closed_at IS NULL)
        OR (listing_status <> 'active' AND closed_at IS NOT NULL)
    )
);

-- 同一链上同一 (合集地址, tokenId) 仅允许一条「在售」记录；撤单/成交后行保留为软删，可再次挂单插入新行。
CREATE UNIQUE INDEX IF NOT EXISTS ux_nft_active_listings_one_active
    ON nft_active_listings (chain_id, collection_address, token_id)
    WHERE listing_status = 'active';

CREATE INDEX IF NOT EXISTS idx_nft_active_listings_market
    ON nft_active_listings (marketplace_contract_id);
CREATE INDEX IF NOT EXISTS idx_nft_active_listings_price_active
    ON nft_active_listings (price_wei)
    WHERE listing_status = 'active';
CREATE INDEX IF NOT EXISTS idx_nft_active_listings_collection_token_status
    ON nft_active_listings (collection_address, token_id, listing_status);

COMMENT ON TABLE nft_active_listings IS
'挂单快照表（软删除）：ItemListed 插入新行（或改价时更新同一 active 行，由索引器约定）；ListingCanceled / ItemSold 将 listing_status 置为 cancelled/sold 并填写 closed_at/close_*，不物理删除；再次上架插入新 active 行。市场列表查询应过滤 listing_status=active。';

COMMENT ON COLUMN nft_active_listings.id IS '内部主键；同一 (合集, token) 可存在多行历史（多轮上架）。';
COMMENT ON COLUMN nft_active_listings.chain_id IS '链 id。';
COMMENT ON COLUMN nft_active_listings.marketplace_contract_id IS '市场合约 id。';
COMMENT ON COLUMN nft_active_listings.collection_address IS '合集合约地址。';
COMMENT ON COLUMN nft_active_listings.token_id IS '上架的 tokenId。';
COMMENT ON COLUMN nft_active_listings.seller_account_id IS '该条快照对应的挂单人。';
COMMENT ON COLUMN nft_active_listings.price_wei IS '该条快照对应的挂单价（wei）；改价时更新。';
COMMENT ON COLUMN nft_active_listings.listed_block_number IS '当前价格或首次上架生效所依据的区块。';
COMMENT ON COLUMN nft_active_listings.listed_tx_hash IS '当前价格或首次上架生效所依据的交易哈希。';
COMMENT ON COLUMN nft_active_listings.listing_status IS
'active=在售；cancelled=用户撤单软删；sold=成交软删。仅 active 参与唯一性约束与卖场列表。';
COMMENT ON COLUMN nft_active_listings.closed_at IS '转为非 active 的时间（撤单/成交写入时填充）；active 时为 NULL。';
COMMENT ON COLUMN nft_active_listings.close_tx_hash IS '结束挂单链上交易哈希（ListingCanceled / ItemSold）。';
COMMENT ON COLUMN nft_active_listings.close_log_index IS '结束挂单事件在该交易内的 log_index。';
COMMENT ON COLUMN nft_active_listings.created_at IS '本行插入时间（该轮快照首次写入库）。';
COMMENT ON COLUMN nft_active_listings.updated_at IS '本行任意字段最后更新时间（含改价、软删状态变更）。';

-- ---------------------------------------------------------------------------
-- 10. 合集链上元数据运维事件（BaseURI、版税、所有权等；可选索引）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nft_collection_events (
    id                   BIGSERIAL PRIMARY KEY,
    chain_id             BIGINT        NOT NULL,
    collection_id        BIGINT        NOT NULL
        REFERENCES nft_collections (id) ON DELETE CASCADE,
    event_type           VARCHAR(48)   NOT NULL,
    block_number         BIGINT        NOT NULL,
    block_time           TIMESTAMPTZ   NOT NULL,
    tx_hash              VARCHAR(66)   NOT NULL,
    log_index            INTEGER       NOT NULL,
    payload_json         JSONB         NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ux_nft_collection_events_chain_tx_log UNIQUE (chain_id, tx_hash, log_index),
    CONSTRAINT ck_nft_collection_events_type CHECK (event_type IN (
        'BaseURIUpdated',
        'DefaultRoyaltyUpdated',
        'OwnershipTransferred',
        'Initialized',
        'Approval',
        'ApprovalForAll'
    ))
);

CREATE INDEX IF NOT EXISTS idx_nft_collection_events_collection_block
    ON nft_collection_events (collection_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_nft_collection_events_type
    ON nft_collection_events (event_type);

COMMENT ON TABLE nft_collection_events IS
'各克隆合集（ERC721 代理）上除「已在 nft_transfers 专表处理」外的元数据/运维类事件可选存证；若不希望与 nft_transfers 重复存储 Transfer，索引器可对 Transfer 仅写 nft_transfers 而不写本表。';

COMMENT ON COLUMN nft_collection_events.id IS '内部主键。';
COMMENT ON COLUMN nft_collection_events.chain_id IS '链 id。';
COMMENT ON COLUMN nft_collection_events.collection_id IS 'nft_collections.id。';
COMMENT ON COLUMN nft_collection_events.event_type IS '克隆合约上发出的事件名。';
COMMENT ON COLUMN nft_collection_events.block_number IS '区块号。';
COMMENT ON COLUMN nft_collection_events.block_time IS '区块时间。';
COMMENT ON COLUMN nft_collection_events.tx_hash IS '交易哈希。';
COMMENT ON COLUMN nft_collection_events.log_index IS '日志索引。';
COMMENT ON COLUMN nft_collection_events.payload_json IS '事件参数 JSON。';
COMMENT ON COLUMN nft_collection_events.created_at IS '入库时间。';


