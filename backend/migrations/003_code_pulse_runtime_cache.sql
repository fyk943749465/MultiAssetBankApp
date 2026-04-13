-- Code Pulse 链上状态缓存 / 角色同步元数据

ALTER TABLE cp_wallet_roles
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'db',
    ADD COLUMN IF NOT EXISTS source_block_number BIGINT NULL,
    ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS cp_system_states (
    contract_address    VARCHAR(42) PRIMARY KEY,
    owner_address       VARCHAR(42) NOT NULL,
    paused              BOOLEAN     NOT NULL DEFAULT false,
    source              TEXT        NOT NULL DEFAULT 'chain',
    source_block_number BIGINT      NULL,
    synced_at           TIMESTAMPTZ NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cp_system_states_synced_at
    ON cp_system_states (synced_at DESC NULLS LAST);
