import { useEffect, useMemo, useState } from "react";
import { useAccount } from "wagmi";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  fetchLendingATokenBurns,
  fetchLendingATokenMints,
  fetchLendingChainlinkFeedSet,
  fetchLendingEmodeCategoryConfigured,
  fetchLendingHybridPoolSet,
  fetchLendingInterestRateStrategyDeployed,
  fetchLendingReportsAuthorizedOracleSet,
  fetchLendingReportsNativeSwept,
  fetchLendingReportsTokenSwept,
  fetchLendingReserveInitialized,
  fetchLendingVariableDebtTokenBurns,
  fetchLendingVariableDebtTokenMints,
  type LendingATokenBurnResponse,
  type LendingATokenMintResponse,
  type LendingChainlinkFeedSetResponse,
  type LendingEmodeCategoryConfiguredResponse,
  type LendingHybridPoolSetResponse,
  type LendingInterestRateStrategyDeployedResponse,
  type LendingReportsAuthorizedOracleSetResponse,
  type LendingReportsNativeSweptResponse,
  type LendingReportsTokenSweptResponse,
  type LendingReserveInitializedResponse,
  type LendingVariableDebtTokenBurnResponse,
  type LendingVariableDebtTokenMintResponse,
} from "@/features/lending/api";
import {
  getLendingChainlinkPriceOracleAddress,
  getLendingHybridPriceOracleAddress,
  getLendingInterestRateStrategyAddress,
  getLendingPoolAddress,
  getLendingReportsVerifierAddress,
  LENDING_CHAIN_ID,
} from "@/config/lending";

const BASESCAN = "https://sepolia.basescan.org";

function shortAddr(a: string): string {
  if (!a || a.length < 12) return a;
  return `${a.slice(0, 6)}…${a.slice(-4)}`;
}

function txUrl(hash: string): string {
  return `${BASESCAN}/tx/${hash}`;
}

type Pack = {
  reserve: LendingReserveInitializedResponse | null;
  emode: LendingEmodeCategoryConfiguredResponse | null;
  hybridPool: LendingHybridPoolSetResponse | null;
  revAuth: LendingReportsAuthorizedOracleSetResponse | null;
  revToken: LendingReportsTokenSweptResponse | null;
  revNative: LendingReportsNativeSweptResponse | null;
  clFeed: LendingChainlinkFeedSetResponse | null;
  irDeployed: LendingInterestRateStrategyDeployedResponse | null;
  aMint: LendingATokenMintResponse | null;
  aBurn: LendingATokenBurnResponse | null;
  dMint: LendingVariableDebtTokenMintResponse | null;
  dBurn: LendingVariableDebtTokenBurnResponse | null;
};

const emptyPack: Pack = {
  reserve: null,
  emode: null,
  hybridPool: null,
  revAuth: null,
  revToken: null,
  revNative: null,
  clFeed: null,
  irDeployed: null,
  aMint: null,
  aBurn: null,
  dMint: null,
  dBurn: null,
};

export function LendingExtendedPgSection() {
  const { address } = useAccount();
  const pool = getLendingPoolAddress();
  const hybrid = getLendingHybridPriceOracleAddress();
  const chainlink = getLendingChainlinkPriceOracleAddress();
  const verifier = getLendingReportsVerifierAddress();
  const strategy = getLendingInterestRateStrategyAddress();

  const baseQuery = useMemo(
    () => ({
      chain_id: LENDING_CHAIN_ID,
      page: 1,
      page_size: 12,
    }),
    [],
  );

  const [pack, setPack] = useState<Pack>(emptyPack);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setErr(null);
    (async () => {
      const userQ = address ? { user_address: address } : {};
      try {
        const [
          reserve,
          emode,
          hybridPool,
          revAuth,
          revToken,
          revNative,
          clFeed,
          irDeployed,
          aMint,
          aBurn,
          dMint,
          dBurn,
        ] = await Promise.all([
          fetchLendingReserveInitialized({ ...baseQuery, pool_address: pool }),
          fetchLendingEmodeCategoryConfigured({ ...baseQuery, pool_address: pool }),
          fetchLendingHybridPoolSet({ ...baseQuery, oracle_address: hybrid }),
          fetchLendingReportsAuthorizedOracleSet({ ...baseQuery, verifier_address: verifier }),
          fetchLendingReportsTokenSwept({ ...baseQuery, verifier_address: verifier, ...userQ }),
          fetchLendingReportsNativeSwept({ ...baseQuery, verifier_address: verifier, ...userQ }),
          fetchLendingChainlinkFeedSet({ ...baseQuery, oracle_address: chainlink }),
          fetchLendingInterestRateStrategyDeployed({ ...baseQuery, strategy_address: strategy }),
          fetchLendingATokenMints({ ...baseQuery, ...userQ }),
          fetchLendingATokenBurns({ ...baseQuery, ...userQ }),
          fetchLendingVariableDebtTokenMints({ ...baseQuery, ...userQ }),
          fetchLendingVariableDebtTokenBurns({ ...baseQuery, ...userQ }),
        ]);
        if (!cancelled) {
          setPack({
            reserve,
            emode,
            hybridPool,
            revAuth,
            revToken,
            revNative,
            clFeed,
            irDeployed,
            aMint,
            aBurn,
            dMint,
            dBurn,
          });
        }
      } catch (e) {
        if (!cancelled) {
          setPack(emptyPack);
          setErr(e instanceof Error ? e.message : String(e));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [baseQuery, pool, hybrid, chainlink, verifier, strategy, address]);

  return (
    <Card className="glass-card border-white/10">
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle className="text-lg">PostgreSQL 扩展事件（007）</CardTitle>
          <Badge variant="outline" className="font-mono text-[10px]">
            GET /api/lending/*
          </Badge>
          <Badge variant="secondary">database</Badge>
        </div>
        <CardDescription>
          与迁移 007 表对齐；数据由 RPC 扫块写入。未配置索引器时各表多为空。已连接钱包时，Reports sweep 与 aToken / debt Mint·Burn 请求会附带{" "}
          <code className="rounded bg-muted px-1 font-mono text-xs">user_address</code> 过滤（若适用）。
        </CardDescription>
      </CardHeader>
      <CardContent>{renderExtendedBody(loading, err, pack)}</CardContent>
    </Card>
  );
}

function renderExtendedBody(loading: boolean, err: string | null, pack: Pack) {
  if (loading) {
    return <p className="text-sm text-muted-foreground">加载扩展事件…</p>;
  }
  if (err) {
    return <p className="text-sm text-destructive">{err}</p>;
  }
  return (
    <Tabs defaultValue="reserve" className="w-full">
            <TabsList variant="line" className="mb-4 h-auto min-h-8 w-full flex-wrap justify-start gap-1 py-1">
              <TabsTrigger value="reserve">Reserve 初始化</TabsTrigger>
              <TabsTrigger value="emode">E-Mode 类别</TabsTrigger>
              <TabsTrigger value="hybrid">Hybrid PoolSet</TabsTrigger>
              <TabsTrigger value="rev-auth">Verifier 授权预言机</TabsTrigger>
              <TabsTrigger value="rev-token">Verifier TokenSweep</TabsTrigger>
              <TabsTrigger value="rev-native">Verifier NativeSweep</TabsTrigger>
              <TabsTrigger value="cl-feed">Chainlink FeedSet</TabsTrigger>
              <TabsTrigger value="ir-dep">策略 Deployed</TabsTrigger>
              <TabsTrigger value="am">aToken Mint</TabsTrigger>
              <TabsTrigger value="ab">aToken Burn</TabsTrigger>
              <TabsTrigger value="dm">Debt Mint</TabsTrigger>
              <TabsTrigger value="db">Debt Burn</TabsTrigger>
            </TabsList>

            <TabsContent value="reserve" className="space-y-2">
              <ApiHint path="/api/lending/reserve-initialized?pool_address=…" />
              {renderReserveTable(pack.reserve)}
            </TabsContent>
            <TabsContent value="emode" className="space-y-2">
              <ApiHint path="/api/lending/emode-category-configured?pool_address=…" />
              {renderEmodeTable(pack.emode)}
            </TabsContent>
            <TabsContent value="hybrid" className="space-y-2">
              <ApiHint path="/api/lending/hybrid-pool-set?oracle_address=…" />
              {renderHybridPoolTable(pack.hybridPool)}
            </TabsContent>
            <TabsContent value="rev-auth" className="space-y-2">
              <ApiHint path="/api/lending/reports-authorized-oracle-set?verifier_address=…" />
              {renderRevAuthTable(pack.revAuth)}
            </TabsContent>
            <TabsContent value="rev-token" className="space-y-2">
              <ApiHint path="/api/lending/reports-token-swept?verifier_address=…" />
              {renderRevTokenTable(pack.revToken)}
            </TabsContent>
            <TabsContent value="rev-native" className="space-y-2">
              <ApiHint path="/api/lending/reports-native-swept?verifier_address=…" />
              {renderRevNativeTable(pack.revNative)}
            </TabsContent>
            <TabsContent value="cl-feed" className="space-y-2">
              <ApiHint path="/api/lending/chainlink-feed-set?oracle_address=…" />
              {renderClFeedTable(pack.clFeed)}
            </TabsContent>
            <TabsContent value="ir-dep" className="space-y-2">
              <ApiHint path="/api/lending/interest-rate-strategy-deployed?strategy_address=…" />
              {renderIrDeployedTable(pack.irDeployed)}
            </TabsContent>
            <TabsContent value="am" className="space-y-2">
              <ApiHint path="/api/lending/a-token-mints" />
              {renderATokenMintTable(pack.aMint)}
            </TabsContent>
            <TabsContent value="ab" className="space-y-2">
              <ApiHint path="/api/lending/a-token-burns" />
              {renderATokenBurnTable(pack.aBurn)}
            </TabsContent>
            <TabsContent value="dm" className="space-y-2">
              <ApiHint path="/api/lending/variable-debt-token-mints" />
              {renderDebtMintTable(pack.dMint)}
            </TabsContent>
            <TabsContent value="db" className="space-y-2">
              <ApiHint path="/api/lending/variable-debt-token-burns" />
              {renderDebtBurnTable(pack.dBurn)}
            </TabsContent>
          </Tabs>
  );
}

function ApiHint(props: Readonly<{ path: string }>) {
  const { path } = props;
  return (
    <p className="text-xs text-muted-foreground">
      示例：<span className="font-mono">{path}</span>
    </p>
  );
}

function renderReserveTable(data: LendingReserveInitializedResponse | null) {
  if (!data?.reserve_initialized?.length) return <EmptyHint />;
  const rows = data.reserve_initialized;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>资产</TableHead>
            <TableHead>aToken</TableHead>
            <TableHead>debt</TableHead>
            <TableHead>策略</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.asset_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.a_token_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.debt_token_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.interest_rate_strategy_address)}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderEmodeTable(data: LendingEmodeCategoryConfiguredResponse | null) {
  if (!data?.emode_category_configured?.length) return <EmptyHint />;
  const rows = data.emode_category_configured;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>类别</TableHead>
            <TableHead>label</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell>{r.category_id}</TableCell>
              <TableCell className="max-w-[200px] truncate text-xs">{r.label}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderHybridPoolTable(data: LendingHybridPoolSetResponse | null) {
  if (!data?.hybrid_pool_set?.length) return <EmptyHint />;
  const rows = data.hybrid_pool_set;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Pool</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.pool_address)}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderRevAuthTable(data: LendingReportsAuthorizedOracleSetResponse | null) {
  if (!data?.reports_authorized_oracle_set?.length) return <EmptyHint />;
  const rows = data.reports_authorized_oracle_set;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>预言机</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.oracle_address)}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderRevTokenTable(data: LendingReportsTokenSweptResponse | null) {
  if (!data?.reports_token_swept?.length) return <EmptyHint />;
  const rows = data.reports_token_swept;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>token</TableHead>
            <TableHead>to</TableHead>
            <TableHead>amount</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.token_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.to_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.amount_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderRevNativeTable(data: LendingReportsNativeSweptResponse | null) {
  if (!data?.reports_native_swept?.length) return <EmptyHint />;
  const rows = data.reports_native_swept;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>to</TableHead>
            <TableHead>amount</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.to_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.amount_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderClFeedTable(data: LendingChainlinkFeedSetResponse | null) {
  if (!data?.chainlink_feed_set?.length) return <EmptyHint />;
  const rows = data.chainlink_feed_set;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>资产</TableHead>
            <TableHead>feed</TableHead>
            <TableHead>stalePeriod</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.asset_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.feed_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.stale_period_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderIrDeployedTable(data: LendingInterestRateStrategyDeployedResponse | null) {
  if (!data?.interest_rate_strategy_deployed?.length) return <EmptyHint />;
  const rows = data.interest_rate_strategy_deployed;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>策略</TableHead>
            <TableHead>U_opt</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.strategy_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.optimal_utilization_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderATokenMintTable(data: LendingATokenMintResponse | null) {
  if (!data?.a_token_mints?.length) return <EmptyHint />;
  const rows = data.a_token_mints;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>token</TableHead>
            <TableHead>to</TableHead>
            <TableHead>scaled</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.token_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.to_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.scaled_amount_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderATokenBurnTable(data: LendingATokenBurnResponse | null) {
  if (!data?.a_token_burns?.length) return <EmptyHint />;
  const rows = data.a_token_burns;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>token</TableHead>
            <TableHead>from</TableHead>
            <TableHead>scaled</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.token_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.from_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.scaled_amount_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderDebtMintTable(data: LendingVariableDebtTokenMintResponse | null) {
  if (!data?.variable_debt_token_mints?.length) return <EmptyHint />;
  const rows = data.variable_debt_token_mints;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>token</TableHead>
            <TableHead>to</TableHead>
            <TableHead>scaled</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.token_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.to_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.scaled_amount_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function renderDebtBurnTable(data: LendingVariableDebtTokenBurnResponse | null) {
  if (!data?.variable_debt_token_burns?.length) return <EmptyHint />;
  const rows = data.variable_debt_token_burns;
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>token</TableHead>
            <TableHead>from</TableHead>
            <TableHead>scaled</TableHead>
            <TableHead>区块</TableHead>
            <TableHead>交易</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{shortAddr(r.token_address)}</TableCell>
              <TableCell className="font-mono text-xs">{shortAddr(r.from_address)}</TableCell>
              <TableCell className="font-mono text-xs">{r.scaled_amount_raw}</TableCell>
              <TableCell className="text-xs">{r.block_number}</TableCell>
              <TableCell className="font-mono text-[10px]">
                <a href={txUrl(r.tx_hash)} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                  {shortAddr(r.tx_hash)}
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="text-xs text-muted-foreground">total {data.total}</p>
    </div>
  );
}

function EmptyHint() {
  return <p className="text-sm text-muted-foreground">暂无行数据（total 可能为 0，或索引器尚未写入）。</p>;
}
