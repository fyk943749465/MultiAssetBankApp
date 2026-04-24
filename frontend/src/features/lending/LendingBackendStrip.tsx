import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { fetchLendingChainStatus, fetchLendingSyncStatus, type LendingChainStatus, type LendingSyncStatus } from "@/features/lending/api";
import { LENDING_CHAIN_ID } from "@/config/lending";

type State =
  | { status: "idle" | "loading" }
  | { status: "ok"; chain: LendingChainStatus; sync: LendingSyncStatus }
  | { status: "error"; message: string };

/** 不依赖钱包链；在 Base Sepolia 门禁之外也可看后端借贷配置是否就绪 */
export function LendingBackendStrip() {
  const [state, setState] = useState<State>({ status: "idle" });

  useEffect(() => {
    let cancelled = false;
    setState({ status: "loading" });
    (async () => {
      try {
        const [chain, sync] = await Promise.all([
          fetchLendingChainStatus(),
          fetchLendingSyncStatus(LENDING_CHAIN_ID),
        ]);
        if (!cancelled) setState({ status: "ok", chain, sync });
      } catch (e) {
        if (!cancelled) setState({ status: "error", message: e instanceof Error ? e.message : String(e) });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  if (state.status === "error") {
    return (
      <div className="rounded-xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
        借贷后端不可用：{state.message}（请确认 Go 服务已启动且 Vite 代理 <code className="mx-1 rounded bg-muted px-1">/api</code> 指向该服务）
      </div>
    );
  }

  if (state.status !== "ok") {
    return (
      <div className="rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-muted-foreground">
        正在探测借贷后端（专用 RPC / 子图配置）…
      </div>
    );
  }

  const { chain, sync } = state;
  const rpcOk = chain.configured === true;
  const subgraphOk = sync.lending_subgraph_configured;
  const dbOk = sync.database_configured !== false;

  return (
    <div className="flex flex-col gap-2 rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm sm:flex-row sm:flex-wrap sm:items-center sm:gap-3">
      <span className="font-medium text-foreground">后端借贷</span>
      <Badge variant={rpcOk ? "default" : "secondary"} className="w-fit">
        专用 RPC {rpcOk ? `chain #${chain.chain_id}` : "未连接"}
      </Badge>
      <Badge variant={subgraphOk ? "default" : "outline"} className="w-fit">
        子图 {subgraphOk ? "已配置" : "未配置"}
      </Badge>
      <Badge variant={dbOk ? "outline" : "secondary"} className="w-fit">
        PostgreSQL {dbOk ? "已连接" : "未配置"}
      </Badge>
      {sync.lending_subgraph_bearer_present === false && subgraphOk ? (
        <span className="text-xs text-amber-600 dark:text-amber-400">子图 URL 已设但未检测到 Bearer，Studio 可能返回 401</span>
      ) : null}
    </div>
  );
}
