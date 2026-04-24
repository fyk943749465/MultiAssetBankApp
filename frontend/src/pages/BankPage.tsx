import { useCallback, useEffect, useState } from "react";
import {
  fetchApiInfo,
  fetchChainStatus,
  fetchCounterValue,
  fetchHealth,
  postCounterCount,
  type ApiInfo,
  type ChainStatus,
} from "../api";
import { ModuleChainGate } from "@/components/ModuleChainGate";
import { BankDeposit } from "../components/BankDeposit";
import { BankLedgerHistory } from "../components/BankLedgerHistory";
import { L1_MODULE_CHAIN_ID } from "@/lib/chain-policy";
import { Card, CardContent, CardHeader, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";

export function BankPage() {
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
    <div className="space-y-12">
      <div className="mx-auto max-w-2xl text-center">
        <p className="inline-flex items-center justify-center gap-2 rounded-full border border-primary/20 bg-primary/10 px-4 py-1.5 text-sm font-medium text-primary shadow-sm mb-6">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary opacity-75"></span>
            <span className="relative inline-flex h-2 w-2 rounded-full bg-primary"></span>
          </span>
          Backend Connected via Vite Proxy
        </p>
        <p className="text-sm leading-relaxed text-muted-foreground">
          银行链上交互（MultiAssetBank 存取等）仅在 <strong className="text-foreground">Ethereum Sepolia（L1）</strong>{" "}
          执行；已连接钱包且网络不对时，下方账本与存取卡片将锁定。后端健康与 Counter 代理仍可在任意网络下浏览。
        </p>
      </div>

      <ModuleChainGate requiredChainId={L1_MODULE_CHAIN_ID} moduleName="银行与后端（链上存取）">
        <div className="space-y-6">
          <BankLedgerHistory />
          <BankDeposit />
        </div>
      </ModuleChainGate>

      {err && (
        <Alert variant="destructive" className="glass-card border-destructive/50 bg-destructive/10 text-destructive-foreground animate-in fade-in slide-in-from-bottom-4">
          <AlertTitle className="font-bold flex items-center gap-2">
            连接后端失败
          </AlertTitle>
          <AlertDescription>
            <pre className="whitespace-pre-wrap break-words font-mono text-sm mt-2">{err}</pre>
            <p className="mt-3 text-xs opacity-90">
              请启动 PostgreSQL 与后端，并配置{" "}
              <code className="rounded border border-destructive/20 bg-destructive/20 px-1.5 py-0.5 font-mono text-[11px]">DATABASE_URL</code>。
            </p>
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <Card className="glass-card hover-lift">
          <CardHeader>
            <CardDescription className="text-xs font-bold uppercase tracking-[0.2em] text-primary/70">
              Health Check
            </CardDescription>
          </CardHeader>
          <CardContent>
            {health ? (
              <pre className="whitespace-pre-wrap rounded-xl border border-white/5 bg-black/40 p-4 font-mono text-xs leading-relaxed text-emerald-400 shadow-inner">
                {JSON.stringify({ status: health }, null, 2)}
              </pre>
            ) : (
              !err && <p className="text-sm text-muted-foreground animate-pulse">加载中…</p>
            )}
          </CardContent>
        </Card>

        <Card className="glass-card hover-lift">
          <CardHeader>
            <CardDescription className="text-xs font-bold uppercase tracking-[0.2em] text-primary/70">
              API Info
            </CardDescription>
          </CardHeader>
          <CardContent>
            {info ? (
              <pre className="whitespace-pre-wrap rounded-xl border border-white/5 bg-black/40 p-4 font-mono text-xs leading-relaxed text-blue-400 shadow-inner">
                {JSON.stringify(info, null, 2)}
              </pre>
            ) : (
              !err && <p className="text-sm text-muted-foreground animate-pulse">加载中…</p>
            )}
          </CardContent>
        </Card>

        <Card className="glass-card hover-lift md:col-span-2">
          <CardHeader>
            <CardDescription className="text-xs font-bold uppercase tracking-[0.2em] text-primary/70">
              Chain Status (Backend RPC)
            </CardDescription>
          </CardHeader>
          <CardContent>
            {chain ? (
              <pre className="whitespace-pre-wrap rounded-xl border border-white/5 bg-black/40 p-4 font-mono text-xs leading-relaxed text-purple-400 shadow-inner">
                {JSON.stringify(chain, null, 2)}
              </pre>
            ) : (
              !err && <p className="text-sm text-muted-foreground animate-pulse">加载中…</p>
            )}
          </CardContent>
        </Card>

        <Card className="glass-card hover-lift md:col-span-2">
          <CardHeader>
            <CardDescription className="text-xs font-bold uppercase tracking-[0.2em] text-primary/70">
              Counter Contract (Backend Proxy)
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <code className="rounded border border-primary/20 bg-primary/10 px-2 py-1 font-mono text-[11px] text-primary/80">
                GET /api/contract/counter/value
              </code>
              <code className="rounded border border-primary/20 bg-primary/10 px-2 py-1 font-mono text-[11px] text-primary/80">
                POST /api/contract/counter/count
              </code>
            </div>
            <div className="flex flex-wrap gap-3 mt-4">
              <Button 
                variant="outline" 
                onClick={() => void loadCounter()} 
                disabled={counterLoading}
                className="bg-transparent border-primary/30 hover:bg-primary/10 hover:text-primary transition-all duration-300"
              >
                {counterLoading ? "读取中…" : "刷新数值"}
              </Button>
              <Button 
                onClick={() => void handleCount()} 
                disabled={countLoading}
                className="bg-primary/90 hover:bg-primary hover:shadow-[0_0_15px_rgba(45,212,191,0.4)] transition-all duration-300"
              >
                {countLoading ? "发送交易中…" : "调用 count()"}
              </Button>
            </div>
            
            {counterErr && <pre className="text-sm text-destructive font-mono p-3 rounded-lg bg-destructive/10 border border-destructive/20">{counterErr}</pre>}
            {counterValue !== null && !counterErr && (
              <pre className="whitespace-pre-wrap rounded-xl border border-white/5 bg-black/40 p-4 font-mono text-xs leading-relaxed text-emerald-400 shadow-inner mt-4">
                {JSON.stringify({ value: counterValue }, null, 2)}
              </pre>
            )}
            {!counterLoading && counterValue === null && !counterErr && (
              <p className="text-sm text-muted-foreground mt-4 italic">暂无数据（检查后端 COUNTER_CONTRACT_ADDRESS）</p>
            )}
            {countErr && <pre className="mt-2 text-sm text-destructive font-mono p-3 rounded-lg bg-destructive/10 border border-destructive/20">count: {countErr}</pre>}
            {countTx && (
              <pre className="whitespace-pre-wrap rounded-xl border border-white/5 bg-black/40 p-4 font-mono text-xs leading-relaxed text-emerald-400 shadow-inner mt-4">
                {JSON.stringify({ tx_hash: countTx }, null, 2)}
              </pre>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
