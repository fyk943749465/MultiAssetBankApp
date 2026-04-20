const base = import.meta.env.DEV ? "" : (import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:8080");

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${base}${path}`);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text || path}`);
  }
  return res.json() as Promise<T>;
}

export type NftDataSource = "database" | "subgraph";

export type NftSyncStatus = {
  chain_id: number;
  /** 后端读策略说明，如 subgraph_first */
  read_policy?: string;
  nft_subgraph_configured: boolean;
  nft_subgraph_persists_to_pg: boolean;
  note: string;
};

export type NftContractRow = {
  id: number;
  chain_id: number;
  address: string;
  contract_kind: string;
  display_label?: string | null;
  deployed_block?: number | null;
  deployed_tx_hash?: string | null;
  created_at: string;
};

export type NftContractsResponse = {
  chain_id: number;
  data_source: NftDataSource;
  contracts: NftContractRow[];
};

/** 库内合集列表行（GET /api/nft/collections，data_source=database） */
export type NftCollectionDbRow = {
  id: number;
  chain_id: number;
  contract_id: number;
  creator_account_id: number;
  collection_name?: string | null;
  collection_symbol?: string | null;
  base_uri?: string | null;
  deploy_salt_hex?: string | null;
  fee_paid_wei?: string | null;
  created_block_number: number;
  created_tx_hash: string;
  created_log_index: number;
  created_at: string;
  updated_at: string;
  contract_address: string;
  creator_address: string;
};

/** 子图兜底合集项 */
export type NftCollectionSubgraphRow = {
  subgraph_entity_id: string;
  collection_address: string;
  creator_address: string;
  fee_paid_wei: string;
  salt_hex?: string;
  block_number: string;
  block_timestamp: string;
  transaction_hash: string;
};

export type NftCollectionsResponse = {
  chain_id: number;
  data_source: NftDataSource;
  page: number;
  page_size: number;
  total: number;
  collections: NftCollectionDbRow[] | NftCollectionSubgraphRow[];
  subgraph_note?: string;
  total_note?: string;
  has_more?: boolean;
  subgraph_fallback_error?: string;
};

export type NftCollectionDetailResponse = {
  data_source: NftDataSource;
  collection: NftCollectionDbRow;
};

export type NftTokenRow = {
  id: number;
  chain_id: number;
  collection_id: number;
  token_id: string;
  owner_account_id: number;
  mint_tx_hash?: string | null;
  mint_block_number?: number | null;
  last_transfer_tx_hash?: string | null;
  last_transfer_block?: number | null;
  updated_at: string;
  owner_address: string;
};

export type NftTokensResponse = {
  data_source: NftDataSource;
  collection_id: number;
  page: number;
  page_size: number;
  total: number;
  tokens: NftTokenRow[];
};

/** GET /api/nft/holdings?owner= — 库内索引的持有人视图 */
export type NftHoldingRow = NftTokenRow & {
  collection_contract_address: string;
  collection_name?: string | null;
};

export type NftHoldingsResponse = {
  data_source: NftDataSource;
  chain_id: number;
  owner: string;
  page: number;
  page_size: number;
  total: number;
  holdings: NftHoldingRow[];
};

/** 库内活跃挂单 */
export type NftActiveListingDbRow = {
  id: number;
  chain_id: number;
  marketplace_contract_id: number;
  collection_address: string;
  token_id: string;
  seller_account_id: number;
  price_wei: string;
  listed_block_number: number;
  listed_tx_hash: string;
  listing_status: string;
  closed_at?: string | null;
  close_tx_hash?: string | null;
  close_log_index?: number | null;
  created_at: string;
  updated_at: string;
  seller_address: string;
};

export type NftListingSubgraphRow = {
  subgraph_entity_id: string;
  collection_address: string;
  token_id: string;
  seller_address: string;
  price_wei: string;
  block_number: string;
  block_timestamp: string;
  transaction_hash: string;
};

export type NftListingsResponse = {
  chain_id: number;
  data_source: NftDataSource;
  page: number;
  page_size: number;
  total: number;
  listings: NftActiveListingDbRow[] | NftListingSubgraphRow[];
  subgraph_note?: string;
  total_note?: string;
  has_more?: boolean;
  subgraph_fallback_error?: string;
};

export type NftListQuery = {
  page?: number;
  page_size?: number;
};

function listQs(q?: NftListQuery): string {
  const p = new URLSearchParams();
  if (q?.page != null && q.page > 0) p.set("page", String(q.page));
  if (q?.page_size != null && q.page_size > 0) p.set("page_size", String(q.page_size));
  const s = p.toString();
  return s ? `?${s}` : "";
}

function num(v: unknown, fallback: number): number {
  if (typeof v === "number" && !Number.isNaN(v)) return v;
  if (typeof v === "string" && v.trim() !== "") {
    const n = Number(v);
    if (!Number.isNaN(n)) return n;
  }
  return fallback;
}

function str(v: unknown): string {
  if (typeof v === "string") return v;
  if (v == null) return "";
  try {
    return JSON.stringify(v);
  } catch {
    return "[unserializable]";
  }
}

function sanitizeSyncStatus(raw: unknown): NftSyncStatus {
  const j = raw && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  return {
    chain_id: num(j.chain_id, 0),
    read_policy: str(j.read_policy),
    nft_subgraph_configured: Boolean(j.nft_subgraph_configured),
    nft_subgraph_persists_to_pg: Boolean(j.nft_subgraph_persists_to_pg),
    note: str(j.note),
  };
}

function sanitizeContracts(raw: unknown): NftContractsResponse {
  const j = raw && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  const rows = j.contracts;
  return {
    chain_id: num(j.chain_id, 0),
    data_source: j.data_source === "subgraph" ? "subgraph" : "database",
    contracts: Array.isArray(rows) ? (rows as NftContractsResponse["contracts"]) : [],
  };
}

function sanitizeCollections(raw: unknown): NftCollectionsResponse {
  const j = raw && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  const rows = j.collections;
  return {
    chain_id: num(j.chain_id, 0),
    data_source: j.data_source === "subgraph" ? "subgraph" : "database",
    page: num(j.page, 1),
    page_size: num(j.page_size, 20),
    total: num(j.total, 0),
    collections: Array.isArray(rows) ? (rows as NftCollectionsResponse["collections"]) : [],
    subgraph_note: j.subgraph_note == null ? undefined : str(j.subgraph_note),
    total_note: j.total_note == null ? undefined : str(j.total_note),
    has_more: typeof j.has_more === "boolean" ? j.has_more : undefined,
    subgraph_fallback_error:
      j.subgraph_fallback_error == null ? undefined : str(j.subgraph_fallback_error),
  };
}

function sanitizeListings(raw: unknown): NftListingsResponse {
  const j = raw && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  const rows = j.listings;
  return {
    chain_id: num(j.chain_id, 0),
    data_source: j.data_source === "subgraph" ? "subgraph" : "database",
    page: num(j.page, 1),
    page_size: num(j.page_size, 20),
    total: num(j.total, 0),
    listings: Array.isArray(rows) ? (rows as NftListingsResponse["listings"]) : [],
    subgraph_note: j.subgraph_note == null ? undefined : str(j.subgraph_note),
    total_note: j.total_note == null ? undefined : str(j.total_note),
    has_more: typeof j.has_more === "boolean" ? j.has_more : undefined,
    subgraph_fallback_error:
      j.subgraph_fallback_error == null ? undefined : str(j.subgraph_fallback_error),
  };
}

export async function fetchNftSyncStatus(): Promise<NftSyncStatus> {
  const raw = await getJSON<unknown>("/api/nft/sync-status");
  return sanitizeSyncStatus(raw);
}

export async function fetchNftContracts(): Promise<NftContractsResponse> {
  const raw = await getJSON<unknown>("/api/nft/contracts");
  return sanitizeContracts(raw);
}

export async function fetchNftCollections(query?: NftListQuery): Promise<NftCollectionsResponse> {
  const raw = await getJSON<unknown>(`/api/nft/collections${listQs(query)}`);
  return sanitizeCollections(raw);
}

export function fetchNftCollectionById(collectionId: number | string) {
  return getJSON<NftCollectionDetailResponse>(`/api/nft/collections/${collectionId}`);
}

/** GET /api/nft/collections/by-contract/:contract — 仅库内已索引的合集合约；未入库返回 404。 */
export function fetchNftCollectionByContractAddress(contractAddress: string) {
  const a = contractAddress.trim();
  return getJSON<NftCollectionDetailResponse>(`/api/nft/collections/by-contract/${a}`);
}

export function fetchNftCollectionTokens(collectionId: number | string, query?: NftListQuery) {
  return getJSON<NftTokensResponse>(`/api/nft/collections/${collectionId}/tokens${listQs(query)}`);
}

/** 按 owner 查询 PostgreSQL 中已索引的持有记录（非链上实时 balanceOf）。 */
export function fetchNftHoldings(ownerAddress: string, query?: NftListQuery) {
  const p = new URLSearchParams();
  p.set("owner", ownerAddress.trim());
  if (query?.page != null && query.page > 0) p.set("page", String(query.page));
  if (query?.page_size != null && query.page_size > 0) p.set("page_size", String(query.page_size));
  return getJSON<NftHoldingsResponse>(`/api/nft/holdings?${p.toString()}`);
}

export async function fetchNftActiveListings(query?: NftListQuery): Promise<NftListingsResponse> {
  const raw = await getJSON<unknown>(`/api/nft/listings/active${listQs(query)}`);
  return sanitizeListings(raw);
}

/** 子图 _meta；后端未配置子图 URL 时返回 503 */
export function fetchNftSubgraphMeta() {
  return getJSON<unknown>("/api/nft/subgraph/meta");
}

export type NftSubgraphCollectionLookup = {
  data_source: string;
  address: string;
  matches: NftCollectionSubgraphRow[];
};

/** GET /api/nft/subgraph/collection?address= — 按新合集合约地址查子图是否已有 CollectionCreated（404=子图尚无记录） */
export async function fetchNftSubgraphCollectionByAddress(address: string): Promise<NftSubgraphCollectionLookup> {
  const q = new URLSearchParams({ address: address.trim() });
  return getJSON<NftSubgraphCollectionLookup>(`/api/nft/subgraph/collection?${q}`);
}

/** 库内行必有数值主键 id；子图兜底行无 id、有 subgraph_entity_id。 */
export function isDbCollection(row: NftCollectionDbRow | NftCollectionSubgraphRow): row is NftCollectionDbRow {
  return typeof (row as NftCollectionDbRow).id === "number";
}

export function isDbListing(row: NftActiveListingDbRow | NftListingSubgraphRow): row is NftActiveListingDbRow {
  return typeof (row as NftActiveListingDbRow).id === "number";
}
