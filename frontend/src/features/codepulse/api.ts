import type {
  ActionCheckRequest,
  ActionCheckResponse,
  AdminDashboard,
  CPConfig,
  CPSummary,
  InitiatorListResponse,
  OkAddressResponse,
  PlatformFundsResponse,
  CampaignContributionResponse,
  CampaignDetailResponse,
  CampaignListParams,
  CampaignListResponse,
  ContributionListParams,
  ContributorDashboard,
  DeveloperDashboard,
  InitiatorDashboard,
  ProposalDetailResponse,
  ProposalListParams,
  ProposalListResponse,
  SyncStatusResponse,
  AdminEventsResponse,
  TxAttemptResponse,
  TxBuildRequest,
  TxBuildResponse,
  TxSubmitResponse,
  TimelineParams,
  TimelineResponse,
  WalletOverview,
} from "./types";

const base = import.meta.env.DEV ? "" : (import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:8080");

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${base}${path}`);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text || path}`);
  }
  return res.json() as Promise<T>;
}

async function sendJSON<T>(path: string, method: "POST" | "DELETE", body?: unknown): Promise<T> {
  const res = await fetch(`${base}${path}`, {
    method,
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text || path}`);
  }
  return res.json() as Promise<T>;
}

function buildQuery(
  params: Record<string, string | number | boolean | undefined | null>,
): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") continue;
    query.set(key, String(value));
  }
  const text = query.toString();
  return text ? `?${text}` : "";
}

export function fetchCodePulseSummary() {
  return getJSON<CPSummary>("/api/code-pulse/summary");
}

export function fetchCodePulseConfig() {
  return getJSON<CPConfig>("/api/code-pulse/config");
}

export function fetchCodePulseProposals(params: ProposalListParams = {}) {
  return getJSON<ProposalListResponse>(
    `/api/code-pulse/proposals${buildQuery({
      status: params.status,
      organizer: params.organizer,
      review_state: params.review_state,
      waiting_launch_queue: params.waiting_launch_queue,
      has_pending_round: params.has_pending_round,
      page: params.page,
      page_size: params.page_size,
      sort: params.sort,
    })}`,
  );
}

export function fetchCodePulseProposalDetail(proposalId: string | number) {
  return getJSON<ProposalDetailResponse>(`/api/code-pulse/proposals/${proposalId}`);
}

export function fetchCodePulseProposalTimeline(
  proposalId: string | number,
  params: TimelineParams = {},
) {
  return getJSON<TimelineResponse>(
    `/api/code-pulse/proposals/${proposalId}/timeline${buildQuery({
      page: params.page,
      page_size: params.page_size,
    })}`,
  );
}

export function fetchCodePulseCampaigns(params: CampaignListParams = {}) {
  return getJSON<CampaignListResponse>(
    `/api/code-pulse/campaigns${buildQuery({
      state: params.state,
      proposal_id: params.proposal_id,
      organizer: params.organizer,
      developer: params.developer,
      contributor: params.contributor,
      page: params.page,
      page_size: params.page_size,
      sort: params.sort,
    })}`,
  );
}

function assertCampaignId(campaignId: string | number): string {
  const s = String(campaignId).trim();
  if (!s || s === "undefined" || !/^\d+$/.test(s)) {
    throw new Error("无效的活动编号（campaign_id 须为正整数）");
  }
  return s;
}

export function fetchCodePulseCampaignDetail(campaignId: string | number) {
  const id = assertCampaignId(campaignId);
  return getJSON<CampaignDetailResponse>(`/api/code-pulse/campaigns/${id}`);
}

export function fetchCodePulseCampaignTimeline(
  campaignId: string | number,
  params: TimelineParams = {},
) {
  const id = assertCampaignId(campaignId);
  return getJSON<TimelineResponse>(
    `/api/code-pulse/campaigns/${id}/timeline${buildQuery({
      page: params.page,
      page_size: params.page_size,
    })}`,
  );
}

export function fetchCodePulseCampaignContributions(
  campaignId: string | number,
  params: ContributionListParams = {},
) {
  const id = assertCampaignId(campaignId);
  return getJSON<CampaignContributionResponse>(
    `/api/code-pulse/campaigns/${id}/contributions${buildQuery({
      contributor: params.contributor,
      sort: params.sort,
      page: params.page,
      page_size: params.page_size,
    })}`,
  );
}

export function fetchCodePulseWalletOverview(address: string) {
  return getJSON<WalletOverview>(`/api/code-pulse/wallets/${address}/overview`);
}

export function fetchCodePulseInitiatorDashboard(address: string) {
  return getJSON<InitiatorDashboard>(`/api/code-pulse/initiators/${address}/dashboard`);
}

export function fetchCodePulseContributorDashboard(address: string) {
  return getJSON<ContributorDashboard>(`/api/code-pulse/contributors/${address}/dashboard`);
}

export function fetchCodePulseDeveloperDashboard(address: string) {
  return getJSON<DeveloperDashboard>(`/api/code-pulse/developers/${address}/dashboard`);
}

export function checkCodePulseAction(body: ActionCheckRequest) {
  return sendJSON<ActionCheckResponse>("/api/code-pulse/actions/check", "POST", body);
}

export function buildCodePulseTx(body: TxBuildRequest) {
  return sendJSON<TxBuildResponse>("/api/code-pulse/tx/build", "POST", body);
}

export function submitCodePulseTx(body: TxBuildRequest) {
  return sendJSON<TxSubmitResponse>("/api/code-pulse/tx/submit", "POST", body);
}

export function fetchCodePulseTxAttempt(attemptId: string | number) {
  return getJSON<TxAttemptResponse>(`/api/code-pulse/tx/${attemptId}`);
}

export function fetchCodePulseAdminDashboard() {
  return getJSON<AdminDashboard>("/api/code-pulse/admin/dashboard");
}

export function fetchCodePulseInitiators() {
  return getJSON<InitiatorListResponse>("/api/code-pulse/admin/proposal-initiators");
}

export function addCodePulseInitiator(address: string) {
  return sendJSON<OkAddressResponse>("/api/code-pulse/admin/proposal-initiators", "POST", { address });
}

export function removeCodePulseInitiator(address: string) {
  return sendJSON<OkAddressResponse>(`/api/code-pulse/admin/proposal-initiators/${address}`, "DELETE");
}

export function fetchCodePulsePlatformFunds(page = 1, pageSize = 20) {
  return getJSON<PlatformFundsResponse>(
    `/api/code-pulse/admin/platform-funds${buildQuery({ page, page_size: pageSize })}`,
  );
}

export function fetchCodePulseSyncStatus() {
  return getJSON<SyncStatusResponse>("/api/code-pulse/admin/sync-status");
}

export type AdminEventsParams = {
  page?: number;
  page_size?: number;
  event_name?: string;
  proposal_id?: number;
  campaign_id?: number;
};

/** 链上事件流水（公开 GET，优先子图；与 /admin/events 同源） */
export function fetchCodePulseEventLog(params: AdminEventsParams = {}) {
  return getJSON<AdminEventsResponse>(
    `/api/code-pulse/events${buildQuery({
      page: params.page,
      page_size: params.page_size,
      event_name: params.event_name,
      proposal_id: params.proposal_id,
      campaign_id: params.campaign_id,
    })}`,
  );
}

/** @deprecated 请使用 fetchCodePulseEventLog；仍指向同一后端路径族 */
export const fetchCodePulseAdminEvents = fetchCodePulseEventLog;
