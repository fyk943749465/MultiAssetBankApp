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
  const q = chainId != null ? `?chain_id=${chainId}` : "";
  return getJSON<LendingSyncStatus>(`/api/lending/sync-status${q}`);
}

export function fetchLendingContracts(chainId?: number) {
  const q = chainId != null ? `?chain_id=${chainId}` : "";
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
