import { useCallback, useEffect, useState } from "react";
import {
  fetchApiInfo,
  fetchChainStatus,
  fetchCounterValue,
  fetchHealth,
  postCounterCount,
  type ApiInfo,
  type ChainStatus,
} from "./api";
import { BankDeposit } from "./components/BankDeposit";
import { BankLedgerHistory } from "./components/BankLedgerHistory";
import { ConnectWallet } from "./components/ConnectWallet";
import { btn, code, preJson, sectionTitle, surface, surfaceDanger, surfaceMuted } from "./ui/styles";

export default function App() {
  const [health, setHealth] = useState<string | null>(null);
  const [info, setInfo] = useState<ApiInfo | null>(null);
  const [chain, setChain] = useState<ChainStatus | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const [counterValue, setCounterValue] = useState<string | null>(null);
  const [counterErr, setCounterErr] = useState<string | null>(null);
  const [counterLoading, setCounterLoading] = useState(false);
  const [countLoading, setCountLoading] = useState(false);
  const [countTx, setCountTx] = useState<string | null>(null);
  const [countErr, setCountErr] = useState<string | null>(null);

  const loadCounter = useCallback(async () => {
    setCounterLoading(true);
    setCounterErr(null);
    try {
      const r = await fetchCounterValue();
      setCounterValue(r.value);
    } catch (e) {
      setCounterValue(null);
      setCounterErr(e instanceof Error ? e.message : String(e));
    } finally {
      setCounterLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [h, i, c] = await Promise.all([
          fetchHealth(),
          fetchApiInfo(),
          fetchChainStatus(),
        ]);
        if (cancelled) return;
        setHealth(h.status);
        setInfo(i);
        setChain(c);
        setErr(null);
      } catch (e) {
        if (cancelled) return;
        setErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    void loadCounter();
  }, [loadCounter]);

  async function handleCount() {
    setCountLoading(true);
    setCountErr(null);
    setCountTx(null);
    try {
      const r = await postCounterCount();
      setCountTx(r.tx_hash);
      await loadCounter();
    } catch (e) {
      setCountErr(e instanceof Error ? e.message : String(e));
    } finally {
      setCountLoading(false);
    }
  }

  return (
    <div className="mx-auto min-h-screen max-w-5xl px-4 pb-16 pt-10 sm:px-6 lg:px-8">
      <header className="mb-10 text-center sm:mb-12">
        <p className="mb-2 text-xs font-semibold uppercase tracking-[0.35em] text-emerald-500/80">
          Sepolia · Wagmi · Viem
        </p>
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
          <span className="text-gradient-brand">GO-CHAIN</span>
        </h1>
        <p className="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-slate-400">
          链上交互与 Go 后端通过 Vite 代理 <code className={code}>/api</code> 联通；下方可连接钱包、查看账本与存款。
        </p>
      </header>

      <div className="mb-10 space-y-5">
        <ConnectWallet />
        <BankLedgerHistory />
        <BankDeposit />
      </div>

      {err && (
        <div className={`${surfaceDanger} mb-6`}>
          <h2 className={sectionTitle}>连接后端</h2>
          <pre className="whitespace-pre-wrap break-words text-sm text-red-200/90">{err}</pre>
          <p className="mt-3 text-xs text-red-300/80">
            请启动 PostgreSQL 与后端，并配置 <code className={code}>DATABASE_URL</code>。
          </p>
        </div>
      )}

      <div className="grid gap-5 md:grid-cols-2">
        <div className={surfaceMuted}>
          <h2 className={sectionTitle}>健康检查</h2>
          {health ? (
            <pre className={`${preJson} text-emerald-300/90`}>{JSON.stringify({ status: health }, null, 2)}</pre>
          ) : (
            !err && <p className="text-sm text-slate-500">加载中…</p>
          )}
        </div>

        <div className={surfaceMuted}>
          <h2 className={sectionTitle}>API 信息</h2>
          {info ? (
            <pre className={preJson}>{JSON.stringify(info, null, 2)}</pre>
          ) : (
            !err && <p className="text-sm text-slate-500">加载中…</p>
          )}
        </div>

        <div className={`${surfaceMuted} md:col-span-2`}>
          <h2 className={sectionTitle}>链状态（后端 RPC）</h2>
          {chain ? (
            <pre className={preJson}>{JSON.stringify(chain, null, 2)}</pre>
          ) : (
            !err && <p className="text-sm text-slate-500">加载中…</p>
          )}
        </div>

        <div className={`${surface} md:col-span-2`}>
          <h2 className={sectionTitle}>Counter 合约（后端代理）</h2>
          <p className="mb-4 text-xs text-slate-500">
            <code className={code}>GET /api/contract/counter/value</code>
            <span className="mx-2 text-slate-600">·</span>
            <code className={code}>POST /api/contract/counter/count</code>
          </p>
          <div className="mb-4 flex flex-wrap gap-2">
            <button type="button" onClick={() => void loadCounter()} disabled={counterLoading} className={btn}>
              {counterLoading ? "读取中…" : "刷新数值"}
            </button>
            <button type="button" onClick={() => void handleCount()} disabled={countLoading} className={btn}>
              {countLoading ? "发送交易中…" : "调用 count()"}
            </button>
          </div>
          {counterErr && <pre className="text-sm text-red-400">{counterErr}</pre>}
          {counterValue !== null && !counterErr && (
            <pre className={`${preJson} text-emerald-300/90`}>{JSON.stringify({ value: counterValue }, null, 2)}</pre>
          )}
          {!counterLoading && counterValue === null && !counterErr && (
            <p className="text-sm text-slate-500">暂无数据（检查后端 COUNTER_CONTRACT_ADDRESS）</p>
          )}
          {countErr && <pre className="mt-2 text-sm text-red-400">count: {countErr}</pre>}
          {countTx && (
            <pre className={`${preJson} mt-2 text-emerald-300/90`}>{JSON.stringify({ tx_hash: countTx }, null, 2)}</pre>
          )}
        </div>
      </div>

      <footer className="mt-14 border-t border-slate-800/80 pt-8 text-center text-xs text-slate-600">
        GO-CHAIN · 本地开发请使用系统浏览器打开前端地址以使用 <code className={code}>window.ethereum</code>
      </footer>
    </div>
  );
}
