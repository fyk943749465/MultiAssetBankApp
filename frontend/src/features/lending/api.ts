const base = import.meta.env.DEV ? "" : (import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:8080");

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${base}${path}`);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text || path}`);
  }
  return res.json() as Promise<T>;
}

export type LendingChainStatus =
  | { configured: false; scope: string; message: string }
  | { configured: true; scope: string; chain_id: number; note?: string };

export type LendingSyncStatus = {
  chain_id: number;
  database_configured?: boolean;
  read_policy: string;
  lending_subgraph_configured: boolean;
  lending_subgraph_persists_to_pg: boolean;
  lending_rpc_configured: boolean;
  lending_subgraph_api_key_source?: string;
  lending_subgraph_bearer_present?: boolean;
  note: string;
  configuration_hints?: Record<string, string>;
};

export type LendingContractRow = {
  id: number;
  chain_id: number;
  address: string;
  contract_kind: string;
  display_label?: string | null;
  deployed_block?: number | null;
  deployed_tx_hash?: string | null;
  created_at: string;
};

export type LendingContractsResponse = {
  chain_id: number;
  data_source: string;
  contracts: LendingContractRow[];
};

export type LendingSupplyPg = {
  id: number;
  chain_id: number;
  pool_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  asset_address: string;
  user_address: string;
  amount_raw: string;
  created_at: string;
};

/** 子图 supplies 实体（字段与 The Graph JSON 一致） */
export type LendingSupplySubgraph = {
  id: string;
  asset: string;
  user: string;
  amount: string;
  blockNumber: string;
  blockTimestamp: string;
  transactionHash: string;
};

export type LendingSuppliesResponse = {
  chain_id: number;
  data_source: "subgraph" | "database";
  page: number;
  page_size: number;
  total: number;
  total_note?: string;
  subgraph_fallback_reason?: string;
  supplies: LendingSupplyPg[] | LendingSupplySubgraph[];
};

export type LendingListQuery = {
  chain_id?: number;
  pool_address?: string;
  user_address?: string;
  page?: number;
  page_size?: number;
};

/** 007 扩展 PG 列表 API 的查询参数（在 LendingListQuery 之上） */
export type LendingPgExtendedQuery = LendingListQuery & {
  asset_address?: string;
  oracle_address?: string;
  verifier_address?: string;
  token_address?: string;
  strategy_address?: string;
  to_address?: string;
  from_address?: string;
};

function toSearchParams(q: LendingListQuery): string {
  const p = new URLSearchParams();
  if (q.chain_id != null) p.set("chain_id", String(q.chain_id));
  if (q.pool_address) p.set("pool_address", q.pool_address);
  if (q.user_address) p.set("user_address", q.user_address);
  if (q.page != null) p.set("page", String(q.page));
  if (q.page_size != null) p.set("page_size", String(q.page_size));
  const s = p.toString();
  return s ? `?${s}` : "";
}

function toPgExtendedSearchParams(q: LendingPgExtendedQuery): string {
  const p = new URLSearchParams();
  if (q.chain_id != null) p.set("chain_id", String(q.chain_id));
  if (q.pool_address) p.set("pool_address", q.pool_address);
  if (q.user_address) p.set("user_address", q.user_address);
  if (q.asset_address) p.set("asset_address", q.asset_address);
  if (q.oracle_address) p.set("oracle_address", q.oracle_address);
  if (q.verifier_address) p.set("verifier_address", q.verifier_address);
  if (q.token_address) p.set("token_address", q.token_address);
  if (q.strategy_address) p.set("strategy_address", q.strategy_address);
  if (q.to_address) p.set("to_address", q.to_address);
  if (q.from_address) p.set("from_address", q.from_address);
  if (q.page != null) p.set("page", String(q.page));
  if (q.page_size != null) p.set("page_size", String(q.page_size));
  const s = p.toString();
  return s ? `?${s}` : "";
}

export type LendingPgListBase = {
  chain_id: number;
  data_source: string;
  page: number;
  page_size: number;
  total: number;
};

export type LendingReserveInitializedRow = {
  id: number;
  chain_id: number;
  pool_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  asset_address: string;
  a_token_address: string;
  debt_token_address: string;
  interest_rate_strategy_address: string;
  ltv_raw: string;
  liquidation_threshold_raw: string;
  liquidation_bonus_raw: string;
  supply_cap_raw: string;
  borrow_cap_raw: string;
  created_at: string;
};

export type LendingReserveInitializedResponse = LendingPgListBase & {
  reserve_initialized: LendingReserveInitializedRow[];
};

export type LendingEmodeCategoryConfiguredRow = {
  id: number;
  chain_id: number;
  pool_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  category_id: number;
  ltv_raw: string;
  liquidation_threshold_raw: string;
  liquidation_bonus_raw: string;
  label: string;
  created_at: string;
};

export type LendingEmodeCategoryConfiguredResponse = LendingPgListBase & {
  emode_category_configured: LendingEmodeCategoryConfiguredRow[];
};

export type LendingHybridPoolSetRow = {
  id: number;
  chain_id: number;
  oracle_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  pool_address: string;
  created_at: string;
};

export type LendingHybridPoolSetResponse = LendingPgListBase & { hybrid_pool_set: LendingHybridPoolSetRow[] };

export type LendingReportsAuthorizedOracleSetRow = {
  id: number;
  chain_id: number;
  verifier_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  oracle_address: string;
  created_at: string;
};

export type LendingReportsAuthorizedOracleSetResponse = LendingPgListBase & {
  reports_authorized_oracle_set: LendingReportsAuthorizedOracleSetRow[];
};

export type LendingReportsTokenSweptRow = {
  id: number;
  chain_id: number;
  verifier_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  token_address: string;
  to_address: string;
  amount_raw: string;
  created_at: string;
};

export type LendingReportsTokenSweptResponse = LendingPgListBase & {
  reports_token_swept: LendingReportsTokenSweptRow[];
};

export type LendingReportsNativeSweptRow = {
  id: number;
  chain_id: number;
  verifier_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  to_address: string;
  amount_raw: string;
  created_at: string;
};

export type LendingReportsNativeSweptResponse = LendingPgListBase & {
  reports_native_swept: LendingReportsNativeSweptRow[];
};

export type LendingChainlinkFeedSetRow = {
  id: number;
  chain_id: number;
  oracle_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  asset_address: string;
  feed_address: string;
  stale_period_raw: string;
  created_at: string;
};

export type LendingChainlinkFeedSetResponse = LendingPgListBase & {
  chainlink_feed_set: LendingChainlinkFeedSetRow[];
};

export type LendingInterestRateStrategyDeployedRow = {
  id: number;
  chain_id: number;
  strategy_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  optimal_utilization_raw: string;
  base_borrow_rate_raw: string;
  slope1_raw: string;
  slope2_raw: string;
  reserve_factor_raw: string;
  created_at: string;
};

export type LendingInterestRateStrategyDeployedResponse = LendingPgListBase & {
  interest_rate_strategy_deployed: LendingInterestRateStrategyDeployedRow[];
};

export type LendingATokenMintRow = {
  id: number;
  chain_id: number;
  token_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  to_address: string;
  scaled_amount_raw: string;
  created_at: string;
};

export type LendingATokenMintResponse = LendingPgListBase & { a_token_mints: LendingATokenMintRow[] };

export type LendingATokenBurnRow = {
  id: number;
  chain_id: number;
  token_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  from_address: string;
  scaled_amount_raw: string;
  created_at: string;
};

export type LendingATokenBurnResponse = LendingPgListBase & { a_token_burns: LendingATokenBurnRow[] };

export type LendingVariableDebtTokenMintRow = {
  id: number;
  chain_id: number;
  token_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  to_address: string;
  scaled_amount_raw: string;
  created_at: string;
};

export type LendingVariableDebtTokenMintResponse = LendingPgListBase & {
  variable_debt_token_mints: LendingVariableDebtTokenMintRow[];
};

export type LendingVariableDebtTokenBurnRow = {
  id: number;
  chain_id: number;
  token_address: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  from_address: string;
  scaled_amount_raw: string;
  created_at: string;
};

export type LendingVariableDebtTokenBurnResponse = LendingPgListBase & {
  variable_debt_token_burns: LendingVariableDebtTokenBurnRow[];
};

export function fetchLendingReserveInitialized(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingReserveInitializedResponse>(`/api/lending/reserve-initialized${toPgExtendedSearchParams(query)}`);
}

export function fetchLendingEmodeCategoryConfigured(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingEmodeCategoryConfiguredResponse>(
    `/api/lending/emode-category-configured${toPgExtendedSearchParams(query)}`,
  );
}

export function fetchLendingHybridPoolSet(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingHybridPoolSetResponse>(`/api/lending/hybrid-pool-set${toPgExtendedSearchParams(query)}`);
}

export function fetchLendingReportsAuthorizedOracleSet(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingReportsAuthorizedOracleSetResponse>(
    `/api/lending/reports-authorized-oracle-set${toPgExtendedSearchParams(query)}`,
  );
}

export function fetchLendingReportsTokenSwept(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingReportsTokenSweptResponse>(`/api/lending/reports-token-swept${toPgExtendedSearchParams(query)}`);
}

export function fetchLendingReportsNativeSwept(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingReportsNativeSweptResponse>(`/api/lending/reports-native-swept${toPgExtendedSearchParams(query)}`);
}

export function fetchLendingChainlinkFeedSet(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingChainlinkFeedSetResponse>(`/api/lending/chainlink-feed-set${toPgExtendedSearchParams(query)}`);
}

export function fetchLendingInterestRateStrategyDeployed(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingInterestRateStrategyDeployedResponse>(
    `/api/lending/interest-rate-strategy-deployed${toPgExtendedSearchParams(query)}`,
  );
}

export function fetchLendingATokenMints(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingATokenMintResponse>(`/api/lending/a-token-mints${toPgExtendedSearchParams(query)}`);
}

export function fetchLendingATokenBurns(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingATokenBurnResponse>(`/api/lending/a-token-burns${toPgExtendedSearchParams(query)}`);
}

export function fetchLendingVariableDebtTokenMints(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingVariableDebtTokenMintResponse>(
    `/api/lending/variable-debt-token-mints${toPgExtendedSearchParams(query)}`,
  );
}

export function fetchLendingVariableDebtTokenBurns(query: LendingPgExtendedQuery = {}) {
  return getJSON<LendingVariableDebtTokenBurnResponse>(
    `/api/lending/variable-debt-token-burns${toPgExtendedSearchParams(query)}`,
  );
}

export function fetchLendingChainStatus() {
  return getJSON<LendingChainStatus>("/api/lending/chain-status");
}

/** GET /api/lending/native-balance — 借贷专用 RPC 上的原生 ETH（wei 字符串） */
export type LendingNativeBalanceResponse = {
  address: string;
  balance_wei: string;
  chain_id: number;
  rpc_scope: string;
};

export function fetchLendingNativeBalance(address: string) {
  const q = new URLSearchParams({ address: address.trim() });
  return getJSON<LendingNativeBalanceResponse>(`/api/lending/native-balance?${q}`);
}

export function fetchLendingSyncStatus(chainId?: number) {
  const q = chainId == null ? "" : `?chain_id=${chainId}`;
  return getJSON<LendingSyncStatus>(`/api/lending/sync-status${q}`);
}

export function fetchLendingContracts(chainId?: number) {
  const q = chainId == null ? "" : `?chain_id=${chainId}`;
  return getJSON<LendingContractsResponse>(`/api/lending/contracts${q}`);
}

export function fetchLendingSupplies(query: LendingListQuery = {}) {
  return getJSON<LendingSuppliesResponse>(`/api/lending/supplies${toSearchParams(query)}`);
}

export type LendingEventListKey = "withdrawals" | "borrows" | "repays" | "liquidations";

export type LendingEventListResponse = {
  chain_id: number;
  data_source: string;
  page: number;
  page_size: number;
  total: number;
} & Partial<{
  withdrawals: unknown[];
  borrows: unknown[];
  repays: unknown[];
  liquidations: unknown[];
}>;

export function fetchLendingEventList(kind: LendingEventListKey, query: LendingListQuery = {}) {
  return getJSON<LendingEventListResponse>(`/api/lending/${kind}${toSearchParams(query)}`);
}
