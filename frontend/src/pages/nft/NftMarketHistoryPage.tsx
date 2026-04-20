import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useAccount, useChainId } from "wagmi";
import { formatEther } from "viem";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  fetchNftMarketTradeEvents,
  type NftMarketTradeEventRow,
  type NftMarketTradeEventType,
  type NftMarketTradeEventsResponse,
} from "@/features/nft/api";
import { shortHash } from "@/features/codepulse/format";

const pageSize = 30;

function etherscanRoot(cid: number): string {
  if (cid === 11_155_111) return "https://sepolia.etherscan.io";
  return "https://etherscan.io";
}

function explorerTx(cid: number, hash: string): string {
  return `${etherscanRoot(cid)}/tx/${hash}`;
}

function explorerToken(cid: number, contract: string): string {
  return `${etherscanRoot(cid)}/token/${contract}`;
}

const EVENT_LABEL: Record<string, string> = {
  ItemListed: "上架",
  ListingPriceUpdated: "改价",
  ListingCanceled: "撤单",
  ItemSold: "成交",
};

function weiEth(wei: string | null | undefined): string {
  if (wei == null || wei === "") return "—";
  try {
    const n = BigInt(wei);
    if (n === 0n) return "0 ETH";
    return `${formatEther(n)} ETH`;
  } catch {
    return wei;
  }
}

function priceSummary(ev: NftMarketTradeEventRow): string {
  switch (ev.event_type) {
    case "ItemListed":
    case "ListingCanceled":
    case "ItemSold":
      return weiEth(ev.price_wei);
    case "ListingPriceUpdated":
      return `${weiEth(ev.old_price_wei)} → ${weiEth(ev.new_price_wei)}`;
    default:
      return "—";
  }
}

export function NftMarketHistoryPage() {
  const { address, isConnected } = useAccount();
  const walletChainId = useChainId();

  const [page, setPage] = useState(1);
  const [eventType, setEventType] = useState<"" | NftMarketTradeEventType>("");
  const [onlyMine, setOnlyMine] = useState(false);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [data, setData] = useState<NftMarketTradeEventsResponse | null>(null);

  const involves =
    onlyMine && isConnected && address ? address : undefined;

  const load = useCallback(async () => {
    setLoading(true);
    setErr(null);
    try {
      const r = await fetchNftMarketTradeEvents({
        page,
        page_size: pageSize,
        event_type: eventType || undefined,
        involves,
      });
      setData(r);
    } catch (e) {
      setData(null);
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [page, eventType, involves]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    setPage(1);
  }, [eventType, onlyMine, address]);

  const total = data?.total ?? 0;
  const apiChainId = data?.chain_id ?? walletChainId;
  const chainMismatch = data?.chain_id != null && walletChainId > 0 && walletChainId !== data.chain_id;

  const rows = useMemo(() => data?.events ?? [], [data]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <Link
          to="/nft"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          ← 返回概览
        </Link>
        <Link
          to="/nft/market"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          ← 活跃挂单
        </Link>
      </div>

      {chainMismatch ? (
        <Alert>
          <AlertTitle>钱包网络与后端链不一致</AlertTitle>
          <AlertDescription>
            当前钱包 chainId={walletChainId}，历史接口 chain_id={data?.chain_id}。历史数据仍可读；链上链接请切到对应网络查看。
          </AlertDescription>
        </Alert>
      ) : null}

      {onlyMine && !isConnected ? (
        <Alert>
          <AlertTitle>未连接钱包</AlertTitle>
          <AlertDescription>勾选「仅与我相关」需连接钱包，否则无法按地址过滤。</AlertDescription>
        </Alert>
      ) : null}

      {err ? (
        <Alert variant="destructive">
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription className="break-words">{err}</AlertDescription>
        </Alert>
      ) : null}

      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">市场事件历史</CardTitle>
          <CardDescription className="space-y-2 text-[13px] leading-relaxed">
            <p>
              数据来自 PostgreSQL 表{" "}
              <code className="rounded bg-muted px-1 font-mono text-xs">nft_market_trade_events</code>
              （后端扫块写入），事件类型包括：上架、改价、撤单、成交。
            </p>
            <p className="text-muted-foreground">
              接口{" "}
              <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/nft/market/trade-events</code>
              与活跃挂单列表数据源独立；索引延迟时最新几笔可能稍晚出现。
            </p>
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-end gap-4">
            <div className="grid gap-1.5">
              <label htmlFor="nft-ev-type" className="text-xs text-muted-foreground">
                事件类型
              </label>
              <select
                id="nft-ev-type"
                className="h-9 min-w-[11rem] rounded-md border border-input bg-background px-2 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/40"
                value={eventType}
                onChange={(e) => setEventType((e.target.value || "") as "" | NftMarketTradeEventType)}
              >
                <option value="">全部</option>
                <option value="ItemListed">上架</option>
                <option value="ListingPriceUpdated">改价</option>
                <option value="ListingCanceled">撤单</option>
                <option value="ItemSold">成交</option>
              </select>
            </div>
            <div className="flex items-center gap-2 pb-0.5">
              <Checkbox
                id="nft-only-mine"
                checked={onlyMine}
                onCheckedChange={(c) => setOnlyMine(c === true)}
              />
              <label htmlFor="nft-only-mine" className="cursor-pointer text-sm font-normal">
                仅与我相关（卖家或买家为当前钱包）
              </label>
            </div>
            <Button type="button" variant="outline" size="sm" disabled={loading} onClick={() => void load()}>
              刷新
            </Button>
          </div>

          {data ? (
            <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <Badge variant="secondary" className="font-mono text-[10px]">
                data_source: {data.data_source}
              </Badge>
              <span>
                共 {total} 条 · 第 {page} 页
              </span>
            </div>
          ) : null}

          {loading && !data ? <p className="text-sm text-muted-foreground">加载事件…</p> : null}

          {!loading && data && rows.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无记录。若刚部署市场，请确认索引器已同步到当前链。</p>
          ) : null}

          {rows.length > 0 ? (
            <div className="overflow-x-auto rounded-lg border border-border/60">
              <table className="w-full min-w-[720px] border-collapse text-left text-[13px]">
                <thead>
                  <tr className="border-b border-border/60 bg-muted/30 text-[11px] uppercase tracking-wide text-muted-foreground">
                    <th className="px-3 py-2 font-medium">时间</th>
                    <th className="px-3 py-2 font-medium">类型</th>
                    <th className="px-3 py-2 font-medium">Token</th>
                    <th className="px-3 py-2 font-medium">卖家</th>
                    <th className="px-3 py-2 font-medium">买家</th>
                    <th className="px-3 py-2 font-medium">价格</th>
                    <th className="px-3 py-2 font-medium">交易</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((ev) => {
                    const when = ev.block_time ? new Date(ev.block_time).toLocaleString() : "—";
                    const typeLabel = EVENT_LABEL[ev.event_type] ?? ev.event_type;
                    return (
                      <tr key={ev.id} className="border-b border-border/40 last:border-0 hover:bg-muted/20">
                        <td className="whitespace-nowrap px-3 py-2 text-muted-foreground">{when}</td>
                        <td className="px-3 py-2">
                          <Badge variant="outline" className="font-normal">
                            {typeLabel}
                          </Badge>
                        </td>
                        <td className="px-3 py-2 font-mono text-[11px]">
                          <a
                            href={explorerToken(apiChainId, ev.collection_address)}
                            target="_blank"
                            rel="noreferrer"
                            className="text-primary underline-offset-2 hover:underline"
                          >
                            {shortHash(ev.collection_address, 6, 4)}
                          </a>
                          <span className="text-muted-foreground"> #{ev.token_id}</span>
                        </td>
                        <td className="px-3 py-2 font-mono text-[11px] text-muted-foreground">
                          {ev.seller_address ? shortHash(ev.seller_address, 6, 4) : "—"}
                        </td>
                        <td className="px-3 py-2 font-mono text-[11px] text-muted-foreground">
                          {ev.buyer_address ? shortHash(ev.buyer_address, 6, 4) : "—"}
                        </td>
                        <td className="whitespace-nowrap px-3 py-2 text-xs">{priceSummary(ev)}</td>
                        <td className="px-3 py-2 font-mono text-[11px]">
                          <a
                            href={explorerTx(apiChainId, ev.tx_hash)}
                            target="_blank"
                            rel="noreferrer"
                            className="text-primary underline-offset-2 hover:underline"
                          >
                            {shortHash(ev.tx_hash)}
                          </a>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : null}

          {data && rows.length > 0 ? (
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1 || loading} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                上一页
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={loading || page * pageSize >= total}
                onClick={() => setPage((p) => p + 1)}
              >
                下一页
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
