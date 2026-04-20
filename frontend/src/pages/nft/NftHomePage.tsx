import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useChainId } from "wagmi";
import { formatUnits } from "viem";
import { sepolia } from "wagmi/chains";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getNftFactoryAddress, getNftMarketplaceAddress, getNftTemplateAddress } from "@/config/nft";
import {
  fetchNftActiveListings,
  fetchNftCollectionByContractAddress,
  fetchNftCollections,
  fetchNftContracts,
  fetchNftSubgraphCollectionByAddress,
  fetchNftSubgraphMeta,
  fetchNftSyncStatus,
  isDbCollection,
  isDbListing,
  type NftCollectionsResponse,
  type NftContractsResponse,
  type NftListingsResponse,
  type NftSubgraphCollectionLookup,
  type NftSyncStatus,
} from "@/features/nft/api";
import { EmptyState, ErrorState, LoadingState } from "@/features/codepulse/components";
import { shortHash } from "@/features/codepulse/format";

function explorerAddressUrl(chainId: number, address: string): string {
  if (chainId === sepolia.id) return `https://sepolia.etherscan.io/address/${address}`;
  return `https://etherscan.io/address/${address}`;
}

function explorerTxUrl(chainId: number, hash: string): string {
  if (chainId === sepolia.id) return `https://sepolia.etherscan.io/tx/${hash}`;
  return `https://etherscan.io/tx/${hash}`;
}

function formatPriceWei(wei: string): string {
  try {
    return `${formatUnits(BigInt(wei), 18)} ETH`;
  } catch {
    return wei;
  }
}

export function NftHomePage() {
  const navigate = useNavigate();
  const walletChainId = useChainId();
  const template = getNftTemplateAddress();
  const factory = getNftFactoryAddress();
  const marketplace = getNftMarketplaceAddress();

  const [loading, setLoading] = useState(true);
  const [sync, setSync] = useState<NftSyncStatus | null>(null);
  const [contracts, setContracts] = useState<NftContractsResponse | null>(null);
  const [collections, setCollections] = useState<NftCollectionsResponse | null>(null);
  const [listings, setListings] = useState<NftListingsResponse | null>(null);
  const [syncErr, setSyncErr] = useState<string | null>(null);
  const [contractsErr, setContractsErr] = useState<string | null>(null);
  const [collectionsErr, setCollectionsErr] = useState<string | null>(null);
  const [listingsErr, setListingsErr] = useState<string | null>(null);

  const [metaLoading, setMetaLoading] = useState(false);
  const [metaJson, setMetaJson] = useState<string | null>(null);
  const [metaErr, setMetaErr] = useState<string | null>(null);

  const [lookupAddr, setLookupAddr] = useState("");
  const [lookupBusy, setLookupBusy] = useState(false);
  const [lookupRes, setLookupRes] = useState<NftSubgraphCollectionLookup | null>(null);
  const [lookupErr, setLookupErr] = useState<string | null>(null);

  /** 子图行点「铸造」：先 GET by-contract 校验 PG，成功才跳转。 */
  const [subgraphMintChecking, setSubgraphMintChecking] = useState<string | null>(null);
  const [subgraphMintErr, setSubgraphMintErr] = useState<string | null>(null);

  const loadCore = useCallback(async () => {
    setLoading(true);
    setSubgraphMintErr(null);
    setSyncErr(null);
    setContractsErr(null);
    setCollectionsErr(null);
    setListingsErr(null);

    const [a, b, c, d] = await Promise.allSettled([
      fetchNftSyncStatus(),
      fetchNftContracts(),
      fetchNftCollections({ page: 1, page_size: 50 }),
      fetchNftActiveListings({ page: 1, page_size: 50 }),
    ]);

    if (a.status === "fulfilled") setSync(a.value);
    else {
      setSync(null);
      setSyncErr(a.reason instanceof Error ? a.reason.message : String(a.reason));
    }
    if (b.status === "fulfilled") setContracts(b.value);
    else {
      setContracts(null);
      setContractsErr(b.reason instanceof Error ? b.reason.message : String(b.reason));
    }
    if (c.status === "fulfilled") setCollections(c.value);
    else {
      setCollections(null);
      setCollectionsErr(c.reason instanceof Error ? c.reason.message : String(c.reason));
    }
    if (d.status === "fulfilled") setListings(d.value);
    else {
      setListings(null);
      setListingsErr(d.reason instanceof Error ? d.reason.message : String(d.reason));
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    void loadCore();
  }, [loadCore]);

  const loadSubgraphMeta = async () => {
    setMetaLoading(true);
    setMetaErr(null);
    setMetaJson(null);
    try {
      const raw = await fetchNftSubgraphMeta();
      setMetaJson(JSON.stringify(raw, null, 2));
    } catch (e) {
      setMetaErr(e instanceof Error ? e.message : String(e));
    } finally {
      setMetaLoading(false);
    }
  };

  const runCollectionLookup = async () => {
    const a = lookupAddr.trim();
    setLookupErr(null);
    setLookupRes(null);
    if (!a) {
      setLookupErr("请输入部署完成后返回的「合集合约地址」（0x 开头 42 字符）。");
      return;
    }
    setLookupBusy(true);
    try {
      const r = await fetchNftSubgraphCollectionByAddress(a);
      setLookupRes(r);
    } catch (e) {
      setLookupErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLookupBusy(false);
    }
  };

  const chainFromApi = sync?.chain_id ?? collections?.chain_id ?? contracts?.chain_id ?? listings?.chain_id;
  const chainMismatch =
    chainFromApi != null && walletChainId > 0 && walletChainId !== chainFromApi;

  const contractRows: { title: string; kind: string; address: string; note: string }[] = [
    { title: "NFTTemplate", kind: "nft_template", address: template, note: "ERC721 逻辑实现 / 克隆所用模板合约。" },
    { title: "NFTFactory", kind: "nft_factory", address: factory, note: "创建克隆合集、CollectionCreated 等工厂事件来源。" },
    { title: "NFTMarketPlace", kind: "nft_marketplace", address: marketplace, note: "挂单、改价、撤单与成交等二级市场逻辑。" },
  ];

  if (loading) {
    return <LoadingState message="正在请求 /api/nft …" />;
  }

  return (
    <div className="space-y-8">
      {chainMismatch ? (
        <Alert>
          <AlertTitle>钱包网络与后端链不一致</AlertTitle>
          <AlertDescription>
            当前钱包 chainId={walletChainId}，后端 NFT API 使用 ETH_RPC_URL 解析为 chain_id={chainFromApi}。读模型以后端为准。
          </AlertDescription>
        </Alert>
      ) : null}

      <Card className="glass-card border-white/10">
        <CardHeader>
          <div className="flex flex-wrap items-center gap-2">
            <CardTitle className="text-xl">NFT 平台概览</CardTitle>
            <Badge variant="secondary" className="font-mono text-[10px]">
              wallet {walletChainId || "—"}
            </Badge>
            {chainFromApi != null ? (
              <Badge variant="outline" className="font-mono text-[10px]">
                API chain {chainFromApi}
              </Badge>
            ) : null}
          </div>
          <CardDescription className="text-[15px] leading-relaxed">
            下方「合集」「挂单」由 Go 后端 <code className="rounded bg-muted px-1 py-0.5 text-xs">/api/nft/*</code> 提供：已配置子图且本页子图有数据时{" "}
            <strong className="text-foreground/90">优先子图</strong>（通常快于扫块入库）；子图不可用或本页子图无数据时用 PostgreSQL。平台合约表仍读库。
          </CardDescription>
          <p className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-sm text-primary/90">
            <Link to="/nft/me" className="font-medium underline-offset-4 hover:underline">
              我的 NFT（按地址查库内持有）
            </Link>
            <Link to="/nft/create" className="font-medium underline-offset-4 hover:underline">
              创作者：一键创建合集（连接钱包即可，无需懂合约）
            </Link>
          </p>
        </CardHeader>
      </Card>

      <div className="grid gap-4 sm:grid-cols-1">
        {contractRows.map((row) => (
          <Card key={row.kind} className="border-white/10 bg-card/40">
            <CardHeader className="pb-2">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <CardTitle className="text-base">{row.title}</CardTitle>
                <Badge variant="outline" className="font-mono text-[10px] uppercase tracking-wide">
                  {row.kind}
                </Badge>
              </div>
              <CardDescription>{row.note}</CardDescription>
            </CardHeader>
            <CardContent>
              <a
                href={explorerAddressUrl(sepolia.id, row.address)}
                target="_blank"
                rel="noreferrer"
                className="break-all font-mono text-sm text-primary underline-offset-4 hover:underline"
              >
                {row.address}
              </a>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* 读模型：sync-status */}
      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle className="text-base">读模型状态</CardTitle>
            <Button variant="outline" size="sm" onClick={() => void loadCore()}>
              刷新
            </Button>
          </div>
          <CardDescription>GET /api/nft/sync-status</CardDescription>
        </CardHeader>
        <CardContent>
          {syncErr ? <ErrorState message={syncErr} /> : null}
          {!syncErr && sync ? (
            <ul className="space-y-2 text-sm text-muted-foreground">
              <li>
                <span className="text-foreground/80">读策略：</span>
                <span className="font-mono text-xs">{sync.read_policy || "—"}</span>
              </li>
              <li>
                <span className="text-foreground/80">子图 URL 已配置：</span>
                {sync.nft_subgraph_configured ? "是" : "否"}
              </li>
              <li>
                <span className="text-foreground/80">子图写入 PG：</span>
                {sync.nft_subgraph_persists_to_pg ? "是（不应出现）" : "否"}
              </li>
              <li className="text-[13px] leading-relaxed">{sync.note}</li>
            </ul>
          ) : !syncErr ? (
            <EmptyState title="无数据" description="sync-status 未返回" />
          ) : null}
        </CardContent>
      </Card>

      {/* 平台合约（库） */}
      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-base">平台合约（PostgreSQL）</CardTitle>
          <CardDescription>GET /api/nft/contracts</CardDescription>
        </CardHeader>
        <CardContent>
          {contractsErr ? <ErrorState message={contractsErr} /> : null}
          {!contractsErr && contracts ? (
            <>
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <Badge variant="secondary" className="font-mono text-[10px]">
                  data_source: {contracts.data_source}
                </Badge>
              </div>
              {contracts.contracts.length === 0 ? (
                <EmptyState title="库中暂无合约行" description="执行迁移与种子或等待索引写入。" />
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>kind</TableHead>
                      <TableHead>address</TableHead>
                      <TableHead>label</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {contracts.contracts.map((c) => (
                      <TableRow key={c.id}>
                        <TableCell className="font-mono text-xs">{c.contract_kind}</TableCell>
                        <TableCell className="font-mono text-xs">
                          {chainFromApi != null ? (
                            <a
                              href={explorerAddressUrl(chainFromApi, c.address)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-4 hover:underline"
                            >
                              {shortHash(c.address, 10, 8)}
                            </a>
                          ) : (
                            shortHash(c.address, 10, 8)
                          )}
                        </TableCell>
                        <TableCell className="text-xs">{c.display_label ?? "—"}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </>
          ) : null}
        </CardContent>
      </Card>

      {/* 合集 */}
      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-base">合集</CardTitle>
          <CardDescription className="space-y-1.5">
            <span>GET /api/nft/collections</span>
            {collections && collections.data_source === "subgraph" ? (
              <span className="block text-[13px] leading-relaxed text-muted-foreground">
                当前为<strong className="text-foreground/80">子图</strong>读模型，可能与重组后的链不一致。点<strong className="text-foreground/80">铸造</strong>时会先请求{" "}
                <code className="rounded bg-muted px-1 font-mono text-[11px]">GET /api/nft/collections/by-contract/…</code>{" "}
                校验 PostgreSQL：<strong className="text-foreground/80">已入库才进入铸造页</strong>，未入库则提示错误、不跳转。
              </span>
            ) : null}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {collectionsErr ? <ErrorState message={collectionsErr} /> : null}
          {!collectionsErr && collections ? (
            <>
              {collections.data_source === "subgraph" && subgraphMintErr ? (
                <Alert variant="destructive" className="mb-3">
                  <AlertTitle>无法前往铸造</AlertTitle>
                  <AlertDescription>{subgraphMintErr}</AlertDescription>
                </Alert>
              ) : null}
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <Badge variant="secondary" className="font-mono text-[10px]">
                  data_source: {collections.data_source}
                </Badge>
                {collections.subgraph_note ? (
                  <span className="text-xs text-amber-600/90 dark:text-amber-400/90">{collections.subgraph_note}</span>
                ) : null}
                {collections.subgraph_fallback_error ? (
                  <span className="text-xs text-destructive">{collections.subgraph_fallback_error}</span>
                ) : null}
                {collections.total_note ? (
                  <span className="text-xs text-muted-foreground">{collections.total_note}</span>
                ) : null}
                {collections.has_more ? <Badge variant="outline">has_more</Badge> : null}
              </div>
              {collections.collections.length === 0 ? (
                <EmptyState title="暂无合集" description="库与子图当前页均无数据。" />
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>id / 实体</TableHead>
                      <TableHead>名称</TableHead>
                      <TableHead>合约</TableHead>
                      <TableHead>创建者</TableHead>
                      <TableHead>操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {collections.collections.map((row) => {
                      const cid = collections.chain_id;
                      if (isDbCollection(row)) {
                        return (
                          <TableRow key={row.id}>
                            <TableCell className="font-mono text-xs">{row.id}</TableCell>
                            <TableCell className="max-w-[140px] truncate text-xs">
                              {row.collection_name ?? "—"}
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              <a
                                href={explorerAddressUrl(cid, row.contract_address)}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary underline-offset-4 hover:underline"
                              >
                                {shortHash(row.contract_address, 8, 6)}
                              </a>
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              <a
                                href={explorerAddressUrl(cid, row.creator_address)}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary underline-offset-4 hover:underline"
                              >
                                {shortHash(row.creator_address, 6, 4)}
                              </a>
                            </TableCell>
                            <TableCell className="space-x-2">
                              <Link
                                to={`/nft/collections/${row.id}`}
                                className="text-xs font-medium text-primary underline-offset-4 hover:underline"
                              >
                                详情
                              </Link>
                              <Link
                                to={`/nft/collections/${row.contract_address}/mint`}
                                className="text-xs font-medium text-primary underline-offset-4 hover:underline"
                              >
                                铸造
                              </Link>
                            </TableCell>
                          </TableRow>
                        );
                      }
                      return (
                        <TableRow key={row.subgraph_entity_id}>
                          <TableCell className="max-w-[120px] truncate font-mono text-[10px]" title={row.subgraph_entity_id}>
                            {shortHash(row.subgraph_entity_id, 12, 6)}
                          </TableCell>
                          <TableCell className="text-xs text-muted-foreground">子图</TableCell>
                          <TableCell className="font-mono text-xs">
                            <a
                              href={explorerAddressUrl(cid, row.collection_address)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-4 hover:underline"
                            >
                              {shortHash(row.collection_address, 8, 6)}
                            </a>
                          </TableCell>
                          <TableCell className="font-mono text-xs">
                            <a
                              href={explorerAddressUrl(cid, row.creator_address)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-4 hover:underline"
                            >
                              {shortHash(row.creator_address, 6, 4)}
                            </a>
                          </TableCell>
                          <TableCell className="max-w-[180px]">
                            <Button
                              type="button"
                              variant="link"
                              size="sm"
                              className="h-auto px-0 py-0 text-xs font-medium text-primary"
                              disabled={subgraphMintChecking === row.collection_address}
                              onClick={() => {
                                void (async () => {
                                  setSubgraphMintErr(null);
                                  setSubgraphMintChecking(row.collection_address);
                                  try {
                                    await fetchNftCollectionByContractAddress(row.collection_address);
                                    navigate(`/nft/collections/${row.collection_address}/mint`);
                                  } catch (e) {
                                    const msg = e instanceof Error ? e.message : String(e);
                                    if (msg.includes("404")) {
                                      setSubgraphMintErr(
                                        "该合集合约尚未写入 PostgreSQL，不能铸造。子图仅作展示，请等待扫块入库后再试。"
                                      );
                                    } else {
                                      setSubgraphMintErr(msg);
                                    }
                                  } finally {
                                    setSubgraphMintChecking(null);
                                  }
                                })();
                              }}
                            >
                              {subgraphMintChecking === row.collection_address ? "校验中…" : "铸造"}
                            </Button>
                            <p className="mt-1 text-[10px] leading-snug text-muted-foreground">先查库，通过才跳转</p>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              )}
            </>
          ) : null}
        </CardContent>
      </Card>

      {/* 活跃挂单 */}
      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-base">活跃挂单 / 上架事件</CardTitle>
          <CardDescription>GET /api/nft/listings/active</CardDescription>
        </CardHeader>
        <CardContent>
          {listingsErr ? <ErrorState message={listingsErr} /> : null}
          {!listingsErr && listings ? (
            <>
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <Badge variant="secondary" className="font-mono text-[10px]">
                  data_source: {listings.data_source}
                </Badge>
                {listings.subgraph_note ? (
                  <span className="text-xs text-amber-600/90 dark:text-amber-400/90">{listings.subgraph_note}</span>
                ) : null}
                {listings.subgraph_fallback_error ? (
                  <span className="text-xs text-destructive">{listings.subgraph_fallback_error}</span>
                ) : null}
              </div>
              {listings.listings.length === 0 ? (
                <EmptyState title="暂无挂单" description="库中无 active 记录，且子图兜底为空或未配置。" />
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>合集</TableHead>
                      <TableHead>token</TableHead>
                      <TableHead>卖家</TableHead>
                      <TableHead>价格</TableHead>
                      <TableHead>tx</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {listings.listings.map((row) => {
                      const cid = listings.chain_id;
                      if (isDbListing(row)) {
                        return (
                          <TableRow key={row.id}>
                            <TableCell className="font-mono text-xs">
                              <a
                                href={explorerAddressUrl(cid, row.collection_address)}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary underline-offset-4 hover:underline"
                              >
                                {shortHash(row.collection_address, 6, 4)}
                              </a>
                            </TableCell>
                            <TableCell className="font-mono text-xs">{row.token_id}</TableCell>
                            <TableCell className="font-mono text-xs">
                              <a
                                href={explorerAddressUrl(cid, row.seller_address)}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary underline-offset-4 hover:underline"
                              >
                                {shortHash(row.seller_address, 6, 4)}
                              </a>
                            </TableCell>
                            <TableCell className="font-mono text-xs">{formatPriceWei(row.price_wei)}</TableCell>
                            <TableCell className="font-mono text-[10px]">
                              <a
                                href={explorerTxUrl(cid, row.listed_tx_hash)}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary underline-offset-4 hover:underline"
                              >
                                {shortHash(row.listed_tx_hash)}
                              </a>
                            </TableCell>
                          </TableRow>
                        );
                      }
                      return (
                        <TableRow key={row.subgraph_entity_id}>
                          <TableCell className="font-mono text-xs">
                            <a
                              href={explorerAddressUrl(cid, row.collection_address)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-4 hover:underline"
                            >
                              {shortHash(row.collection_address, 6, 4)}
                            </a>
                          </TableCell>
                          <TableCell className="font-mono text-xs">{row.token_id}</TableCell>
                          <TableCell className="font-mono text-xs">
                            <a
                              href={explorerAddressUrl(cid, row.seller_address)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-4 hover:underline"
                            >
                              {shortHash(row.seller_address, 6, 4)}
                            </a>
                          </TableCell>
                          <TableCell className="font-mono text-xs">{formatPriceWei(row.price_wei)}</TableCell>
                          <TableCell className="font-mono text-[10px]">
                            <a
                              href={explorerTxUrl(cid, row.transaction_hash)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-4 hover:underline"
                            >
                              {shortHash(row.transaction_hash)}
                            </a>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              )}
            </>
          ) : null}
        </CardContent>
      </Card>

      {/* 子图 _meta（可选） */}
      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-base">子图探测</CardTitle>
          <CardDescription>GET /api/nft/subgraph/meta（需后端配置 SUBGRAPH_NFT_URL）</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <Button variant="outline" size="sm" disabled={metaLoading} onClick={() => void loadSubgraphMeta()}>
            {metaLoading ? "请求中…" : "拉取 _meta"}
          </Button>
          {metaErr ? <ErrorState message={metaErr} /> : null}
          {metaJson ? (
            <pre className="max-h-64 overflow-auto rounded-lg bg-muted/40 p-3 font-mono text-[11px] leading-relaxed">
              {metaJson}
            </pre>
          ) : null}
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-base">按合集合约地址反查子图</CardTitle>
          <CardDescription>
            若 <code className="rounded bg-muted px-1 text-xs">deployProxy</code> 已成功但「合集」表里看不到，把<strong>新克隆出来的 ERC721 合约地址</strong>（不是工厂地址）贴到下面，确认子图是否已索引{" "}
            <code className="rounded bg-muted px-1 text-xs">CollectionCreated</code>。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <Input
              placeholder="0x… 合集合约地址"
              value={lookupAddr}
              onChange={(e) => setLookupAddr(e.target.value)}
              className="font-mono text-sm"
              spellCheck={false}
            />
            <Button variant="outline" size="sm" disabled={lookupBusy} onClick={() => void runCollectionLookup()}>
              {lookupBusy ? "查询中…" : "查询子图"}
            </Button>
          </div>
          {lookupErr ? <ErrorState message={lookupErr} /> : null}
          {lookupRes ? (
            <div className="rounded-lg border border-primary/20 bg-primary/5 p-3 text-sm">
              <p className="font-mono text-xs text-muted-foreground">匹配 {lookupRes.matches.length} 条</p>
              <pre className="mt-2 max-h-56 overflow-auto text-[11px] leading-relaxed">
                {JSON.stringify(lookupRes.matches, null, 2)}
              </pre>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
