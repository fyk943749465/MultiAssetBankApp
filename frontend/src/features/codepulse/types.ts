export type Pagination = {
  page: number;
  page_size: number;
  total: number;
};

export type CPSummary = {
  proposal_total: number;
  pending_review: number;
  approved_waiting: number;
  campaign_total: number;
  fundraising: number;
  successful: number;
  failed: number;
  total_raised_wei: string;
  total_refunded_wei: string;
};

export type CPConfig = {
  contract_address: string;
  contract_configured: boolean;
  subgraph_configured: boolean;
  /** 合约 owner（来自链上 / cp_system_states），用于前端判断管理员动作是否展示 */
  owner_address?: string;
  /** 仅当 CODE_PULSE_SERVER_TX=1 且配置了 ETH_PRIVATE_KEY 时为代签地址 */
  server_tx_relayer_address?: string;
  /** 是否开放 POST /api/code-pulse/tx/submit（默认 false，由钱包签名发送） */
  code_pulse_server_tx_enabled?: boolean;
  milestone_num: number;
  min_campaign_target: string;
  min_campaign_duration: number;
  max_developers_per_campaign: number;
  max_github_url_length: number;
  stale_funds_sweep_delay: number;
  milestone_unlock_delays: number[];
};

export type CPProposal = {
  proposal_id: number;
  organizer_address: string;
  github_url: string;
  github_url_hash?: string | null;
  target_wei: string;
  duration_seconds: number;
  status: string;
  status_code: number;
  last_campaign_id?: number | null;
  current_round_count: number;
  pending_round_target_wei?: string | null;
  pending_round_duration_seconds?: number | null;
  round_review_state?: string | null;
  round_review_state_code?: number | null;
  submitted_tx_hash?: string | null;
  submitted_block_number?: number | null;
  submitted_at?: string | null;
  reviewed_at?: string | null;
  approved_at?: string | null;
  rejected_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type CPCampaign = {
  campaign_id: number;
  proposal_id: number;
  round_index: number;
  organizer_address: string;
  github_url: string;
  target_wei: string;
  deadline_at: string;
  amount_raised_wei: string;
  total_withdrawn_wei: string;
  unclaimed_refund_pool_wei: string;
  state: string;
  state_code: number;
  donor_count: number;
  developer_count: number;
  finalized_at?: string | null;
  success_at?: string | null;
  dormant_funds_swept: boolean;
  launched_tx_hash: string;
  launched_block_number: number;
  launched_at: string;
  created_at: string;
  updated_at: string;
};

export type CPContribution = {
  campaign_id: number;
  contributor_address: string;
  total_contributed_wei: string;
  refund_claimed_wei: string;
  last_donated_at?: string | null;
  last_refund_at?: string | null;
  created_at: string;
  updated_at: string;
};

/** 活动详情「贡献」接口：单笔 Donated，非 cp_contributions 聚合行。 */
export type CPCampaignDonationRow = {
  campaign_id: number;
  contributor_address: string;
  amount_wei: string;
  donated_at: string;
  tx_hash: string;
  log_index: number;
  refund_claimed_wei: string;
  data_source?: string;
};

export type CPProposalMilestone = {
  id: number;
  proposal_id: number;
  round_ordinal: number;
  milestone_index: number;
  description: string;
  percentage_raw: string;
  source_type: string;
  created_at: string;
  updated_at: string;
};

export type CPCampaignMilestone = {
  campaign_id: number;
  milestone_index: number;
  description: string;
  percentage_raw: string;
  approved: boolean;
  claimed: boolean;
  approved_at?: string | null;
  unlock_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type CPCampaignDeveloper = {
  id: number;
  campaign_id: number;
  developer_address: string;
  is_active: boolean;
  added_tx_hash?: string | null;
  removed_tx_hash?: string | null;
  added_at?: string | null;
  removed_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type CPEventLog = {
  id: number;
  chain_id: number;
  contract_address: string;
  event_name: string;
  proposal_id?: number | null;
  campaign_id?: number | null;
  wallet_address?: string | null;
  entity_key?: string | null;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_timestamp: string;
  payload: unknown;
  source: string;
  created_at: string;
};

export type CPWalletRole = {
  id?: number;
  wallet_address: string;
  role: string;
  scope_type: string;
  scope_id?: string | null;
  active: boolean;
  derived_from: string;
  source?: string;
  created_at?: string;
  updated_at?: string;
};

export type CPMilestoneClaim = {
  campaign_id: number;
  milestone_index: number;
  developer_address: string;
  claimed_amount_wei: string;
  claimed_tx_hash: string;
  claimed_at: string;
};

export type ProposalListParams = {
  status?: string;
  organizer?: string;
  review_state?: string;
  /** 仅「待进入募资」队列（与首页 Launch Queue 一致） */
  waiting_launch_queue?: boolean;
  has_pending_round?: boolean;
  page?: number;
  page_size?: number;
  sort?: "submitted_at_desc" | "submitted_at_asc";
};

export type CampaignListParams = {
  state?: string;
  proposal_id?: number;
  organizer?: string;
  developer?: string;
  contributor?: string;
  page?: number;
  page_size?: number;
  sort?: "launched_at_desc" | "deadline_at_asc" | "amount_raised_desc";
};

export type TimelineParams = {
  page?: number;
  page_size?: number;
};

export type ContributionListParams = {
  contributor?: string;
  sort?: "amount_desc" | "latest";
  page?: number;
  page_size?: number;
};

export type ProposalListResponse = {
  proposals: CPProposal[];
  pagination: Pagination;
};

export type ProposalDetailResponse = {
  proposal: CPProposal;
  milestones: CPProposalMilestone[];
  campaigns: CPCampaign[];
};

export type CampaignListResponse = {
  campaigns: CPCampaign[];
  pagination: Pagination;
};

/** 活动详情里开发者名单来自哪里（与后端 developers_source 一致） */
export type CampaignDevelopersSource = "subgraph" | "database" | "empty";

export type CampaignDetailResponse = {
  campaign: CPCampaign;
  milestones: CPCampaignMilestone[];
  developers: CPCampaignDeveloper[];
  /** 开发者名单：子图折叠事件优先；子图不可用时为 PostgreSQL cp_campaign_developers */
  developers_source?: CampaignDevelopersSource;
  donor_count: number;
  /** 活动主体 campaign 字段来源：subgraph | database */
  data_source?: string;
};

export type TimelineResponse = {
  events: CPEventLog[];
  pagination: Pagination;
  /** 活动 timeline：子图成功为 subgraph，子图失败且走 PG 为 database，无数据为 empty */
  data_source?: string;
};

export type CampaignContributionResponse = {
  contributions: CPCampaignDonationRow[];
  pagination: Pagination;
  /** 子图成功为 subgraph；子图失败且走 cp_event_log 为 database；无数据为 empty */
  data_source?: string;
};

export type WalletOverview = {
  wallet_address: string;
  roles: CPWalletRole[];
  is_admin: boolean;
  is_proposal_initiator: boolean;
  proposal_count: number;
  campaign_as_organizer_count: number;
  donation_count: number;
  developer_campaign_count: number;
  available_dashboards: string[];
  /** 子图里已有该地址作为 organizer 的提案（PG 可能尚未同步） */
  subgraph_organizer_proposals?: boolean;
};

export type InitiatorDashboard = {
  proposals_total: number;
  pending_review: CPProposal[];
  approved_waiting: CPProposal[];
  round_review_pending: CPProposal[];
  round_review_approved: CPProposal[];
  rejected: CPProposal[];
  settled_can_follow_on: CPProposal[];
  fundraising_campaigns: CPCampaign[];
  campaigns_total: number;
  /** 配置子图时：工作台只读分组以子图事件为准（`postgresql` 未返回则仍为 PG） */
  view_data_source?: "subgraph" | string;
  /** @deprecated 见 view_data_source */
  subgraph_supplement?: boolean;
};

export type ContributorDashboardEntry = CPContribution & {
  campaign_state: string;
  github_url: string;
};

export type ContributorDashboard = {
  contributions_total: number;
  total_donated_wei: string;
  all: ContributorDashboardEntry[];
  refundable: ContributorDashboardEntry[];
  fundraising: ContributorDashboardEntry[];
  successful: ContributorDashboardEntry[];
};

export type DeveloperDashboard = {
  campaigns: CPCampaign[];
  claims: CPMilestoneClaim[];
  total_claimed_wei: string;
  pending_milestones: CPCampaignMilestone[];
};

export type CodePulseAction =
  | "submit_proposal"
  | "review_proposal"
  | "submit_first_round_for_review"
  | "submit_follow_on_round_for_review"
  | "review_funding_round"
  | "launch_approved_round"
  | "donate"
  | "donate_to_platform"
  | "finalize_campaign"
  | "claim_refund"
  | "add_developer"
  | "remove_developer"
  | "approve_milestone"
  | "claim_milestone_share"
  | "sweep_stale_funds"
  | "set_proposal_initiator"
  | "withdraw_platform_funds"
  | "pause"
  | "unpause"
  | "transfer_ownership"
  | "renounce_ownership";

export type ActionCheckRequest = {
  action: CodePulseAction;
  wallet: string;
  proposal_id?: number;
  campaign_id?: number;
  milestone_index?: number;
  params?: Record<string, unknown>;
};

export type ActionCheckResponse = {
  allowed: boolean;
  required_role: string;
  current_state?: string;
  reason_code?: string;
  reason_message?: string;
  revert_error_name?: string;
  revert_error_args?: unknown;
  /** 不阻止交易，仅补充说明（如撤销 initiator 对已有 organizer 提案的影响） */
  advisory_code?: string;
  advisory_message?: string;
};

export type TxBuildRequest = {
  action: CodePulseAction;
  wallet: string;
  params?: Record<string, unknown>;
};

export type TxBuildResponse = {
  to: string;
  data: string;
  value: string;
  simulation_ok: boolean;
  /** 请求体里的 wallet（用于审计）；链上发送方见 tx_submit_signer */
  request_wallet?: string;
  /** 与 TxSubmit 一致：有 ETH_PRIVATE_KEY 时为代签地址 */
  tx_submit_signer?: string;
  /** 固定为 wallet_sign：由连接的钱包广播 */
  tx_submit_mode?: string;
  /** 后端是否额外开放服务端 submit（默认 false） */
  server_tx_submit_available?: boolean;
  chain_id?: number;
  gas_estimate?: number;
  /** 与请求 params 对照：实际打入 calldata 的众筹时长（秒，十进制字符串） */
  duration_seconds_packed?: string;
  /** 与请求 params 对照：实际打入 calldata 的目标金额（wei） */
  target_wei_packed?: string;
  revert_error_name?: string;
  revert_error_args?: unknown;
  revert_message?: string;
  advisory_code?: string;
  advisory_message?: string;
};

export type TxSubmitResponse = {
  tx_hash: string;
  action: CodePulseAction;
  /** 实际链上交易发起地址 */
  from?: string;
  tx_submit_mode?: string;
  request_wallet?: string;
};

export type CPTxAttempt = {
  id: number;
  wallet_address: string;
  role_snapshot: unknown;
  action: string;
  proposal_id?: number | null;
  campaign_id?: number | null;
  milestone_index?: number | null;
  request_payload: unknown;
  simulation_ok?: boolean | null;
  revert_error_name?: string | null;
  revert_error_args?: unknown;
  tx_hash?: string | null;
  tx_status: string;
  receipt_block_number?: number | null;
  failure_stage?: string | null;
  failure_message?: string | null;
  created_at: string;
  updated_at: string;
};

export type TxAttemptResponse = {
  attempt: CPTxAttempt;
};

export type CPPlatformFundMovement = {
  id: number;
  direction: string;
  wallet_address: string;
  amount_wei: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_timestamp: string;
  created_at: string;
};

export type CPSyncCursor = {
  sync_name: string;
  last_block_number?: number | null;
  last_block_timestamp?: string | null;
  last_event_id?: string | null;
  updated_at: string;
};

export type AdminPendingMilestone = CPCampaignMilestone & {
  github_url: string;
};

export type AdminDashboard = {
  pending_proposals: CPProposal[];
  pending_rounds: CPProposal[];
  live_campaigns: CPCampaign[];
  pending_milestones: AdminPendingMilestone[];
  initiators: CPWalletRole[];
  platform_donations: string;
  platform_withdrawals: string;
};

export type InitiatorListResponse = {
  initiators: string[];
  total: number;
  /** 列表来源：子图折叠链上 ProposalInitiatorUpdated；失败时为 PostgreSQL 角色表 */
  data_source?: "subgraph" | "database";
};

export type PlatformFundsResponse = {
  total_donations: string;
  total_withdrawals: string;
  movements: CPPlatformFundMovement[];
  pagination: Pagination;
};

/** 与后端索引器一致的 RPC 链头（可与游标对照） */
export type ChainRPCHeads = {
  latest_block: number;
  safe_block?: number;
  finalized_block?: number;
  /** 索引扫块上界（finalized / safe / latest-12 之一） */
  confirmed_tip_block: number;
  confirmed_tip_source: "finalized" | "safe" | "latest_minus_12";
};

export type SyncStatusResponse = {
  cursors: CPSyncCursor[];
  /** 子图可用且计数成功时为子图实体总数；否则为 cp_event_log 行数 */
  event_count: number;
  event_count_source?: "subgraph" | "database";
  /** 始终为 PostgreSQL cp_event_log 行数（可与 event_count 对照） */
  event_count_database?: number;
  chain_heads?: ChainRPCHeads;
};

export type AdminEventsResponse = {
  events: CPEventLog[];
  pagination: Pagination;
  /** 子图直连或回退数据库 */
  data_source?: "subgraph" | "database";
};

export type OkAddressResponse = {
  ok: boolean;
  address: string;
};
