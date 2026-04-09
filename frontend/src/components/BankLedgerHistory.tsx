import { useCallback, useEffect, useState } from "react";
import { useAccount, useReadContract } from "wagmi";
import { sepolia } from "wagmi/chains";
import { formatUnits } from "viem";
import { multiAssetBankAbi } from "../abi/multiAssetBank";
import { fetchBankDeposits, fetchBankWithdrawals, type BankLedgerRow } from "../api";
import { getBankAddress } from "../config/bank";
import { SEPOLIA_ERC20_PRESETS } from "../config/sepoliaErc20";
import { btnGhost, sectionTitleAccent, surface } from "../ui/styles";

const LEDGER_REFRESH = "bank-ledger-refresh";

/** 存款成功后由 BankDeposit 触发，便于本组件重新拉库 */
export function notifyBankLedgerRefresh() {
  window.dispatchEvent(new Event(LEDGER_REFRESH));
}

function tokenLabel(tokenAddress: string, ethSentinel: string | undefined): string {
  const t = tokenAddress.toLowerCase();
  if (ethSentinel && t === ethSentinel.toLowerCase()) return "ETH";
  const preset = SEPOLIA_ERC20_PRESETS.find((p) => p.address.toLowerCase() === t);
  if (preset) return preset.symbol;
  return `${tokenAddress.slice(0, 8)}…${tokenAddress.slice(-6)}`;
}

function formatAmount(row: BankLedgerRow, ethSentinel: string | undefined): string {
  const label = tokenLabel(row.token_address, ethSentinel);
  try {
    if (label === "ETH" || (ethSentinel && row.token_address.toLowerCase() === ethSentinel.toLowerCase())) {
      return `${formatUnits(BigInt(row.amount_raw), 18)} ETH`;
    }
    return `${formatUnits(BigInt(row.amount_raw), 18)} ${label}`;
  } catch {
    return `${row.amount_raw} wei (${label})`;
  }
}

function txUrl(hash: string) {
  return `https://sepolia.etherscan.io/tx/${hash}`;
}

const bank = getBankAddress();

export function BankLedgerHistory() {
  const { address, isConnected } = useAccount();
  const { data: ethSentinel } = useReadContract({
    address: bank,
    abi: multiAssetBankAbi,
    functionName: "ETH_ADDRESS",
    chainId: sepolia.id,
    query: { enabled: isConnected },
  });
  const [deposits, setDeposits] = useState<BankLedgerRow[]>([]);
  const [withdrawals, setWithdrawals] = useState<BankLedgerRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!address) return;
    setLoading(true);
    setErr(null);
    try {
      const [d, w] = await Promise.all([fetchBankDeposits(address, 30), fetchBankWithdrawals(address, 30)]);
      setDeposits(d);
      setWithdrawals(w);
    } catch (e) {
      setDeposits([]);
      setWithdrawals([]);
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [address]);

  useEffect(() => {
    if (!isConnected || !address) {
      setDeposits([]);
      setWithdrawals([]);
      setErr(null);
      return;
    }
    void load();
  }, [isConnected, address, load]);

  useEffect(() => {
    const onRefresh = () => {
      if (address) void load();
    };
    window.addEventListener(LEDGER_REFRESH, onRefresh);
    return () => window.removeEventListener(LEDGER_REFRESH, onRefresh);
  }, [address, load]);

  if (!isConnected || !address) {
    return null;
  }

  return (
    <section className={surface}>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className={sectionTitleAccent}>充值 / 提现记录</h2>
        <button type="button" className={btnGhost} disabled={loading} onClick={() => void load()}>
          {loading ? "加载中…" : "刷新"}
        </button>
      </div>
      <p className="mb-4 font-mono text-xs text-slate-500">
        数据来源：后端索引库 · <span className="text-slate-400">{address}</span>
      </p>
      {err && (
        <p className="mb-4 rounded-xl border border-red-500/30 bg-red-950/30 px-3 py-2.5 text-sm text-red-300">{err}</p>
      )}
      {!err && !loading && deposits.length === 0 && withdrawals.length === 0 && (
        <p className="text-sm text-slate-400">
          暂无记录。请确认后端已配置 <span className="font-mono text-slate-300">BANK_CONTRACT_ADDRESS</span>、
          <span className="font-mono text-slate-300">ETH_RPC_URL</span> 且索引器在运行；新交易需等待数秒再刷新。
        </p>
      )}

      <div className="space-y-6">
        <LedgerTable title="充值（Deposited）" rows={deposits} ethSentinel={ethSentinel} emptyHint="暂无充值记录" />
        <LedgerTable
          title="提现（Withdrawn）"
          rows={withdrawals}
          ethSentinel={ethSentinel}
          emptyHint="暂无提现记录"
        />
      </div>
    </section>
  );
}

function LedgerTable({
  title,
  rows,
  ethSentinel,
  emptyHint,
}: {
  title: string;
  rows: BankLedgerRow[];
  ethSentinel: string | undefined;
  emptyHint: string;
}) {
  if (rows.length === 0) {
    return (
      <div>
        <h3 className="mb-1.5 text-sm font-semibold text-slate-200">{title}</h3>
        <p className="text-xs text-slate-500">{emptyHint}</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-slate-800/80 bg-slate-950/35">
      <h3 className="border-b border-slate-800/80 px-3 py-2.5 text-sm font-semibold text-slate-200">{title}</h3>
      <table className="w-full min-w-[36rem] border-collapse text-left text-xs">
        <thead>
          <tr className="border-b border-slate-800/90 bg-slate-950/50 text-[11px] uppercase tracking-wide text-slate-500">
            <th className="py-2.5 pl-3 pr-2 font-semibold">时间（UTC）</th>
            <th className="py-2.5 pr-2 font-semibold">资产</th>
            <th className="py-2.5 pr-2 font-semibold">数量</th>
            <th className="py-2.5 pr-2 font-semibold">区块</th>
            <th className="py-2.5 pr-3 font-semibold">交易</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr
              key={`${r.tx_hash}-${r.log_index}`}
              className="border-b border-slate-800/50 text-slate-300 transition hover:bg-slate-800/30"
            >
              <td className="py-2.5 pl-3 pr-2 font-mono text-[11px] whitespace-nowrap text-slate-400">
                {r.block_time ? new Date(r.block_time).toISOString().replace("T", " ").slice(0, 19) : "—"}
              </td>
              <td className="py-2.5 pr-2 font-mono text-[11px]" title={r.token_address}>
                {tokenLabel(r.token_address, ethSentinel)}
              </td>
              <td className="py-2.5 pr-2 font-mono text-slate-200">{formatAmount(r, ethSentinel)}</td>
              <td className="py-2.5 pr-2 font-mono text-slate-400">{r.block_number}</td>
              <td className="py-2.5 pr-3">
                <a
                  href={txUrl(r.tx_hash)}
                  target="_blank"
                  rel="noreferrer"
                  className="font-mono text-emerald-400/95 underline decoration-emerald-500/30 underline-offset-2 transition hover:text-emerald-300 hover:decoration-emerald-400"
                >
                  {r.tx_hash.slice(0, 10)}…
                </a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
