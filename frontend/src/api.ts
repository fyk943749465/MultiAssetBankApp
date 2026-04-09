const base = import.meta.env.DEV ? "" : (import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:8080");

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${base}${path}`);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text || path}`);
  }
  return res.json() as Promise<T>;
}

async function postJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${base}${path}`, {
    method: "POST",
    headers: { Accept: "application/json" },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text || path}`);
  }
  return res.json() as Promise<T>;
}

export type ApiInfo = { name: string; version: string };

export type ChainStatus =
  | { configured: false; message: string }
  | { configured: true; chain_id: number };

export function fetchHealth() {
  return getJSON<{ status: string }>("/health");
}

export function fetchApiInfo() {
  return getJSON<ApiInfo>("/api/info");
}

export function fetchChainStatus() {
  return getJSON<ChainStatus>("/api/chain/status");
}

export type CounterValue = { value: string };

export type CounterCountResult = { tx_hash: string };

/** GET /api/contract/counter/value — 只读 get() */
export function fetchCounterValue() {
  return getJSON<CounterValue>("/api/contract/counter/value");
}

/** POST /api/contract/counter/count — 发送交易调用 count() */
export function postCounterCount() {
  return postJSON<CounterCountResult>("/api/contract/counter/count");
}

/** 后端索引的链上充值 / 提现（PostgreSQL） */
export type BankLedgerRow = {
  id: number;
  chain_id: number;
  tx_hash: string;
  log_index: number;
  block_number: number;
  block_time: string;
  token_address: string;
  user_address: string;
  amount_raw: string;
  created_at: string;
};

export async function fetchBankDeposits(user: string, limit = 30): Promise<BankLedgerRow[]> {
  const q = new URLSearchParams({ user, limit: String(limit) });
  const j = await getJSON<{ deposits: BankLedgerRow[] }>(`/api/bank/deposits?${q}`);
  return j.deposits ?? [];
}

export async function fetchBankWithdrawals(user: string, limit = 30): Promise<BankLedgerRow[]> {
  const q = new URLSearchParams({ user, limit: String(limit) });
  const j = await getJSON<{ withdrawals: BankLedgerRow[] }>(`/api/bank/withdrawals?${q}`);
  return j.withdrawals ?? [];
}
