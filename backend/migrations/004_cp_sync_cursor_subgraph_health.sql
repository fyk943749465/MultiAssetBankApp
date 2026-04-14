-- 子图同步健康字段：便于观测子图不可用、HTTP 错误及运维排障。
-- 链重组（reorg）无法仅靠增量同步自动纠正，需依赖子图重放 + 人工清库重扫，见 backend/README.md。

ALTER TABLE cp_sync_cursors
    ADD COLUMN IF NOT EXISTS last_subgraph_query_ok_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS last_subgraph_error TEXT NULL,
    ADD COLUMN IF NOT EXISTS last_subgraph_error_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS subgraph_consecutive_errors INTEGER NOT NULL DEFAULT 0;
