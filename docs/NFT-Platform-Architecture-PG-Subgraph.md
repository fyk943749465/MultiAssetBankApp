# NFT 平台架构说明：PostgreSQL 权威、Go 扫块与子图展示

> **适用范围**：本仓库 `NFTFactory.sol`、`NFTTemplate.sol`（克隆）、`NFTMarketPlace.sol` 三合约联动的「部署合集 → 铸造 → 市场挂单与成交」流程。  
> **本文目标**：约定 **链上事实以 PostgreSQL 为准**、**Go 后端扫已确认块** 落库、**子图服务读体验与实时性** 的分工；供前后端、索引与运维统一对齐。

---

## 1. 设计前提（必读）

### 1.1 单一事实来源（Source of Truth）

| 层级 | 作用 | 可信度 |
|------|------|--------|
| **链上** | 最终裁决 | 绝对正确，但直连成本高、不利于复杂查询。 |
| **PostgreSQL** | **业务与对账的权威数据源** | 由本系统 **Go 扫块程序** 仅处理 **已 finalize / safe** 高度的区块写入；**所有需要入账、对账、权限判断、与资金相关的链上结论，均以 PG 为准**。 |
| **Subgraph（The Graph 等）** | **列表与详情的主要读取路径** | 索引快、查询方便，但 **可能因链重组（reorg）短暂错误或与 PG 不一致**；**展示优先走子图，关键字段或与资金相关展示须能与 PG 对齐或降级**。 |

### 1.2 扫块策略

- Go 服务只消费 **`finalized` 块**（PoS 以太坊）或各链文档约定的 **`safe` 块**（若部署在 L2，以该链「不可回滚」的确认块为准）。  
- **不得**将未充分确认的块内交易直接写入「不可回滚」的业务表；若需「待确认」体验，使用单独状态（见 §6）。

### 1.3 重组（Reorg）

- 在确认深度之前的块可能被替换；子图索引器若先处理了被丢弃的块，会出现 **错误事件或错误状态**。  
- **PG 侧**：扫块程序应按链参数保留 **重组窗口**（例如回溯 N 个块重新执行同一高度范围的事件幂等写入），或对 `(block_hash, log_index)` 做校验，发现父哈希变化时 **回滚该分叉上未 finalize 的写入**（实现细节由工程选型：状态表 + 链重组检测）。  
- **原则**：**用户余额、订单是否成交、累计平台费等，只以 PG 中基于 finalized/safe 的数据为准**；子图仅作展示加速。

---

## 2. 角色与链上职责（摘要）

| 角色 | 说明 |
|------|------|
| **合约部署者（平台）** | 部署 `NFTTemplate` implementation、`NFTFactory`、`NFTMarketPlace`；配置 `creationFee`、市场费率与暂停、工厂/市场提现等。 |
| **NFT 创作者** | 调用工厂 `deploy*` 创建克隆合集；作为 `owner` 执行 `mint`、`setBaseURI`、`setDefaultRoyaltyFee`；对市场授权并挂单。 |
| **卖家** | 持有 NFT 并对市场授权；`listFormatItem`、`updateListingPrice`、`cancelListing`。 |
| **买家** | `buyItem` 支付 ETH 成交。 |

---

## 3. 合约事件 → 落库 / 子图（索引清单）

以下事件建议 **子图全部订阅**；**Go 扫块落 PG 时至少覆盖「业务状态变化」子集**（可与子图字段对齐，便于对账）。

### 3.1 `NFTFactory`

| 事件 | 含义 | PG 建议 | 子图 |
|------|------|---------|------|
| `CollectionCreated(collection, creator, feePaid, salt)` | 新合集克隆成功 | 插入 `collections`；`salt=0` 表示 CREATE | ✓ |
| `RefundSent(to, amount)` | 创建费多付退回 | 记流水/审计 | ✓ |
| `CreationFeeUpdated(oldFee, newFee)` | 创建费变更 | 配置历史表 | 可选 |
| `Withdrawal(to, amount)` | 工厂提现 | 资金流 | 可选 |
| `EthReceived(from, amount)` | 裸转入工厂 | 异常/误转审计 | 可选 |
| `Paused` / `Unpaused`（OZ） | 暂停部署 | 平台状态 | ✓ |

### 3.2 `NFTTemplate`（每个合集地址）

| 事件 / 标准日志 | 含义 | PG 建议 | 子图 |
|------------------|------|---------|------|
| `Transfer(from, to, tokenId)` | 铸造/转移 | 维护 `tokens.owner`、铸造记录 | ✓ |
| `DefaultRoyaltyUpdated(receiver, feeNumerator)` | 默认版税 | 更新合集版税展示 | ✓ |
| `OwnershipTransferred`（OZ） | 合集 owner 变更 | 更新 `collections.owner` | ✓ |
| `BaseURIUpdated(newBaseURI)` | 元数据前缀 | 更新 `collections.base_uri` | ✓ |

### 3.3 `NFTMarketPlace`

| 事件 | 含义 | PG 建议 | 子图 |
|------|------|---------|------|
| `ItemListed` | 上架 | `listings` 插入/更新为 active | ✓ |
| `ListingPriceUpdated` | 改价 | 更新 `listings.price` | ✓ |
| `ListingCanceled` | 撤单 | `listings.active=false` | ✓ |
| `ItemSold`（含 `platformFee`, `royaltyAmount`, `feeBpsSnapshot`） | 成交 | `sales` 插入；`listings` 关闭；累计费用于报表 | ✓ |
| `PlatformFeeUpdated` / `MaxRoyaltyBpsUpdated` | 费率 | 平台配置版本 | ✓ |
| `PlatformFeesWithdrawn` / `UntrackedEthWithdrawn` | 提现 | 金库流水 | ✓ |
| `Paused` / `Unpaused` | 市场暂停 | 全站交易开关 | ✓ |

---

## 4. PostgreSQL 数据模型（提纲）

以下为 **逻辑表** 建议；具体字段类型、索引与分区由实现阶段细化。

### 4.1 链与同步元数据

- **`chain_sync_state`**：`chain_id`、当前已处理 **safe/finalized** 块高、`last_block_hash`、更新时间。  
- **`raw_logs`（可选）**：原始 `(tx_hash, log_index, block_number, block_hash, address, topics, data)` 便于重放与审计。

### 4.2 业务主数据（以 PG 为准）

- **`collections`**：`address`、`creator`、`factory_tx`、`created_at_block`、`salt`、`name/symbol`（若链上无事件带 name，可由后端在部署后补录或从链下注册表写入）。  
- **`tokens`**：`collection`、`token_id`、`owner`、`mint_tx`、最后 `transfer` 块高。  
- **`listings`**：`collection`、`token_id`、`seller`、`price`、`active`、`listed_block`、`cancel_tx` / `sale_tx` 等。  
- **`sales`**：关联 `listing` 或 `(collection, token_id)`、`buyer`、`price`、`platform_fee`、`royalty_amount`、`fee_bps_snapshot`、`tx_hash`、`block_number`。  
- **`platform_fee_ledger`**（可选）：与链上 `totalPlatformFees` 累计逻辑对账用。  
- **`config_snapshots`**：工厂 `creationFee`、市场 `platformFeeBps` / `maxRoyaltyBps` 随块高版本化，便于历史成交解释。

### 4.3 幂等与重组

- 所有由事件驱动的写入应以 **`(chain_id, tx_hash, log_index)`** 或等价唯一键 **幂等插入**（`ON CONFLICT DO UPDATE`）。  
- 检测到重组：删除或标记 **已废弃 `block_hash`** 上的行，再重放新区块同一高度的日志（策略需在实现文档中写死确认深度与回溯长度）。

---

## 5. Go 扫块服务设计要点

1. **输入**：RPC/WebSocket 拉取 `safe` / `finalized` 块；解析三合约 ABI 相关 `Log`。  
2. **输出**：事务内写入 PG；更新 `chain_sync_state`。  
3. **顺序**：同块内按 `log_index` 排序处理，避免同一交易中多事件顺序错乱。  
4. **与前端 API**：**读接口** 对「是否已成交、是否仍在上架、用户资产列表」等 **必须以 PG 查询结果返回**；子图数据不直接作为入账依据。  
5. **延迟**：finalized 相对链头有延迟；产品需在 UI 标明 **「约 X 个确认后入账」** 或「交易已上链，确认中」。

---

## 6. 子图与 PG 的协同策略（展示优先子图）

### 6.1 读路径

| 场景 | 建议 |
|------|------|
| **市场列表、合集详情、元数据链接** | **优先子图**（低延迟、复杂过滤）。 |
| **购买按钮是否可点、余额类、已售确认、我的订单状态** | **以 PG 或后端读 PG 为准**；可与子图对比，不一致时 **以 PG 覆盖 UI** 或提示「数据同步中」。 |
| **管理后台 / 财务报表** | **仅 PG**。 |

### 6.2 子图错误时的表现

- 短期：子图显示「已售」而 PG 仍为「在售」→ **禁止直接依据子图执行二次写链**；应提示刷新或等待 PG 追上。  
- 长期：定时任务 **子图 vs PG** 抽样对账（同一 `tx_hash` 的 `ItemSold`），差异告警。

### 6.3 可选：「待确认」态

- 若产品需在 finalize 前展示「可能成交」：可增加 **`pending_sales`**（来自 unsafe 块或 mempool 合作方），**明确标注非最终**；finalize 后由扫块写入 `sales` 并关闭 pending。**不得**与 PG 主表混用同一「已成交」语义。

---

## 7. 前端设计要点（与数据层对齐）

1. **首屏列表**：子图拉取；轮询或订阅后 **用后端接口（读 PG）刷新关键状态**（例如「仍可买」）。  
2. **交易提交成功回执**：提示「等待区块确认」；确认数达标后再调用 **PG 权威接口** 刷新「我的 NFT / 我的挂单」。  
3. **错误文案**：与市场合约自定义 `error` 对齐（如 `SellerNotApprovedForSale`、`ListingSellerNotOwner`），与子图无关。  
4. **工厂暂停 / 市场暂停**：子图或轻量 `eth_call` 读 `paused()`；**是否允许继续点按钮** 可与 PG 中最新 `Paused` 事件一致化。

---

## 8. 后端（Go + PG 以外组件）分工

| 组件 | 职责 |
|------|------|
| **扫块写入服务** | 唯一写入业务主路径（PG）；处理重组。 |
| **HTTP API** | 读 PG；必要时读链上 view 做校验；**不写子图**。 |
| **子图** | 索引全量事件；供前端与开放查询；**不参与资金记账**。 |
| **定时对账** | `ItemSold` 累计与链上市场余额/提现事件交叉校验（辅助发现漏块）。 |

---

## 9. 与既有需求文档的关系

- 链上产品规则仍以 `NFT-Marketplace-Requirements.md` 为蓝图时可对照；**本文侧重「链下数据栈」**：**PG + Go finalized 扫块为权威，子图优先展示**。  
- 若后续合约变更事件签名，需同步更新：**子图 schema、Go ABI 解码、PG 迁移** 三处。

---

## 10. 修订记录

| 日期 | 摘要 |
|------|------|
| 2026-04-18 | 初版：PG 权威、Go 扫 safe/finalize、子图优先展示与 reorg 说明、事件与表提纲。 |
