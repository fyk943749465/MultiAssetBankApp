# go-chain backend

HTTP API（Gin）。默认监听地址由环境变量 **`SERVER_ADDR`** 决定，未设置时为 **`:8080`**。

下文以 **`http://127.0.0.1:8080`** 为例；若你改了端口或绑在其它主机上，请替换前缀。

## OpenAPI 与文档页

| 说明 | 地址 |
|------|------|
| OpenAPI JSON（Swagger 2） | `GET http://127.0.0.1:8080/swagger/doc.json` |
| Swagger UI | `http://127.0.0.1:8080/swagger/index.html` |
| ReDoc | `http://127.0.0.1:8080/docs` |
| Scalar | `http://127.0.0.1:8080/scalar` |

## HTTP 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/health` | 健康检查 |
| `GET` | `/api/info` | API 名称与版本 |
| `GET` | `/api/chain/status` | 链连接与 `chain_id`（需配置 `ETH_RPC_URL`） |
| `GET` | `/api/contract/counter/value` | 读 Counter 合约 `get()` |
| `POST` | `/api/contract/counter/count` | 发交易调用 `count()`（无请求体；需私钥等） |
| `GET` | `/api/bank/deposits` | 本地库中的充值记录 |
| `GET` | `/api/bank/withdrawals` | 本地库中的提现记录 |
| `GET` | `/api/bank/subgraph/deposits` | 子图充值（需 `SUBGRAPH_URL`） |
| `GET` | `/api/bank/subgraph/withdrawals` | 子图提现（需 `SUBGRAPH_URL`） |

### NFT 平台（`/api/nft/*`）

依赖 **`DATABASE_URL`**、**`ETH_RPC_URL`**（用于解析当前 `chain_id`）。可选 **`SUBGRAPH_NFT_URL`**（与本仓库 **`subgraph/nft-platform`** 部署一致时，列表类查询可走子图）。

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/nft/sync-status` | 子图是否配置、读策略说明等 |
| `GET` | `/api/nft/contracts` | 链上已登记的 NFT 相关合约（工厂 / 模板 / 市场等） |
| `GET` | `/api/nft/collections` | 合集列表：已配置子图且**本页**子图能拉到数据时优先子图，否则 PostgreSQL |
| `GET` | `/api/nft/collections/:id` | 按库内主键读合集详情（仅 PG） |
| `GET` | `/api/nft/collections/by-contract/:contract` | 按合集合约地址读合集（仅 PG） |
| `GET` | `/api/nft/collections/:id/tokens` | 某合集下已索引的 token 行（PG） |
| `GET` | `/api/nft/holdings` | 按 `owner=0x…` 查库内持有（PG） |
| `GET` | `/api/nft/listings/active` | **活跃挂单列表**：已配置子图时**始终先查子图**（含空列表）；**仅当子图 GraphQL 失败**时用 `nft_active_listings` 兜底，并返回 **`subgraph_fallback_error`**。子图字段名通过自省 + 探测解析（见下）。 |
| `GET` | `/api/nft/listings/verify-active` | **购买前校验**：`collection`、`token_id` 查询 PG 是否存在 `listing_status=active`；返回 `active` 与可选 `listing` 行（含 `seller_address`、`price_wei`） |
| `GET` | `/api/nft/market/trade-events` | **市场事件流水**（PG `nft_market_trade_events`）：上架 / 改价 / 撤单 / 成交；支持 `page`、`page_size`、`event_type`、`involves` |
| `GET` | `/api/nft/subgraph/meta` | 探测 `SUBGRAPH_NFT_URL` 是否可用 |
| `GET` | `/api/nft/subgraph/collection?address=` | 按合集合约查子图是否已有 `CollectionCreated` |

#### NFT 接口查询参数（节选）

- **`GET /api/nft/listings/active`**：`page`、`page_size`（默认 20，最大 100）。
- **`GET /api/nft/listings/verify-active`**：**必填** `collection`（`0x` 地址）、`token_id`（十进制非负整数字符串）。
- **`GET /api/nft/market/trade-events`**：`page`、`page_size`；可选 `event_type`：`ItemListed` \| `ListingPriceUpdated` \| `ListingCanceled` \| `ItemSold`；可选 `involves=0x…`（卖家或买家地址匹配）。
- **`GET /api/nft/holdings`**：**必填** `owner`；可选分页同上。

#### 子图挂单字段与「单条 vs 集合」

Graph Node 为实体 **`NFTMarketItemListed`** 生成的根查询字段名因版本/托管方而异，且可能存在单条查询（需 `id`）与集合查询（`first`/`skip`）同名相近的情况。后端对 **`listings/active` 使用的子图查询**会：

1. 对子图执行 **`__schema { queryType { fields { name } } }`**，仅在字段名符合**集合**特征时采纳（例如名称中含 **`itemlisteds`**，或 `itemlisted` + **`_collection`** 后缀），避免误选 **`nftMarketItemListed(id: …)`** 导致缺少必填参数 `id`。
2. 自省不可用时，对常见复数字段名做轻量探测。
3. 解析结果按子图 endpoint 缓存（进程内）；更换子图部署或升级后重启后端即可重新解析。

#### NFT：PostgreSQL 权威读模型（RPC 扫块）

- 迁移 **`backend/migrations/005_nft_platform.sql`** 定义 `nft_*` 表；**RPC 索引器**在配置 **`DATABASE_URL`** + **`ETH_RPC_URL`** 且能解析 `chain_id` 时启动，将工厂 / 模板 / 市场等事件写入 PG（游标名称形如 **`nft_platform_rpc_<chainId>`**，与银行索引器同表 `chain_indexer_cursors`）。
- **活跃挂单、购买校验、市场事件历史**等以 **PostgreSQL 已索引数据**为准；子图主要用于**列表展示加速**与**子图独有查询**，且**子图默认不回写 PG**。

### 银行模块查询参数（节选）

- **`GET /api/bank/deposits`**、**`GET /api/bank/withdrawals`**  
  - `limit`（可选）：默认 `50`，最大 `200`  
  - `user`（可选）：钱包地址 `0x...`，按用户过滤  

- **`GET /api/bank/subgraph/deposits`**、**`GET /api/bank/subgraph/withdrawals`**  
  - `user`（**必填**）：`0x` 前缀地址  
  - `limit`（可选）：默认 `50`，最大 `200`  

## 运行

在项目 **`backend`** 目录下：

```bash
go run ./cmd/server
```

更新 Swagger 注释后重新生成 OpenAPI：

```bash
go generate ./cmd/server/...
```

## 相关环境变量（常用）

| 变量 | 作用 |
|------|------|
| `SERVER_ADDR` | 监听地址，默认 `:8080` |
| `DATABASE_URL` | PostgreSQL（银行索引落库等） |
| `ETH_RPC_URL` | 以太坊 JSON-RPC |
| `COUNTER_CONTRACT_ADDRESS` | Counter 合约地址 |
| `ETH_PRIVATE_KEY` | 写链交易用私钥（勿提交仓库） |
| `BANK_CONTRACT_ADDRESS` | MultiAssetBank 合约（启动银行索引） |
| `SUBGRAPH_URL` | The Graph 查询 URL |
| `SUBGRAPH_API_KEY` | 子图 API Key（若托管方需要） |
| `SUBGRAPH_NFT_URL` | **NFT 平台子图**查询 URL（与本仓库 `subgraph/nft-platform` 对应；用于合集/挂单列表优先子图、元数据探测等） |
| `CODE_PULSE_ADDRESS` | Code Pulse 众筹合约地址 |
| `CODE_PULSE_INDEXER_START_BLOCK` | **RPC 扫块索引**起始块（与 Bank 类似；`0` 表示首次从当前 safe 头往前约 2000 块）。配置 `DATABASE_URL` + `ETH_RPC_URL` + 本地址后，后端会**将链上日志写入 `cp_*` 表**作为权威读模型。 |
| `SUBGRAPH_CODE_PULSE_URL` | Code Pulse 子图查询 URL（供 API/前端 GraphQL 拉取；**默认不再写入 PG**） |
| `CODE_PULSE_SUBGRAPH_SYNC` | 设为 `true` 或 `1` 时启用**子图→PostgreSQL**增量同步（与 RPC 索引双写；一般不需要） |
| `CODE_PULSE_SUBGRAPH_START_BLOCK` | 子图同步起始块（仅当 `CODE_PULSE_SUBGRAPH_SYNC` 开启；内部存 `块号-1` 作为首次 `blockNumber_gt`） |
| `CODE_PULSE_SUBGRAPH_POLL_SECONDS` | 子图同步轮询间隔（秒），默认 `25` |

## 借贷模块：`006_lending` 迁移与子图

- 迁移文件 **`backend/migrations/006_lending.sql`** 定义 **`lending_*`** 表：与 **`hardhat-tutorial/contracts/lending`** 中 Pool / HybridPriceOracle / InterestRateStrategyFactory 等事件及 **`subgraph/lending`** 中实体字段对齐，供 **Base Sepolia（`chain_id = 84532`）** 上 RPC 扫块落库。
- **扫块进度**仍使用 **`chain_indexer_cursors`**（`001` 中已建表，**无需加列**）。建议游标名含链号以便与 Sepolia 上其它索引器区分，例如 **`lending_pool_rpc_84532`**（仅扫 Pool）；若 Oracle / Factory 独立进程，可另起 **`lending_hybrid_oracle_rpc_84532`** 等**不同 `name`** 各管一段块高。
- **读策略（与 Bank / NFT 叙述一致，由应用层实现）**：
  - **列表 / 历史 / 仪表盘**：已配置子图且子图有数据时**优先 The Graph**（`subgraph/lending`）；子图不可用或无数据时用 **PostgreSQL** 兜底。
  - **存款、取款、借款、还款、清算**等关键链上事实：以 **RPC 写入的 `lending_*` 表**为权威来源；子图用于展示加速，**不应反向覆盖**已落库事件行。
- **与其它模块隔离**：借贷子图 Bearer 仅 **`SUBGRAPH_LENDING_API_KEY`** 或 **`SUBGRAPH_API_SECOND_KEY`**（不使用 **`SUBGRAPH_API_KEY`**）；借贷 JSON-RPC 为 **`LENDING_ETH_RPC_URL`** 或 **`BASE_ETH_RPC_URL`**，与银行/Code Pulse/NFT 使用的 **`ETH_RPC_URL`** 无关。连通性见 **`GET /api/lending/chain-status`**。

## Code Pulse：RPC 索引（权威 PG）与子图（可选入库）

- **PostgreSQL `cp_*` 表**：默认仅由 **RPC `eth_getLogs` 扫块**（与 `BANK_CONTRACT` 索引器同模式：已确认区块游标在 `chain_indexer_cursors`，名称含 `code_pulse_rpc_`）写入；允许相对子图有延迟，但流程上「以库为准」时应读后端 API/DB。
- **子图**：仍可配置 `SUBGRAPH_CODE_PULSE_URL` 供前端/接口快速展示；若需子图也往库里写，设置 `CODE_PULSE_SUBGRAPH_SYNC=1`（可能与 RPC 重复写入同一 `(tx_hash, log_index)`，由唯一约束去重）。

## Code Pulse 子图同步：子图不可用与区块重组（reorg）

### 子图数据「没了」或长期不可用

- 同步器**不会**在子图失败时清空 PostgreSQL；读库会停留在**最后一次成功同步**的快照，可能落后于链。
- 游标表 `cp_sync_cursors`（`sync_name = code_pulse_subgraph`）会写入：
  - **`last_subgraph_query_ok_at`**：最近一次子图 GraphQL 成功时间；
  - **`last_subgraph_error` / `last_subgraph_error_at`**：最近一次失败原因与时间；
  - **`subgraph_consecutive_errors`**：连续失败次数（每次成功拉取会清零），便于对接告警。
- 可通过 **`GET /api/code-pulse/admin/sync-status`** 查看上述字段与 `event_count`。
- **恢复**：修复 `SUBGRAPH_CODE_PULSE_URL` / 网络 / Studio 部署后，进程会自动重试；若子图从空库重建、历史事件与本地不一致，通常需要**运维重置**（见下节「重组」里的清库思路）。

### 区块作废（链重组）与本读模型

- 当前实现是**增量追加**：`cp_event_log` 以 `(tx_hash, log_index)` 去重，**不会在重组时自动删除**已写入的事件行，也不会自动回滚对 `cp_proposals` / `cp_campaigns` 等的聚合更新。
- **实务上**：规范链上状态仍以 **RPC + 合约 view** 为准；列表/统计以 PG 为准时，应接受「子图 + 本地库 = 最终一致」的延迟与偶发偏差。
- **深度重组或必须纠正读库时**（子图重放、换端点、发现脏数据）的常见做法：
  1. 停后端或暂停同步（可先去掉 `SUBGRAPH_CODE_PULSE_URL` 再启）；
  2. 在 PostgreSQL 中删除本模块相关表数据或仅删除 `cp_event_log` 及派生表（按你们运维规范），并删除游标：`DELETE FROM cp_sync_cursors WHERE sync_name = 'code_pulse_subgraph';`；
  3. 设好 **`CODE_PULSE_SUBGRAPH_START_BLOCK`**（建议从可信块或合约部署块前开始），重启后端，让同步器从子图**全量重放**（在子图已索引的前提下）。

更严格的重组处理需要额外存储**区块哈希**、按链头回滚删除事件等，与直接 RPC 扫日志的方案类似，复杂度高，未在本仓库实现。
