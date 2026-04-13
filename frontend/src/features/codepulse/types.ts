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
  id: number;
  wallet_address: string;
  role: string;
  scope_type: string;
  scope_id?: string | null;
  active: boolean;
  derived_from: string;
  created_at: string;
  updated_at: string;
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

export type CampaignDetailResponse = {
  campaign: CPCampaign;
  milestones: CPCampaignMilestone[];
  developers: CPCampaignDeveloper[];
  donor_count: number;
};

export type TimelineResponse = {
  events: CPEventLog[];
  pagination: Pagination;
};

export type CampaignContributionResponse = {
  contributions: CPContribution[];
  pagination: Pagination;
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
  chain_id?: number;
  gas_estimate?: number;
  revert_error_name?: string;
  revert_error_args?: unknown;
  revert_message?: string;
};

export type TxSubmitResponse = {
  tx_hash: string;
  action: CodePulseAction;
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
};

export type PlatformFundsResponse = {
  total_donations: string;
  total_withdrawals: string;
  movements: CPPlatformFundMovement[];
  pagination: Pagination;
};

export type SyncStatusResponse = {
  cursors: CPSyncCursor[];
  event_count: number;
};

export type OkAddressResponse = {
  ok: boolean;
  address: string;
};
