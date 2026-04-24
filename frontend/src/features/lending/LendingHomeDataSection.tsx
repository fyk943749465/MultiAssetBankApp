import { useEffect, useState } from "react";
import { formatEther } from "viem";
import { useAccount } from "wagmi";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  fetchLendingContracts,
  fetchLendingNativeBalance,
  fetchLendingSupplies,
  type LendingContractsResponse,
  type LendingNativeBalanceResponse,
  type LendingSupplyPg,
  type LendingSupplySubgraph,
  type LendingSuppliesResponse,
} from "@/features/lending/api";
import { getLendingPoolAddress, LENDING_CHAIN_ID } from "@/config/lending";

const BASESCAN = "https://sepolia.basescan.org";

function shortAddr(a: string): string {
  if (!a || a.length < 12) return a;
  return `${a.slice(0, 6)}…${a.slice(-4)}`;
}

function isSubgraphSource(source: LendingSuppliesResponse["data_source"]): source is "subgraph" {
  return source === "subgraph";
}

export function LendingHomeDataSection() {
  const { address } = useAccount();
  const pool = getLendingPoolAddress();

  const [contracts, setContracts] = useState<LendingContractsResponse | null>(null);
  const [contractsErr, setContractsErr] = useState<string | null>(null);

  const [supplies, setSupplies] = useState<LendingSuppliesResponse | null>(null);
  const [suppliesErr, setSuppliesErr] = useState<string | null>(null);

  const [nativeBal, setNativeBal] = useState<LendingNativeBalanceResponse | null>(null);
  const [nativeErr, setNativeErr] = useState<string | null>(null);

  useEffect(() => {
    if (!address) {
      setNativeBal(null);
      setNativeErr(null);
      return;
    }
    let cancelled = false;
    (async () => {
      setNativeErr(null);
      try {
        const b = await fetchLendingNativeBalance(address);
        if (!cancelled) setNativeBal(b);
      } catch (e) {
        if (!cancelled) setNativeBal(null);
        if (!cancelled) setNativeErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [address]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setContractsErr(null);
      try {
        const c = await fetchLendingContracts(LENDING_CHAIN_ID);
        if (!cancelled) setContracts(c);
      } catch (e) {
        if (!cancelled) setContracts(null);
        if (!cancelled) setContractsErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setSuppliesErr(null);
      try {
        const q: Parameters<typeof fetchLendingSupplies>[0] = {
          chain_id: LENDING_CHAIN_ID,
          pool_address: pool,
          page: 1,
          page_size: 15,
        };
        if (address) q.user_address = address;
        const s = await fetchLendingSupplies(q);
        if (!cancelled) setSupplies(s);
      } catch (e) {
        if (!cancelled) setSupplies(null);
        if (!cancelled) setSuppliesErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [pool, address]);

  return (
    <div className="space-y-8">
      <Card className="glass-card border-white/10">
        <CardHeader>
          <div className="flex flex-wrap items-center gap-2">
            <CardTitle className="text-lg">原生 ETH 余额（借贷链）</CardTitle>
            <Badge variant="outline" className="font-mono text-[10px]">
              GET /api/lending/native-balance
            </Badge>
          </div>
          <CardDescription>
            由后端通过 <strong className="text-foreground">LENDING_ETH_RPC_URL</strong> 查询，与 Sepolia 主 RPC 无关。需已连接钱包；也可自行带{" "}
            <code className="rounded bg-muted px-1 font-mono text-xs">?address=0x…</code> 调用。
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!address ? (
            <p className="text-sm text-muted-foreground">连接钱包后显示当前账户在 Base Sepolia（借贷 RPC）上的 ETH 余额。</p>
          ) : nativeErr ? (
            <p className="text-sm text-destructive">{nativeErr}</p>
          ) : nativeBal == null ? (
            <p className="text-sm text-muted-foreground">查询中…</p>
          ) : (
            <div className="space-y-1 text-sm">
              <p>
                <span className="text-muted-foreground">账户</span>{" "}
                <span className="font-mono text-xs">{shortAddr(nativeBal.address)}</span>
              </p>
              <p className="text-lg font-semibold tabular-nums text-foreground">
                {formatEther(BigInt(nativeBal.balance_wei))} ETH
              </p>
              <p className="text-xs text-muted-foreground">
                chain_id {nativeBal.chain_id} · wei <span className="font-mono">{nativeBal.balance_wei}</span>
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="glass-card border-white/10">
        <CardHeader>
          <div className="flex flex-wrap items-center gap-2">
            <CardTitle className="text-lg">数据库合约登记</CardTitle>
            <Badge variant="outline" className="font-mono text-[10px]">
              GET /api/lending/contracts
            </Badge>
          </div>
          <CardDescription>与迁移 006/007 种子一致；后端权威列表。</CardDescription>
        </CardHeader>
        <CardContent>
          {contractsErr ? (
            <p className="text-sm text-destructive">{contractsErr}</p>
          ) : contracts == null ? (
            <p className="text-sm text-muted-foreground">加载中…</p>
          ) : contracts.contracts.length === 0 ? (
            <p className="text-sm text-muted-foreground">当前 chain_id 下无登记合约。</p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>类型</TableHead>
                    <TableHead>地址</TableHead>
                    <TableHead>标签</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {contracts.contracts.map((r) => (
                    <TableRow key={`${r.contract_kind}-${r.address}`}>
                      <TableCell className="font-medium">{r.contract_kind}</TableCell>
                      <TableCell className="font-mono text-xs">
                        <a
                          href={`${BASESCAN}/address/${r.address}`}
                          target="_blank"
                          rel="noreferrer"
                          className="text-primary underline-offset-4 hover:underline"
                        >
                          {shortAddr(r.address)}
                        </a>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{r.display_label ?? "—"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="glass-card border-white/10">
        <CardHeader>
          <div className="flex flex-wrap items-center gap-2">
            <CardTitle className="text-lg">Supply 流水</CardTitle>
            <Badge variant="outline" className="font-mono text-[10px]">
              GET /api/lending/supplies
            </Badge>
            <Badge variant="secondary">{supplies?.data_source ?? "—"}</Badge>
          </div>
          <CardDescription>
            已按当前池地址过滤；连接钱包时附加用户地址过滤。子图有数据时优先子图，否则 PostgreSQL。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {supplies?.subgraph_fallback_reason ? (
            <p className="text-xs text-muted-foreground">
              子图回退原因：<span className="font-mono">{supplies.subgraph_fallback_reason}</span>
            </p>
          ) : null}
          {suppliesErr ? (
            <p className="text-sm text-destructive">{suppliesErr}</p>
          ) : supplies == null ? (
            <p className="text-sm text-muted-foreground">加载中…</p>
          ) : supplies.supplies.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              暂无 Supply 记录（子图与库均为空，或索引器尚未写入 lending_supplies）。
            </p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>用户</TableHead>
                    <TableHead>资产</TableHead>
                    <TableHead>数量(raw)</TableHead>
                    <TableHead>区块</TableHead>
                    <TableHead>交易</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {supplies.supplies.map((row) =>
                    isSubgraphSource(supplies.data_source) ? (
                      <TableRow key={(row as LendingSupplySubgraph).id}>
                        <TableCell className="font-mono text-xs">{shortAddr((row as LendingSupplySubgraph).user)}</TableCell>
                        <TableCell className="font-mono text-xs">{shortAddr((row as LendingSupplySubgraph).asset)}</TableCell>
                        <TableCell className="font-mono text-xs">{(row as LendingSupplySubgraph).amount}</TableCell>
                        <TableCell className="text-xs">{(row as LendingSupplySubgraph).blockNumber}</TableCell>
                        <TableCell className="font-mono text-[10px]">
                          <a
                            href={`${BASESCAN}/tx/${(row as LendingSupplySubgraph).transactionHash}`}
                            target="_blank"
                            rel="noreferrer"
                            className="text-primary underline-offset-4 hover:underline"
                          >
                            {shortAddr((row as LendingSupplySubgraph).transactionHash)}
                          </a>
                        </TableCell>
                      </TableRow>
                    ) : (
                      <TableRow key={(row as LendingSupplyPg).id}>
                        <TableCell className="font-mono text-xs">{shortAddr((row as LendingSupplyPg).user_address)}</TableCell>
                        <TableCell className="font-mono text-xs">{shortAddr((row as LendingSupplyPg).asset_address)}</TableCell>
                        <TableCell className="font-mono text-xs">{(row as LendingSupplyPg).amount_raw}</TableCell>
                        <TableCell className="text-xs">{(row as LendingSupplyPg).block_number}</TableCell>
                        <TableCell className="font-mono text-[10px]">
                          <a
                            href={`${BASESCAN}/tx/${(row as LendingSupplyPg).tx_hash}`}
                            target="_blank"
                            rel="noreferrer"
                            className="text-primary underline-offset-4 hover:underline"
                          >
                            {shortAddr((row as LendingSupplyPg).tx_hash)}
                          </a>
                        </TableCell>
                      </TableRow>
                    ),
                  )}
                </TableBody>
              </Table>
            </div>
          )}
          {supplies?.total_note ? <p className="text-xs text-muted-foreground">{supplies.total_note}</p> : null}
        </CardContent>
      </Card>
    </div>
  );
}
