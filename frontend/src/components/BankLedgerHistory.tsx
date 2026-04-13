import { useCallback, useEffect, useState } from "react";
import { useAccount, useReadContract } from "wagmi";
import { sepolia } from "wagmi/chains";
import { formatUnits } from "viem";
import { multiAssetBankAbi } from "../abi/multiAssetBank";
import {
  fetchBankDeposits,
  fetchBankSubgraphDeposits,
  fetchBankSubgraphWithdrawals,
  fetchBankWithdrawals,
  type BankLedgerRow,
} from "../api";
import { getBankAddress } from "../config/bank";
import { SEPOLIA_ERC20_PRESETS } from "../config/sepoliaErc20";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { cn } from "@/lib/utils";

const LEDGER_REFRESH = "bank-ledger-refresh";

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

function ledgerRowKey(r: BankLedgerRow): string {
  if (r.subgraph_entity_id) return r.subgraph_entity_id;
  return `${r.tx_hash}-${r.log_index ?? 0}`;
}

const bank = getBankAddress();

type LedgerSource = "database" | "subgraph";

export function BankLedgerHistory() {
  const { address, isConnected } = useAccount();
  const { data: ethSentinel } = useReadContract({
    address: bank,
    abi: multiAssetBankAbi,
    functionName: "ETH_ADDRESS",
    chainId: sepolia.id,
    query: { enabled: isConnected },
  });
  const [source, setSource] = useState<LedgerSource>("database");
  const [deposits, setDeposits] = useState<BankLedgerRow[]>([]);
  const [withdrawals, setWithdrawals] = useState<BankLedgerRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!address) return;
    setLoading(true);
    setErr(null);
    try {
      if (source === "database") {
        const [d, w] = await Promise.all([fetchBankDeposits(address, 30), fetchBankWithdrawals(address, 30)]);
        setDeposits(d);
        setWithdrawals(w);
      } else {
        const [d, w] = await Promise.all([
          fetchBankSubgraphDeposits(address, 30),
          fetchBankSubgraphWithdrawals(address, 30),
        ]);
        setDeposits(d);
        setWithdrawals(w);
      }
    } catch (e) {
      setDeposits([]);
      setWithdrawals([]);
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [address, source]);

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

  if (!isConnected || !address) return null;

  const tabClass = (active: boolean) =>
    cn(
      "rounded-md px-3 py-1.5 text-xs font-medium transition",
      active ? "bg-secondary text-secondary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
    );

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-primary">充值 / 提现记录</CardTitle>
          <Button variant="outline" size="sm" disabled={loading} onClick={() => void load()}>
            {loading ? "加载中…" : "刷新"}
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="inline-flex rounded-lg border bg-muted/30 p-1">
          <button type="button" className={tabClass(source === "database")} onClick={() => setSource("database")}>
            后端数据库
          </button>
          <button type="button" className={tabClass(source === "subgraph")} onClick={() => setSource("subgraph")}>
            The Graph 子图
          </button>
        </div>

        <p className="font-mono text-xs text-muted-foreground">
          数据来源：
          <span className="text-foreground">
            {source === "database" ? "PostgreSQL（后端索引）" : "The Graph（经 Go 后端代理）"}
          </span>{" "}
          · <span className="text-foreground">{address}</span>
        </p>

        {err && (
          <Alert variant="destructive">
            <AlertDescription>{err}</AlertDescription>
          </Alert>
        )}

        {!err && !loading && deposits.length === 0 && withdrawals.length === 0 && (
          <p className="text-sm text-muted-foreground">
            {source === "database" ? (
              <>
                暂无记录。请确认后端已配置{" "}
                <span className="font-mono text-foreground">BANK_CONTRACT_ADDRESS</span>、
                <span className="font-mono text-foreground">ETH_RPC_URL</span> 且索引器在运行；新交易需等待数秒再刷新。
              </>
            ) : (
              <>
                暂无记录。请确认后端{" "}
                <span className="font-mono text-foreground">SUBGRAPH_URL</span>、
                <span className="font-mono text-foreground">SUBGRAPH_API_KEY</span>{" "}
                已配置，且子图已同步到当前区块；新事件需等待子图索引延迟。
              </>
            )}
          </p>
        )}

        <div className="space-y-6">
          <LedgerTable title="充值（Deposited）" rows={deposits} ethSentinel={ethSentinel} emptyHint="暂无充值记录" />
          <LedgerTable title="提现（Withdrawn）" rows={withdrawals} ethSentinel={ethSentinel} emptyHint="暂无提现记录" />
        </div>
      </CardContent>
    </Card>
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
        <h3 className="mb-1.5 text-sm font-semibold text-foreground">{title}</h3>
        <p className="text-xs text-muted-foreground">{emptyHint}</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border">
      <h3 className="border-b px-3 py-2.5 text-sm font-semibold text-foreground">{title}</h3>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>时间（UTC）</TableHead>
            <TableHead>资产</TableHead>
            <TableHead>数量</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={ledgerRowKey(r)}>
              <TableCell className="font-mono text-[11px] whitespace-nowrap text-muted-foreground">
                {r.block_time ? new Date(r.block_time).toISOString().replace("T", " ").slice(0, 19) : "—"}
              </TableCell>
              <TableCell className="font-mono text-[11px]" title={r.token_address}>
                {tokenLabel(r.token_address, ethSentinel)}
              </TableCell>
              <TableCell className="font-mono">{formatAmount(r, ethSentinel)}</TableCell>
              <TableCell className="font-mono text-muted-foreground">{r.block_number}</TableCell>
              <TableCell>
                <a
                  href={txUrl(r.tx_hash)}
                  target="_blank"
                  rel="noreferrer"
                  className="font-mono text-primary underline decoration-primary/30 underline-offset-2 transition hover:decoration-primary"
                >
                  {r.tx_hash.slice(0, 10)}…
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
