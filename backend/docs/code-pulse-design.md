# Code Pulse 众筹模块设计文档

## 1. 文档目的

本文档用于规划 `Code Pulse` 开源项目众筹模块在当前项目中的后端与前端设计，重点覆盖以下内容：

- PostgreSQL 应创建的表结构
- 后端应提供的 REST API
- 前端按不同角色应提供的页面与交互
- 如何结合智能合约当前事件、链上只读状态与自定义错误进行设计

本文档是实现前的设计稿，不要求与最终代码一字不差，但建议整体遵守这里的分层思路。

## 2. 已知链上事实

### 2.1 合约地址与子图

- 合约地址：`0x3100b1FD5A2180dAc11820106579545D0f1C439b`
- 网络：`Sepolia`
- 子图地址：`https://api.studio.thegraph.com/query/42035/code-pulse-advanced/version/latest`

### 2.2 当前业务流程

基于合约 ABI 与链上函数分析，完整业务流程如下：

1. 管理员通过 `setProposalInitiator` 将某个地址加入白名单，使其有权提交提案。
2. 提案发起人调用 `submitProposal` 提交提案（含 GitHub URL、目标金额、众筹时长、里程碑描述）。
3. 管理员调用 `reviewProposal` 审核提案，可通过或拒绝。
4. 提案审核通过后，提案发起人调用 `submitFirstRoundForReview` 将第一轮 funding round 提交审核。
5. 管理员调用 `reviewFundingRound` 审核 funding round，可通过或拒绝。
6. funding round 审核通过后，提案发起人调用 `launchApprovedRound` 正式启动众筹。
7. 在"提案审核通过但 funding round 尚未通过或尚未启动"的窗口期，管理员仍可通过 `reviewFundingRound(proposalId, false)` 拒绝。
8. 启动后进入众筹阶段。任何人可通过 `donate` 向 campaign 捐款。
9. 众筹截止后，任何人可调用 `finalizeCampaign` 进行结算（此函数无权限限制）。
10. 众筹失败（未达标）：捐款人通过 `claimRefund` 领取退款。
11. 众筹成功：organizer 通过 `addCampaignDeveloper` / `removeCampaignDeveloper` 管理开发者名单。
12. 管理员通过 `approveMilestone` 逐阶段审批里程碑（共 3 阶段，每阶段有解锁延迟）。
13. 开发者通过 `claimMilestoneShare` 按阶段领取份额。
14. 所有阶段完成后，如果仍有未领取退款超过宽限期，organizer 可调用 `sweepStaleFunds` 清扫沉睡资金。
15. 首轮众筹结束后，organizer 可通过 `submitFollowOnRoundForReview` 提交后续轮次（含新的目标、时长、里程碑），重复步骤 5-14。

### 2.3 角色与权限总结

基于合约 ABI 中的 custom error 和函数调用关系，各角色权限如下：

| 角色 | 权限范围 | 对应合约函数 |
|------|------|------|
| 管理员（owner） | 审核提案 | `reviewProposal` |
| | 审核 funding round | `reviewFundingRound` |
| | 管理 initiator 白名单 | `setProposalInitiator` |
| | 审批里程碑 | `approveMilestone` |
| | 暂停/恢复合约 | `pause`, `unpause` |
| | 提现平台资金 | `withdrawPlatformFunds` |
| | 转移所有权 | `transferOwnership`, `renounceOwnership` |
| 提案发起人（organizer） | 提交提案 | `submitProposal` |
| | 提交第一轮审核 | `submitFirstRoundForReview` |
| | 提交后续轮审核 | `submitFollowOnRoundForReview` |
| | 启动已审核轮次 | `launchApprovedRound` |
| | 管理开发者名单 | `addCampaignDeveloper`, `removeCampaignDeveloper` |
| | 清扫沉睡资金 | `sweepStaleFunds` |
| 捐款人 | 向 campaign 捐款 | `donate` |
| | 失败后领取退款 | `claimRefund` |
| 开发者 | 领取里程碑份额 | `claimMilestoneShare` |
| 任何人 | 结算 campaign | `finalizeCampaign` |
| | 向平台捐赠 | `donateToPlatform` |

### 2.4 子图当前已索引的事件

当前 `subgraph/code-pulse-advanced` 已覆盖以下类型的链上事件：

- `ProposalSubmitted`
- `ProposalReviewed`
- `FundingRoundSubmittedForReview`
- `FundingRoundReviewed`
- `CrowdfundingLaunched`
- `Donated`
- `CampaignFinalized`
- `RefundClaimed`
- `DeveloperAdded`
- `DeveloperRemoved`
- `MilestoneApproved`
- `MilestoneShareClaimed`
- `StaleFundsSwept`
- `ProposalInitiatorUpdated`
- `PlatformDonated`
- `PlatformFundsWithdrawn`
- `OwnershipTransferred`
- `Paused`
- `Unpaused`

这些事件足以支持：

- 时间线
- 历史记录
- 审核轨迹
- 捐款与退款记录
- 里程碑审批与领取记录

### 2.5 自定义 error 的处理结论

当前子图没有索引 revert/custom error。ABI 中定义了 44 个自定义错误（完整列表见第 26 节）。

这些错误不会出现在 The Graph 事件查询结果中。

因此设计上应明确：

- 事件历史依赖子图
- 当前实时状态依赖链上只读函数
- 操作失败原因依赖后端在发交易前做 `simulate` / `eth_call` 并解码 revert data

## 3. 总体架构建议

建议将数据源职责拆成四层：

### 3.1 链上 RPC

职责：

- 获取实时状态
- 获取 view / pure 函数结果
- 校验当前某个动作是否可以执行
- 发交易前模拟并解码错误

合约所有只读函数清单（共 27 个）：

全局常量与状态：

- `owner()` → `address`
- `paused()` → `bool`
- `proposalCount()` → `uint256`
- `campaignCount()` → `uint256`
- `platformDonationsBalance()` → `uint256`
- `MILESTONE_NUM()` → `uint8`
- `MIN_CAMPAIGN_TARGET()` → `uint256`
- `MIN_CAMPAIGN_DURATION()` → `uint256`
- `MAX_DEVELOPERS_PER_CAMPAIGN()` → `uint256`
- `MAX_GITHUB_URL_LENGTH()` → `uint256`
- `STALE_FUNDS_SWEEP_DELAY()` → `uint256`
- `milestoneUnlockDelay(mIndex)` → `uint256`（`pure` 函数，返回每阶段硬编码的解锁延迟）

提案相关：

- `proposals(proposalId)` → `(organizer, githubUrl, target, duration, status, lastCampaignId)`
- `proposalMilestones(proposalId, milestoneIndex)` → `(description, percentage)`（提案提交时的初始里程碑，百分比由合约内部计算，非前端传入）
- `pendingFundingRound(proposalId)` → `(target, duration)`
- `pendingRoundMilestones(proposalId, milestoneIndex)` → `(description, percentage)`
- `roundReviewState(proposalId)` → `enum RoundReviewState`
- `isProposalInitiator(address)` → `bool`
- `getProposalRoundCount(proposalId)` → `uint256`
- `getProposalCampaignAt(proposalId, roundIndex)` → `uint256`（返回 campaignId）

Campaign 相关：

- `campaigns(campaignId)` → `(proposalId, organizer, githubUrl, target, deadline, amountRaised, totalWithdrawn, state, finalizedAt, unclaimedRefundPool, dormantFundsSwept, successAt)`
- `contributions(campaignId, contributor)` → `uint256`（返回当前贡献余额；claimRefund 后归零）
- `milestones(campaignId, milestoneIndex)` → `(description, percentage, approved, claimed)`
- `getMilestoneUnlockTime(campaignId, milestoneIndex)` → `uint256`
- `getCampaignDeveloperCount(campaignId)` → `uint256`
- `getCampaignDeveloperAt(campaignId, index)` → `address`
- `isCampaignDeveloper(campaignId, account)` → `bool`

### 3.2 The Graph 子图

职责：

- 提供事件流与历史记录
- 提供分页友好的时间线数据
- 作为后端增量同步的来源

适合读取的内容：

- 提案提交/审核历史
- 众筹启动记录
- 捐款记录
- 退款记录
- 开发者增删记录
- 里程碑审批与领取记录
- 平台捐赠与提现记录

### 3.3 PostgreSQL

职责：

- 保存后端自己的查询型读模型
- 对链上状态和事件做聚合
- 提供高性能列表、搜索、统计、工作台数据
- 保存交易尝试与错误日志

### 3.4 后端 API

职责：

- 聚合链上、子图与数据库结果
- 做统一状态翻译
- 根据角色生成页面所需的数据
- 暴露动作预检接口与交易构造接口

## 4. PostgreSQL 表设计

建议把表分成七类：

1. 主实体表
2. 里程碑与成员表
3. 事件流水表
4. 角色与画像表
5. 交易尝试与错误表
6. 平台资金表
7. 同步状态与统计表

---

## 5. 主实体表

### 5.1 `cp_proposals`

用途：

- 保存提案当前聚合态
- 作为提案列表与提案详情的主表

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `proposal_id` | `bigint primary key` | 提案 ID |
| `organizer_address` | `text not null` | 提案发起人 |
| `github_url` | `text not null` | GitHub 仓库地址 |
| `github_url_hash` | `text null` | `keccak256(githubUrl)`，来自 `CrowdfundingLaunched` 事件，仅在首次 launch 后可用 |
| `target_wei` | `numeric(78,0) not null` | 初始提案目标金额 |
| `duration_seconds` | `bigint not null` | 初始提案众筹持续时间 |
| `status` | `text not null` | 后端翻译后的业务状态（仅基于 `ProposalStatus` 枚举 + `roundReviewState`，不混入 campaign 状态） |
| `status_code` | `int not null` | 链上原始 `ProposalStatus` 枚举值 |
| `last_campaign_id` | `bigint null` | 最近一次启动的 campaign |
| `current_round_count` | `int not null default 0` | 当前提案历史轮次总数（来自 `getProposalRoundCount`） |
| `pending_round_target_wei` | `numeric(78,0) null` | 待审核轮次目标金额 |
| `pending_round_duration_seconds` | `bigint null` | 待审核轮次持续时间 |
| `round_review_state` | `text null` | 当前待审核轮次审核状态 |
| `round_review_state_code` | `int null` | 链上原始 `RoundReviewState` 枚举值 |
| `submitted_tx_hash` | `text null` | 首次提案提交交易哈希 |
| `submitted_block_number` | `bigint null` | 提交区块 |
| `submitted_at` | `timestamptz null` | 提交时间 |
| `reviewed_at` | `timestamptz null` | 最近审核时间 |
| `approved_at` | `timestamptz null` | 最近通过时间 |
| `rejected_at` | `timestamptz null` | 最近拒绝时间 |
| `created_at` | `timestamptz not null` | 记录创建时间 |
| `updated_at` | `timestamptz not null` | 记录更新时间 |

建议索引：

- `idx_cp_proposals_status`
- `idx_cp_proposals_organizer_address`
- `idx_cp_proposals_submitted_at_desc`
- `idx_cp_proposals_last_campaign_id`

### 5.2 `cp_campaigns`

用途：

- 保存每次正式启动的众筹轮次
- 作为众筹列表、详情、进度页的主表

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `campaign_id` | `bigint primary key` | 众筹轮次 ID |
| `proposal_id` | `bigint not null` | 所属提案 |
| `round_index` | `int not null` | 第几轮众筹 |
| `organizer_address` | `text not null` | 发起人 |
| `github_url` | `text not null` | 当前轮 GitHub URL |
| `target_wei` | `numeric(78,0) not null` | 目标金额 |
| `deadline_at` | `timestamptz not null` | 截止时间 |
| `amount_raised_wei` | `numeric(78,0) not null default 0` | 已筹金额 |
| `total_withdrawn_wei` | `numeric(78,0) not null default 0` | 已累计放款金额 |
| `unclaimed_refund_pool_wei` | `numeric(78,0) not null default 0` | 未领取退款池 |
| `state` | `text not null` | 后端翻译后的 campaign 状态 |
| `state_code` | `int not null` | 链上原始 `CampaignState` 枚举值 |
| `donor_count` | `int not null default 0` | 捐款人数 |
| `developer_count` | `int not null default 0` | 当前有效开发者人数 |
| `finalized_at` | `timestamptz null` | 最终结算时间 |
| `success_at` | `timestamptz null` | 募资成功时间 |
| `dormant_funds_swept` | `boolean not null default false` | 是否已清扫沉睡资金 |
| `launched_tx_hash` | `text not null` | 启动众筹交易哈希 |
| `launched_block_number` | `bigint not null` | 启动区块 |
| `launched_at` | `timestamptz not null` | 启动时间 |
| `created_at` | `timestamptz not null` | 记录创建时间 |
| `updated_at` | `timestamptz not null` | 记录更新时间 |

建议索引：

- `idx_cp_campaigns_proposal_id`
- `idx_cp_campaigns_state`
- `idx_cp_campaigns_organizer_address`
- `idx_cp_campaigns_deadline_at`
- `idx_cp_campaigns_launched_at_desc`

### 5.3 `cp_contributions`

用途：

- 保存捐款人的累计捐款与退款状态
- 支撑"我的捐款""可退款金额""贡献榜"

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `campaign_id` | `bigint not null` | 众筹 ID |
| `contributor_address` | `text not null` | 捐款人地址 |
| `total_contributed_wei` | `numeric(78,0) not null default 0` | 累计捐款（从 `Donated` 事件累加） |
| `refund_claimed_wei` | `numeric(78,0) not null default 0` | 已领取退款（从 `RefundClaimed` 事件累加；链上 `contributions()` 在退款后归零，无法直接读取退款金额） |
| `last_donated_at` | `timestamptz null` | 最近捐款时间 |
| `last_refund_at` | `timestamptz null` | 最近退款时间 |
| `created_at` | `timestamptz not null` | 创建时间 |
| `updated_at` | `timestamptz not null` | 更新时间 |

主键建议：

- `(campaign_id, contributor_address)`

索引建议：

- `idx_cp_contributions_contributor` on `(contributor_address)`（支撑"我的捐款"跨 campaign 查询）

---

## 6. 里程碑与成员相关表

### 6.1 `cp_proposal_milestones`

用途：

- 保存提案阶段提交的里程碑定义
- 保存某轮待审核 funding round 的里程碑定义

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `bigserial primary key` | 自增主键 |
| `proposal_id` | `bigint not null` | 提案 ID |
| `round_ordinal` | `int not null` | 第几轮 |
| `milestone_index` | `int not null` | 里程碑下标 |
| `description` | `text not null` | 里程碑描述 |
| `percentage_raw` | `numeric(20,0) not null` | 合约原始比例值（来自链上 `proposalMilestones()` 或 `pendingRoundMilestones()` view 函数读取；合约内部自动计算百分比，`submitProposal` 只接收描述不接收百分比） |
| `source_type` | `text not null` | `proposal_initial` / `pending_round` |
| `created_at` | `timestamptz not null` | 创建时间 |
| `updated_at` | `timestamptz not null` | 更新时间 |

唯一约束建议：

- `(proposal_id, round_ordinal, milestone_index, source_type)`

### 6.2 `cp_campaign_milestones`

用途：

- 保存某次正式众筹轮次的里程碑快照与审批结果

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `campaign_id` | `bigint not null` | 众筹 ID |
| `milestone_index` | `int not null` | 里程碑下标 |
| `description` | `text not null` | 描述 |
| `percentage_raw` | `numeric(20,0) not null` | 原始比例 |
| `approved` | `boolean not null default false` | 是否审批通过（campaign 级标记，来自链上 `milestones()` 返回的 `approved` 字段） |
| `claimed` | `boolean not null default false` | 是否整体已结清（campaign 级标记，来自链上 `milestones()` 返回的 `claimed` 字段；与 `cp_milestone_claims` 中每位开发者的领取记录不同） |
| `approved_at` | `timestamptz null` | 审批通过时间 |
| `unlock_at` | `timestamptz null` | 解锁时间（来自 `getMilestoneUnlockTime`） |
| `created_at` | `timestamptz not null` | 创建时间 |
| `updated_at` | `timestamptz not null` | 更新时间 |

主键建议：

- `(campaign_id, milestone_index)`

索引建议：

- `idx_cp_campaign_milestones_pending_approval` on `(campaign_id) WHERE approved = false`（部分索引，加速"待审批里程碑"查询）

### 6.3 `cp_campaign_developers`

用途：

- 保存某个 campaign 下当前/历史开发者名单

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `bigserial primary key` | 自增主键（支持同一开发者被添加→移除→再添加的历史追溯） |
| `campaign_id` | `bigint not null` | 众筹 ID |
| `developer_address` | `text not null` | 开发者钱包地址 |
| `is_active` | `boolean not null default true` | 当前是否有效 |
| `added_tx_hash` | `text null` | 添加开发者交易 |
| `removed_tx_hash` | `text null` | 移除开发者交易 |
| `added_at` | `timestamptz null` | 添加时间 |
| `removed_at` | `timestamptz null` | 移除时间 |
| `created_at` | `timestamptz not null` | 创建时间 |
| `updated_at` | `timestamptz not null` | 更新时间 |

索引建议：

- `idx_cp_campaign_developers_campaign_developer` on `(campaign_id, developer_address)`
- `idx_cp_campaign_developers_developer_address`
- `idx_cp_campaign_developers_active` on `(campaign_id) WHERE is_active = true`（部分索引，加速"当前有效开发者"查询）

说明：

- 使用自增主键而非 `(campaign_id, developer_address)` 复合主键，因为合约允许同一开发者被添加→移除→再添加。
- 如果只需要当前有效名单，查询时加 `WHERE is_active = true` 即可。
- 完整的添加/移除历史也可从 `cp_event_log` 回溯。

### 6.4 `cp_milestone_claims`

用途：

- 保存开发者每个里程碑实际领取记录

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `campaign_id` | `bigint not null` | 众筹 ID |
| `milestone_index` | `int not null` | 阶段索引 |
| `developer_address` | `text not null` | 开发者地址 |
| `claimed_amount_wei` | `numeric(78,0) not null` | 领取金额 |
| `claimed_tx_hash` | `text not null` | 领取交易哈希 |
| `claimed_at` | `timestamptz not null` | 领取时间 |

主键建议：

- `(campaign_id, milestone_index, developer_address)`

说明：合约中同一 `(campaignId, milestoneIndex, developer)` 只能 claim 一次（`AlreadyClaimed` 错误保证），因此 `claimed_tx_hash` 不需要参与主键。`claimed_tx_hash` 作为普通 `NOT NULL` 列保留，用于溯源。

索引建议：

- `idx_cp_milestone_claims_developer` on `(developer_address)`
- `idx_cp_milestone_claims_campaign` on `(campaign_id)`

---

## 7. 事件流水与同步表

### 7.1 `cp_event_log`

用途：

- 保存统一事件总表
- 提供时间线、审计、回放与重建聚合表能力

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `bigserial primary key` | 自增主键 |
| `chain_id` | `bigint not null` | 链 ID |
| `contract_address` | `text not null` | 合约地址 |
| `event_name` | `text not null` | 事件名 |
| `proposal_id` | `bigint null` | 关联提案 |
| `campaign_id` | `bigint null` | 关联众筹 |
| `wallet_address` | `text null` | 关联地址 |
| `entity_key` | `text null` | 自定义聚合键 |
| `tx_hash` | `text not null` | 交易哈希 |
| `log_index` | `int not null` | 日志索引 |
| `block_number` | `bigint not null` | 区块号 |
| `block_timestamp` | `timestamptz not null` | 区块时间 |
| `payload` | `jsonb not null` | 原始事件数据 |
| `source` | `text not null default 'subgraph'` | 数据来源 |
| `created_at` | `timestamptz not null` | 入库时间 |

唯一约束建议：

- `(tx_hash, log_index)`

说明：当前仅部署在 Sepolia 单链，`(tx_hash, log_index)` 已足够唯一。若未来扩展到多链，可改为 `(chain_id, tx_hash, log_index)`。

建议索引：

- `idx_cp_event_log_event_name`
- `idx_cp_event_log_proposal_id`（部分索引，`WHERE proposal_id IS NOT NULL`）
- `idx_cp_event_log_campaign_id`（部分索引，`WHERE campaign_id IS NOT NULL`）
- `idx_cp_event_log_wallet_address`（部分索引，`WHERE wallet_address IS NOT NULL`）
- `idx_cp_event_log_block_number_desc`

### 7.2 `cp_sync_cursors`

用途：

- 保存子图同步游标
- 记录后端增量同步到 PostgreSQL 的进度

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `sync_name` | `text primary key` | 同步任务名 |
| `last_block_number` | `bigint null` | 最后同步区块 |
| `last_block_timestamp` | `timestamptz null` | 最后同步时间 |
| `last_event_id` | `text null` | 子图最后事件 ID |
| `updated_at` | `timestamptz not null` | 更新时间 |

建议至少有一个同步任务：

- `code_pulse_subgraph_events`

---

## 8. 角色与画像表

### 8.1 `cp_wallet_profiles`

用途：

- 保存钱包的展示信息
- 作为未来扩展资料页、头像、GitHub 用户名等的基础

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `wallet_address` | `text primary key` | 钱包地址 |
| `display_name` | `text null` | 显示名 |
| `github_username` | `text null` | GitHub 用户名 |
| `avatar_url` | `text null` | 头像地址 |
| `bio` | `text null` | 简介 |
| `created_at` | `timestamptz not null` | 创建时间 |
| `updated_at` | `timestamptz not null` | 更新时间 |

### 8.2 `cp_wallet_roles`

用途：

- 统一记录一个钱包的角色身份
- 支撑前端"我的工作台"与入口导航

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `wallet_address` | `text not null` | 钱包地址 |
| `role` | `text not null` | 角色 |
| `scope_type` | `text not null` | 作用域类型 |
| `scope_id` | `text null` | 作用域 ID |
| `active` | `boolean not null default true` | 是否有效 |
| `derived_from` | `text not null` | 来源 |
| `created_at` | `timestamptz not null` | 创建时间 |
| `updated_at` | `timestamptz not null` | 更新时间 |

角色建议：

- `admin`
- `proposal_initiator`
- `organizer`
- `donor`
- `developer`

作用域建议：

- `global`
- `proposal`
- `campaign`

来源建议：

- `owner_view`
- `proposal_initiator_event`
- `proposal_submitted_event`
- `donation_event`
- `developer_added_event`

---

## 9. 交易尝试与错误日志表

### 9.1 `cp_tx_attempts`

用途：

- 保存每一次链上动作尝试
- 记录模拟失败、广播失败、回执失败、自定义 error 解码结果

这是"error 设计"的关键表。

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `bigserial primary key` | 自增主键 |
| `wallet_address` | `text not null` | 发起动作的钱包 |
| `role_snapshot` | `jsonb not null` | 发起时角色快照 |
| `action` | `text not null` | 动作名 |
| `proposal_id` | `bigint null` | 关联提案 |
| `campaign_id` | `bigint null` | 关联众筹 |
| `milestone_index` | `int null` | 关联阶段 |
| `request_payload` | `jsonb not null` | 原始请求参数 |
| `simulation_ok` | `boolean null` | 模拟是否通过 |
| `revert_error_name` | `text null` | 解码后的错误名 |
| `revert_error_args` | `jsonb null` | 错误参数 |
| `tx_hash` | `text null` | 交易哈希 |
| `tx_status` | `text not null` | 当前状态 |
| `receipt_block_number` | `bigint null` | 上链区块 |
| `failure_stage` | `text null` | 失败阶段 |
| `failure_message` | `text null` | 失败说明 |
| `created_at` | `timestamptz not null` | 创建时间 |
| `updated_at` | `timestamptz not null` | 更新时间 |

`tx_status` 建议枚举：

- `simulated_failed`
- `pending_signature`
- `submitted`
- `mined_success`
- `mined_reverted`
- `dropped`

`failure_stage` 建议枚举：

- `validation`
- `simulation`
- `wallet_signature`
- `broadcast`
- `receipt`

---

## 10. 平台资金表

### 10.1 `cp_platform_fund_movements`

用途：

- 保存平台捐赠与提现明细
- 为管理员 dashboard 提供"平台余额、捐赠历史、提现历史"数据

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `bigserial primary key` | 自增主键 |
| `direction` | `text not null` | `donation` / `withdrawal` |
| `wallet_address` | `text not null` | 捐赠人或提现接收人地址 |
| `amount_wei` | `numeric(78,0) not null` | 金额 |
| `tx_hash` | `text not null` | 交易哈希 |
| `log_index` | `int not null` | 日志索引（同一交易内可能有多条日志） |
| `block_number` | `bigint not null` | 区块号 |
| `block_timestamp` | `timestamptz not null` | 区块时间 |
| `created_at` | `timestamptz not null` | 入库时间 |

唯一约束建议：

- `(tx_hash, log_index)`

索引建议：

- `idx_cp_platform_fund_movements_direction`
- `idx_cp_platform_fund_movements_wallet` on `(wallet_address)`
- `idx_cp_platform_fund_movements_block_desc` on `(block_number DESC)`

说明：

- 从 `PlatformDonated` 和 `PlatformFundsWithdrawn` 事件同步。
- 当前链上余额仍以 `platformDonationsBalance()` view 函数为准。

---

## 11. 可选统计表

### 11.1 `cp_snapshots_daily`

用途：

- 为首页 dashboard 提供每日统计快照

建议字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `snapshot_date` | `date primary key` | 日期 |
| `proposal_count` | `int not null` | 提案总数 |
| `campaign_count` | `int not null` | 众筹总数 |
| `live_campaign_count` | `int not null` | 众筹中数量 |
| `successful_campaign_count` | `int not null` | 成功数量 |
| `failed_campaign_count` | `int not null` | 失败数量 |
| `total_raised_wei` | `numeric(78,0) not null` | 累计筹资 |
| `total_refunded_wei` | `numeric(78,0) not null` | 累计退款 |
| `created_at` | `timestamptz not null` | 创建时间 |

---

## 12. 后端 API 设计

建议统一放到：

- `/api/code-pulse/*`

分为五类：

1. 公共读取接口
2. 角色工作台接口
3. 动作预检接口
4. 交易构造/提交接口
5. 管理接口

## 13. 公共读取接口

### 13.1 `GET /api/code-pulse/summary`

用途：

- 首页总览数据

返回建议：

- 提案总数
- 审核中提案数
- 已通过待启动数
- 众筹中项目数
- 众筹成功数
- 众筹失败数
- 总捐款金额
- 总退款金额
- 平台捐赠余额
- `owner`
- `paused`

### 13.2 `GET /api/code-pulse/config`

用途：

- 返回链上规则常量
- 为前端表单校验提供配置

返回建议：

- `owner`
- `paused`
- `milestone_num`
- `min_campaign_target`
- `min_campaign_duration`
- `max_developers_per_campaign`
- `max_github_url_length`
- `stale_funds_sweep_delay`
- 每个 milestone 的 unlock delay（来自 `milestoneUnlockDelay(0/1/2)`）

### 13.3 `GET /api/code-pulse/proposals`

用途：

- 提案列表

查询参数建议：

- `status`
- `organizer`
- `review_state`
- `has_pending_round`
- `page`
- `page_size`
- `sort`

返回字段建议：

- `proposal_id`
- `organizer_address`
- `github_url`
- `target_wei`
- `duration_seconds`
- `status`
- `current_round_count`
- `last_campaign_id`
- `submitted_at`
- `approved_at`
- `rejected_at`

### 13.4 `GET /api/code-pulse/proposals/:proposalId`

用途：

- 提案详情页

返回建议：

- 提案基础信息
- 当前状态
- 审核历史
- 当前待审核轮次
- 初始里程碑（来自 `proposalMilestones`）
- 历史轮次信息
- 关联 campaign 列表
- 当前钱包的可执行动作摘要

### 13.5 `GET /api/code-pulse/proposals/:proposalId/timeline`

用途：

- 提案时间线

返回内容建议：

- `ProposalSubmitted`
- `ProposalReviewed`
- `FundingRoundSubmittedForReview`
- `FundingRoundReviewed`
- `CrowdfundingLaunched`

### 13.6 `GET /api/code-pulse/campaigns`

用途：

- 众筹列表

查询参数建议：

- `state`
- `proposal_id`
- `organizer`
- `developer`
- `contributor`
- `round_index`
- `page`
- `page_size`
- `sort`

返回字段建议：

- `campaign_id`
- `proposal_id`
- `round_index`
- `github_url`
- `target_wei`
- `amount_raised_wei`
- `deadline_at`
- `state`
- `developer_count`
- `donor_count`

### 13.7 `GET /api/code-pulse/campaigns/:campaignId`

用途：

- 众筹详情页

返回建议：

- 基础信息
- 当前筹资进度
- campaign 状态
- 开发者列表
- 里程碑列表
- 是否可退款
- 是否可 finalize（注意：`finalizeCampaign` 无权限限制，任何人可在截止时间后调用）
- 是否可 sweep stale funds（注意：仅 organizer 可调用）
- 当前钱包相关数据

### 13.8 `GET /api/code-pulse/campaigns/:campaignId/timeline`

用途：

- campaign 时间线

返回内容建议：

- `CrowdfundingLaunched`
- `Donated`
- `CampaignFinalized`
- `RefundClaimed`
- `DeveloperAdded`
- `DeveloperRemoved`
- `MilestoneApproved`
- `MilestoneShareClaimed`
- `StaleFundsSwept`

### 13.9 `GET /api/code-pulse/campaigns/:campaignId/contributions`

用途：

- 查看众筹贡献榜与捐款记录

查询参数建议：

- `contributor`
- `sort=latest|amount_desc`
- `page`
- `page_size`

### 13.10 `GET /api/code-pulse/wallets/:address/overview`

用途：

- 钱包总览
- 前端登录后据此判断要显示哪些角色入口

返回建议：

- `wallet_address`
- `roles`
- `is_admin`
- `is_proposal_initiator`
- `proposal_count`
- `campaign_as_organizer_count`
- `donation_count`
- `developer_campaign_count`
- `available_dashboards`

---

## 14. 角色工作台接口

### 14.1 `GET /api/code-pulse/admin/dashboard`

管理员工作台数据：

- 待审核提案
- 已通过待提交 funding round 审核的提案
- 待审核 funding round
- 众筹中 campaign
- 待审批里程碑
- 平台捐赠余额与资金流水
- proposal initiator 白名单

### 14.2 `GET /api/code-pulse/initiators/:address/dashboard`

提案发起人/organizer 工作台数据：

- 我提交的提案
- 审核中提案
- 审核通过待提交 funding round 审核的提案
- funding round 待审核的提案
- funding round 已通过待启动的提案
- 众筹中的 campaign
- 可管理开发者名单的 campaign
- 可提交 follow-on round 的提案
- 可清扫沉睡资金的 campaign
- 被拒绝提案与最近拒绝时间

### 14.3 `GET /api/code-pulse/contributors/:address/dashboard`

捐款人工作台数据：

- 我参与的众筹
- 累计捐款金额
- 可退款项目列表
- 已退款项目列表
- 进行中项目列表
- 历史成功项目列表

### 14.4 `GET /api/code-pulse/developers/:address/dashboard`

开发者工作台数据：

- 我参与的 campaign
- 当前可领取的里程碑份额
- 待审批的里程碑
- 已领取总金额
- 领取历史

---

## 15. 动作预检接口

### 15.1 为什么要有预检接口

前端直接根据状态码自己判断按钮是否可点，会有三个问题：

1. 规则复杂，容易写错
2. 无法准确展示 custom error
3. 链上状态可能在前端渲染后发生变化

所以应设计统一的动作预检接口。

### 15.2 `POST /api/code-pulse/actions/check`

请求体建议：

```json
{
  "action": "launch_approved_round",
  "wallet": "0x...",
  "proposal_id": 12,
  "campaign_id": null,
  "milestone_index": null,
  "params": {}
}
```

返回体建议：

```json
{
  "allowed": false,
  "required_role": "organizer",
  "current_state": "round_approved_waiting_launch",
  "reason_code": "simulation_reverted",
  "reason_message": "当前轮次尚未进入允许启动的状态",
  "revert_error_name": "FundingRoundNotApprovedForLaunch",
  "revert_error_args": {}
}
```

动作枚举（对应合约所有 nonpayable/payable 写入函数）：

- `submit_proposal`
- `review_proposal`
- `submit_first_round_for_review`
- `submit_follow_on_round_for_review`
- `review_funding_round`
- `launch_approved_round`
- `donate`
- `donate_to_platform`
- `finalize_campaign`
- `claim_refund`
- `add_developer`
- `remove_developer`
- `approve_milestone`
- `claim_milestone_share`
- `sweep_stale_funds`
- `set_proposal_initiator`
- `withdraw_platform_funds`
- `pause`
- `unpause`
- `transfer_ownership`
- `renounce_ownership`

---

## 16. 交易相关接口

### 16.1 `POST /api/code-pulse/tx/build`

用途：

- 由后端构造交易请求体
- 前端使用钱包直接签名发送

返回建议：

- `to`
- `data`
- `value`
- `chain_id`
- `gas_estimate`
- `simulation_ok`
- `revert_error_name`
- `revert_error_args`

### 16.2 `POST /api/code-pulse/tx/submit`

用途：

- 如需由后端代发交易时使用

备注：

- 不建议普通用户的所有交易都通过后端代发
- 更推荐后端构造交易，前端钱包签名
- 只有管理员动作或服务账户动作才考虑后端代发

### 16.3 `GET /api/code-pulse/tx/:attemptId`

用途：

- 查询交易尝试状态

返回建议：

- `attempt_id`
- `action`
- `tx_status`
- `tx_hash`
- `simulation_ok`
- `revert_error_name`
- `failure_stage`
- `failure_message`

---

## 17. 管理接口

### 17.1 Proposal Initiator 管理

- `GET /api/code-pulse/admin/proposal-initiators`
- `POST /api/code-pulse/admin/proposal-initiators`
- `DELETE /api/code-pulse/admin/proposal-initiators/:address`

### 17.2 平台资金管理

- `GET /api/code-pulse/admin/platform-funds`
- `POST /api/code-pulse/admin/platform-funds/withdraw`

### 17.3 同步状态接口

- `GET /api/code-pulse/admin/sync-status`

返回建议：

- 子图同步游标
- 最近同步时间
- 最近同步区块
- RPC 健康状态

---

## 18. 后端状态模型建议

建议不要把链上 `uint8` 原样传给前端，而是在后端翻译为明确业务状态。

### 18.1 Proposal 状态建议

Proposal 状态应仅基于链上 `ProposalStatus` 枚举与 `roundReviewState` 组合，不混入 Campaign 的成功/失败状态（一个 Proposal 可以有多轮 Campaign，某轮失败不代表 Proposal 终态）：

- `pending_review` — 提案已提交，等待管理员审核
- `approved` — 提案审核已通过，等待发起人提交 funding round 审核
- `rejected` — 提案被拒绝
- `round_review_pending` — funding round 已提交审核，等待管理员审核
- `round_review_approved` — funding round 审核通过，等待发起人启动众筹
- `round_review_rejected` — funding round 审核被拒绝
- `active` — 存在进行中的 campaign
- `settled` — 最近一轮 campaign 已结算（成功或失败），可发起后续轮
- `completed` — 所有轮次完成

### 18.2 Campaign 状态建议

Campaign 状态基于链上 `CampaignState` 枚举，由后端补充衍生信息：

- `fundraising` — 众筹进行中
- `failed_refundable` — 众筹失败，捐款人可退款
- `successful` — 众筹成功，等待里程碑审批
- `milestone_in_progress` — 里程碑放款进行中（可附带当前阶段索引）
- `completed` — 所有里程碑已结清
- `stale_sweepable` — 存在可清扫的沉睡资金
- `closed` — 最终状态

### 18.3 动作控制建议

后端在详情接口中直接返回：

- `available_actions` — 当前钱包可以执行的动作列表
- `blocked_actions` — 当前不可执行的动作及原因
- `blocked_reason_map` — 每个被阻止动作的原因说明

这样前端就不必复制业务规则。

---

## 19. 前端页面设计

当前 `frontend` 已经预留了一个众筹页面，但还没有真正业务内容。建议将其扩展成众筹模块壳页，并继续拆分子路由。

## 20. 前端顶层路由建议

- `/crowdfunding`
- `/crowdfunding/explore`
- `/crowdfunding/proposals/:proposalId`
- `/crowdfunding/campaigns/:campaignId`
- `/crowdfunding/me`
- `/crowdfunding/admin`

## 21. 公共页面设计

### 21.1 `/crowdfunding`

首页内容建议：

- 平台介绍
- 总览统计卡片
- 正在众筹中的项目
- 最近审核通过待启动的提案
- 最近成功项目
- 最近失败可退款项目

### 21.2 `/crowdfunding/explore`

探索页内容建议：

- 提案列表与 campaign 列表切换
- 搜索 GitHub URL / proposalId / campaignId
- 状态筛选
- 排序
- 卡片形式展示项目

筛选项建议：

- 审核中
- 已通过待启动
- 众筹中
- 成功
- 失败
- 可退款

### 21.3 `/crowdfunding/proposals/:proposalId`

提案详情页建议展示：

- GitHub URL
- 发起人地址
- 初始目标金额与持续时间
- 审核状态（提案审核 + funding round 审核 双阶段）
- 初始里程碑列表（描述与百分比；百分比由合约计算，非用户输入）
- 后续轮次列表
- 时间线
- 关联 campaign 历史

### 21.4 `/crowdfunding/campaigns/:campaignId`

众筹详情页建议展示：

- 当前筹资进度
- 截止时间与倒计时
- 已筹金额 / 目标金额
- 开发者列表
- 3 阶段里程碑进度（含解锁时间）
- 捐款榜
- 历史捐款记录
- 时间线
- 当前钱包可以执行的动作

重要说明：

- `finalizeCampaign` 无权限限制，截止时间到达后任何人都可以调用。前端应在所有用户视角显示 finalize 按钮（不仅限管理员/organizer）。

---

## 22. 管理员角色页面设计

### 22.1 `/crowdfunding/admin`

管理员工作台模块建议：

- 待审核提案列表
- 待审核 funding round 列表
- 进行中 campaign 列表
- 待审批 milestone 列表
- 平台资金概览（余额 + 捐赠/提现流水）
- Proposal Initiator 白名单管理
- 系统状态（paused、owner、subgraph sync、RPC health）

管理员可执行动作（仅限 `onlyOwner` 修饰的函数）：

- 通过/拒绝 proposal（`reviewProposal`）
- 通过/拒绝 funding round（`reviewFundingRound`）
- 设置 Proposal Initiator（`setProposalInitiator`）
- 审批 milestone（`approveMilestone`）
- 提现平台资金（`withdrawPlatformFunds`）
- 暂停/恢复合约（`pause` / `unpause`）
- 转移所有权（`transferOwnership` / `renounceOwnership`）

前端交互建议：

- 每个按钮先调用 `/actions/check`
- 通过后再构造交易
- 若失败则显示 custom error 对应中文提示

---

## 23. 提案发起人/Organizer 页面设计

### 23.1 `/crowdfunding/me`

如果当前钱包具备 organizer 身份，工作台建议展示：

- 我的提案
- 审核中
- 已通过待提交 funding round
- funding round 审核中
- funding round 已通过待启动
- 众筹中
- 已拒绝
- 可提交下一轮审核

### 23.2 `/crowdfunding/me/proposals/new`

提案创建页表单建议：

- GitHub URL
- 目标金额
- 众筹持续时间
- 3 个 milestone 描述

注意：`submitProposal` 只接收 `milestoneDescs`（`string[]`），**不接收百分比**。百分比由合约内部自动计算。前端只需收集描述。

前端应做的本地校验：

- GitHub URL 非空
- URL 长度不超过链上 `MAX_GITHUB_URL_LENGTH` 限制
- 金额不低于 `MIN_CAMPAIGN_TARGET`
- duration 不低于 `MIN_CAMPAIGN_DURATION`
- milestone 数量必须等于 `MILESTONE_NUM`
- 每个 milestone 描述不能为空

### 23.3 提案详情中的 organizer 动作

- 提交提案（`submitProposal`）
- 提交第一轮 funding round 审核（`submitFirstRoundForReview`）
- 启动已审批通过轮次（`launchApprovedRound`）
- 提交后续轮 funding round 审核（`submitFollowOnRoundForReview`）
- 添加/移除 campaign 开发者（`addCampaignDeveloper` / `removeCampaignDeveloper`）
- 清扫沉睡资金（`sweepStaleFunds`）

特别提醒：

页面必须明确提示：

- "提案审核通过"不等于"可以直接启动众筹"，还需要提交 funding round 审核并等待管理员通过。
- 在审核通过但尚未启动的窗口期，管理员仍可能拒绝 funding round。

---

## 24. 捐款人页面设计

### 24.1 `/crowdfunding/me`

如果当前钱包具备 donor 身份，工作台建议展示：

- 我参与的项目
- 累计捐款金额
- 众筹中项目
- 失败可退款项目
- 已退款项目

### 24.2 捐款人详情交互

在 campaign 详情页建议展示：

- 我的累计捐款
- 当前是否可以退款
- 可退款金额
- Donate 表单
- Claim Refund 按钮

前端按钮规则建议：

- `Donate` 在众筹结束后禁用
- `Claim Refund` 只在失败且存在退款金额时启用

---

## 25. 开发者页面设计

### 25.1 `/crowdfunding/me`

如果当前钱包具备 developer 身份，工作台建议展示：

- 我参与的项目
- 当前可领取的阶段
- 待审批阶段
- 已领取金额统计
- 领取历史

### 25.2 开发者详情交互

在 campaign 详情页中建议展示：

- 我是否属于当前开发者集合
- 各阶段是否已审批
- 各阶段解锁时间
- 当前可领取金额
- Claim 按钮

前端文案建议注意：

- 即使钱包地址曾经是开发者，也不等于一定具备当前阶段领取资格
- 若出现 `NotSnapshotDeveloper`，应提示"当前地址不在该阶段快照开发者名单内"

---

## 26. 前端统一的"我的工作台"

建议 `/crowdfunding/me` 不按单一角色实现，而是按当前钱包拥有的角色动态拼接模块。

例如：

- 若是 `admin`，显示管理员卡片入口
- 若是 `proposal_initiator` / `organizer`，显示我的提案模块
- 若是 `donor`，显示我的捐款模块
- 若是 `developer`，显示我的开发模块

这样一个钱包拥有多角色时体验更自然。

---

## 27. custom error 到前端文案映射

建议后端维护一份完整的错误翻译表（覆盖合约 ABI 中全部 44 个 custom error），前端直接展示后端返回的中文消息。

### 27.1 权限与身份类

| error 名称 | 前端展示建议 |
|------|------|
| `OwnableUnauthorizedAccount` | 当前钱包不是管理员（参数 `account`：被拒绝的地址） |
| `OwnableInvalidOwner` | 无效的所有者地址（参数 `owner`：被拒绝的地址） |
| `NotProposalInitiator` | 当前钱包不在提案发起人白名单中 |
| `OnlyOrganizer` | 只有提案发起人可执行该操作 |
| `NotADeveloper` | 当前钱包不是该项目开发者 |
| `NotSnapshotDeveloper` | 当前钱包不在该阶段快照开发者名单中 |
| `InvalidAccount` | 无效的钱包地址 |

### 27.2 提案类

| error 名称 | 前端展示建议 |
|------|------|
| `InvalidProposal` | 提案不存在 |
| `ProposalNotPending` | 当前提案不是待审核状态 |
| `ProposalNotApproved` | 当前提案尚未审核通过 |
| `EmptyGithubUrl` | GitHub URL 不能为空 |
| `GithubUrlTooLong` | GitHub URL 超出长度限制 |
| `TargetBelowMinimum` | 目标金额低于最低要求 |
| `DurationBelowMinimum` | 众筹持续时间低于最低要求 |
| `IncorrectMilestoneCount` | 里程碑数量不正确（必须为 3 个） |
| `EmptyMilestoneDescription` | 第 {index} 个里程碑描述不能为空（参数 `index`：`uint8` 下标） |

### 27.3 Funding Round 类

| error 名称 | 前端展示建议 |
|------|------|
| `FirstRoundAlreadyLaunched` | 第一轮已经启动过 |
| `FollowOnRequiresPriorRound` | 必须先完成上一轮才能发起后续轮 |
| `PriorRoundNotSettled` | 上一轮众筹尚未结算 |
| `NoPendingFundingRound` | 没有待审核的 funding round |
| `FundingRoundReviewInProgress` | 当前轮次仍在审核中 |
| `FundingRoundNotApprovedForLaunch` | 当前轮次尚未审核通过，不能启动众筹 |

### 27.4 Campaign 与众筹类

| error 名称 | 前端展示建议 |
|------|------|
| `InvalidCampaign` | 众筹轮次不存在 |
| `BadState` | 当前状态不允许执行此操作 |
| `CampaignEnded` | 众筹已结束，不能继续捐款 |
| `CampaignDormant` | 众筹处于休眠状态 |
| `NotInFundraising` | 当前不在众筹阶段 |
| `NotReachedDeadline` | 未到截止时间，不能结算 |
| `AlreadyFinalized` | 该众筹已经结算过 |
| `NotFinalized` | 该众筹尚未结算 |
| `NotSuccessful` | 众筹未成功 |
| `ZeroValue` | 金额不能为 0 |
| `NoETH` | 未附带 ETH |

### 27.5 退款类

| error 名称 | 前端展示建议 |
|------|------|
| `RefundNotAvailable` | 当前项目暂不可退款 |
| `NoContribution` | 没有捐款记录，无法退款 |

### 27.6 开发者管理类

| error 名称 | 前端展示建议 |
|------|------|
| `CannotManageDevelopers` | 当前阶段不允许调整开发者名单 |
| `AlreadyDeveloper` | 该地址已经是开发者 |
| `TooManyDevelopers` | 开发者人数已达上限 |
| `NoDevelopers` | 没有开发者，无法结算成功 |

### 27.7 里程碑类

| error 名称 | 前端展示建议 |
|------|------|
| `BadMilestoneIndex` | 里程碑索引无效 |
| `AlreadyApproved` | 该里程碑已经审批通过 |
| `PreviousMilestoneNotApproved` | 前一个里程碑尚未审批通过 |
| `MilestoneLocked` | 里程碑尚未解锁 |
| `TooEarly` | 时间锁未到期 |
| `MilestoneNotApproved` | 该里程碑尚未审核通过 |
| `MilestoneSettled` | 该里程碑已经结清 |
| `AlreadyClaimed` | 该阶段份额已领取 |
| `NoSnapshot` | 缺少开发者快照，无法领取 |

### 27.8 沉睡资金类

| error 名称 | 前端展示建议 |
|------|------|
| `AlreadySwept` | 沉睡资金已经清扫过 |
| `NothingToSweep` | 没有可清扫的沉睡资金 |

### 27.9 平台资金类

| error 名称 | 前端展示建议 |
|------|------|
| `ExceedsPlatformBalance` | 提现金额超过平台余额 |
| `ZeroWithdrawAmount` | 提现金额不能为 0 |

### 27.10 ETH 转账与系统类

| error 名称 | 前端展示建议 |
|------|------|
| `TransferFailed` | ETH 转账失败 |
| `EnforcedPause` | 合约已暂停，无法执行操作 |
| `ExpectedPause` | 合约未处于暂停状态 |
| `ReentrancyGuardReentrantCall` | 系统繁忙，请稍后重试 |

---

## 28. 同步策略建议

建议后端实现一个 `code pulse sync job`，定期把子图事件增量同步到 PostgreSQL。

同步流程建议：

1. 读取 `cp_sync_cursors`
2. 从子图按时间或区块增量拉取事件
3. 写入 `cp_event_log`
4. 更新 `cp_proposals`
5. 更新 `cp_campaigns`
6. 更新 `cp_contributions`
7. 更新 `cp_campaign_developers`
8. 更新 `cp_campaign_milestones`
9. 更新 `cp_platform_fund_movements`
10. 更新角色表
11. 更新游标

注意：

- 关键状态仍以链上 view 为准
- 子图同步更适合"事件与历史"
- 详情页可以在返回前补一次链上实时状态，避免子图延迟造成误导

---

## 29. 推荐的最小可落地版本

建议按三步实施。

### 29.1 第一步：先做只读能力

先实现：

- `/summary`
- `/config`
- `/proposals`
- `/proposals/:id`
- `/campaigns`
- `/campaigns/:id`
- `/timeline`
- `/wallets/:address/overview`

这样前端就可以先把众筹页面真正做起来。

### 29.2 第二步：做动作预检

再实现：

- `/actions/check`
- revert error 解码
- `cp_tx_attempts`

这样所有按钮都可以智能判断能不能点、为什么不能点。

### 29.3 第三步：做角色工作台

最后实现：

- `/admin/dashboard`
- `/initiators/:address/dashboard`
- `/contributors/:address/dashboard`
- `/developers/:address/dashboard`

这样用户体验才完整。

---

## 30. 最终建议

本模块设计时应坚持以下原则：

- 事件历史以子图为主
- 当前状态以链上 view 为准
- 查询性能以 PostgreSQL 聚合表为主
- 操作失败原因以"后端模拟 + revert 解码"为主
- 前端不要自行复制复杂业务规则，优先依赖后端给出的 `status`、`available_actions` 和 `reason_message`

如果后续开始进入实现阶段，建议优先落地以下内容：

1. PostgreSQL 迁移脚本
2. 子图同步任务
3. `/api/code-pulse/config`
4. `/api/code-pulse/proposals`
5. `/api/code-pulse/campaigns`
6. `/api/code-pulse/actions/check`
7. 前端 `/crowdfunding` 总览页与 `/crowdfunding/me` 工作台
