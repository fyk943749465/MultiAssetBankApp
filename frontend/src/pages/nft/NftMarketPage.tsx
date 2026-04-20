import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  useAccount,
  useChainId,
  usePublicClient,
  useReadContract,
  useSwitchChain,
  useWriteContract,
} from "wagmi";
import { formatEther, getAddress, isAddress, parseEther, type Address } from "viem";
import { sepolia } from "wagmi/chains";
import { ImageOff } from "lucide-react";
import { nftTemplateAbi } from "@/abi/nftTemplate";
import { nftMarketplaceAbi } from "@/abi/nftMarketplace";
import { erc721ApproveAbi } from "@/abi/erc721Approve";
import { getNftMarketplaceAddress } from "@/config/nft";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import {
  fetchNftActiveListings,
  fetchNftVerifyActiveListing,
  isDbListing,
  type NftDataSource,
  type NftListingsResponse,
} from "@/features/nft/api";
import { fetchMetadataFromTokenUri, imageUrlFromMetadataField } from "@/features/nft/metadata";
import { shortHash } from "@/features/codepulse/format";

const pageSize = 30;
const META_CONCURRENCY = 4;
const THUMB_PX = 72;

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

type MediaState =
  | { status: "loading" }
  | { status: "ready"; imageUrl: string; displayName?: string }
  | { status: "empty"; detail?: string }
  | { status: "error"; detail: string };

type NormalizedListing = {
  key: string;
  collection: Address;
  tokenId: string;
  priceWei: string;
  seller: string;
  dataSource: NftDataSource;
  listedTxHash?: string;
};

function listingKey(collection: string, tokenId: string): string {
  return `${collection.toLowerCase()}-${tokenId}`;
}

function normalizeRows(listings: NftListingsResponse["listings"]): NormalizedListing[] {
  return listings.map((row) => {
    if (isDbListing(row)) {
      return {
        key: `db-${row.id}`,
        collection: getAddress(row.collection_address as Address),
        tokenId: row.token_id,
        priceWei: row.price_wei,
        seller: row.seller_address.toLowerCase(),
        dataSource: "database",
        listedTxHash: row.listed_tx_hash,
      };
    }
    return {
      key: `sg-${row.subgraph_entity_id}`,
      collection: getAddress(row.collection_address as Address),
      tokenId: row.token_id,
      priceWei: row.price_wei,
      seller: row.seller_address.toLowerCase(),
      dataSource: "subgraph",
      listedTxHash: row.transaction_hash,
    };
  });
}

const thumbFrame = `shrink-0 overflow-hidden rounded-md [width:${THUMB_PX}px] [height:${THUMB_PX}px]`;
const thumbChecker =
  "bg-[repeating-conic-gradient(#e7e5e4_0%_25%,#d6d3d1_0%_50%)_50%_/_10px_10px] dark:bg-[repeating-conic-gradient(#292524_0%_25%,#44403c_0%_50%)_50%_/_8px_8px]";

function MarketThumb({
  media,
  onImageError,
}: {
  media: MediaState | undefined;
  onImageError: () => void;
}) {
  if (media?.status === "ready") {
    return (
      <img
        src={media.imageUrl}
        alt=""
        width={THUMB_PX}
        height={THUMB_PX}
        className={`${thumbFrame} ${thumbChecker} object-contain [image-rendering:pixelated]`}
        loading="lazy"
        onError={onImageError}
      />
    );
  }
  if (media?.status === "empty" || media?.status === "error") {
    return (
      <div
        className={`flex ${thumbFrame} flex-col items-center justify-center gap-0.5 bg-muted/35 p-1 text-center text-[9px] leading-snug text-muted-foreground`}
        title={media.status === "empty" ? media.detail ?? "无图" : media.detail}
      >
        <ImageOff className="size-4 shrink-0 opacity-45" aria-hidden />
        <span className="line-clamp-2">{media.status === "error" ? "失败" : "无图"}</span>
      </div>
    );
  }
  return (
    <div className={`flex ${thumbFrame} items-center justify-center bg-muted/25 text-[11px] text-muted-foreground`}>
      <span className="opacity-60">···</span>
    </div>
  );
}

export function NftMarketPage() {
  const { address, isConnected } = useAccount();
  const walletChainId = useChainId();
  const publicClient = usePublicClient({ chainId: sepolia.id });
  const { switchChainAsync } = useSwitchChain();
  const { writeContractAsync } = useWriteContract();
  const marketAddr = getNftMarketplaceAddress();

  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [listErr, setListErr] = useState<string | null>(null);
  const [listData, setListData] = useState<NftListingsResponse | null>(null);

  const [mediaByKey, setMediaByKey] = useState<Record<string, MediaState>>({});
  const [mediaFatal, setMediaFatal] = useState<string | null>(null);

  const [actionMsg, setActionMsg] = useState<string | null>(null);
  const [actionBusy, setActionBusy] = useState<string | null>(null);
  /** 每条挂单的改价输入（ETH 字符串），key 同 listingKey */
  const [newPriceEthByKey, setNewPriceEthByKey] = useState<Record<string, string>>({});

  const [listCollection, setListCollection] = useState("");
  const [listTokenId, setListTokenId] = useState("");
  const [listPriceEth, setListPriceEth] = useState("");
  const [listBusy, setListBusy] = useState(false);

  const { data: marketPaused } = useReadContract({
    address: marketAddr,
    abi: nftMarketplaceAbi,
    functionName: "paused",
    query: { enabled: walletChainId === sepolia.id },
  });

  const loadListings = useCallback(async () => {
    setLoading(true);
    setListErr(null);
    try {
      const r = await fetchNftActiveListings({ page, page_size: pageSize });
      setListData(r);
    } catch (e) {
      setListData(null);
      setListErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    void loadListings();
  }, [loadListings]);

  const rows = useMemo(() => (listData?.listings ? normalizeRows(listData.listings) : []), [listData]);
  const apiChainId = listData?.chain_id ?? walletChainId;
  const chainMismatch = listData?.chain_id != null && walletChainId > 0 && walletChainId !== listData.chain_id;
  const canReadChain = Boolean(publicClient) && walletChainId === sepolia.id && apiChainId === sepolia.id;

  useEffect(() => {
    setMediaFatal(null);
    setMediaByKey({});
    if (rows.length === 0 || !publicClient || !canReadChain) {
      return;
    }

    const ac = new AbortController();
    const keys = rows.map((r) => listingKey(r.collection, r.tokenId));
    setMediaByKey(Object.fromEntries(keys.map((k) => [k, { status: "loading" as const }])));

    (async () => {
      try {
        const uriResults = await publicClient.multicall({
          contracts: rows.map((r) => ({
            address: r.collection,
            abi: nftTemplateAbi,
            functionName: "tokenURI" as const,
            args: [BigInt(r.tokenId)] as const,
          })),
          allowFailure: true,
        });
        if (ac.signal.aborted) return;

        for (let i = 0; i < rows.length; i += META_CONCURRENCY) {
          if (ac.signal.aborted) return;
          const batch = rows.slice(i, i + META_CONCURRENCY);
          await Promise.all(
            batch.map(async (r, j) => {
              const ii = i + j;
              const key = listingKey(r.collection, r.tokenId);
              const ur = uriResults[ii];
              if (ur.status !== "success" || !ur.result) {
                setMediaByKey((prev) => ({
                  ...prev,
                  [key]: {
                    status: "error",
                    detail: ur.status === "failure" ? String(ur.error) : "无 tokenURI",
                  },
                }));
                return;
              }
              const tokenURI = ur.result as string;
              try {
                const meta = await fetchMetadataFromTokenUri(tokenURI);
                if (ac.signal.aborted) return;
                if (!meta.image?.trim()) {
                  setMediaByKey((prev) => ({ ...prev, [key]: { status: "empty", detail: "无 image" } }));
                  return;
                }
                const imageUrl = imageUrlFromMetadataField(meta.image);
                setMediaByKey((prev) => ({
                  ...prev,
                  [key]: { status: "ready", imageUrl, displayName: meta.name },
                }));
              } catch (e) {
                if (ac.signal.aborted) return;
                setMediaByKey((prev) => ({
                  ...prev,
                  [key]: { status: "error", detail: e instanceof Error ? e.message : String(e) },
                }));
              }
            }),
          );
        }
      } catch (e) {
        if (!ac.signal.aborted) {
          setMediaFatal(e instanceof Error ? e.message : String(e));
        }
      }
    })();

    return () => ac.abort();
  }, [rows, publicClient, canReadChain]);

  async function ensureSepolia() {
    if (switchChainAsync && walletChainId !== sepolia.id) {
      await switchChainAsync({ chainId: sepolia.id });
    }
  }

  async function handleBuy(row: NormalizedListing) {
    if (!address || !publicClient) {
      setActionMsg("请先连接钱包。");
      return;
    }
    if (marketPaused) {
      setActionMsg("市场合约已暂停。");
      return;
    }
    const key = listingKey(row.collection, row.tokenId);
    setActionBusy(key);
    setActionMsg(null);
    try {
      const verify = await fetchNftVerifyActiveListing(row.collection, row.tokenId);
      if (!verify.active || !verify.listing) {
        setActionMsg(
          "索引库中尚无该笔活跃挂单，无法发起购买。子图仅作展示时，请等待扫块写入数据库后再买；或刷新后确认 data_source 为 database。",
        );
        return;
      }
      let listedWei: bigint;
      let rowWei: bigint;
      try {
        listedWei = BigInt(verify.listing.price_wei);
        rowWei = BigInt(row.priceWei);
      } catch {
        setActionMsg("价格字段异常，请刷新页面后重试。");
        return;
      }
      if (listedWei !== rowWei) {
        setActionMsg("库内价格与当前列表不一致，请刷新后再购买。");
        return;
      }
      if (verify.listing.seller_address.toLowerCase() !== row.seller) {
        setActionMsg("库内卖家与当前列表不一致，请刷新后再购买。");
        return;
      }

      await ensureSepolia();
      const price = BigInt(row.priceWei);
      const { request } = await publicClient.simulateContract({
        address: marketAddr,
        abi: nftMarketplaceAbi,
        functionName: "buyItem",
        args: [row.collection, BigInt(row.tokenId)],
        account: address,
        value: price,
      });
      const hash = await writeContractAsync(request);
      await publicClient.waitForTransactionReceipt({ hash });
      setActionMsg(`购买已提交并成功上链：${hash.slice(0, 10)}… 列表可能稍后由索引器更新。`);
      await loadListings();
    } catch (e) {
      setActionMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setActionBusy(null);
    }
  }

  async function handleCancel(row: NormalizedListing) {
    if (!address || !publicClient) {
      setActionMsg("请先连接钱包。");
      return;
    }
    if (!address || address.toLowerCase() !== row.seller) {
      setActionMsg("仅卖家可撤单。");
      return;
    }
    const key = listingKey(row.collection, row.tokenId);
    setActionBusy(key);
    setActionMsg(null);
    try {
      await ensureSepolia();
      const { request } = await publicClient.simulateContract({
        address: marketAddr,
        abi: nftMarketplaceAbi,
        functionName: "cancelListing",
        args: [row.collection, BigInt(row.tokenId)],
        account: address,
      });
      const hash = await writeContractAsync(request);
      await publicClient.waitForTransactionReceipt({ hash });
      setActionMsg("撤单成功。列表将随索引更新。");
      await loadListings();
    } catch (e) {
      setActionMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setActionBusy(null);
    }
  }

  function updatePriceBusyKey(mkey: string): string {
    return `${mkey}#updatePrice`;
  }

  async function handleUpdatePrice(row: NormalizedListing) {
    if (!address || !publicClient) {
      setActionMsg("请先连接钱包。");
      return;
    }
    if (address.toLowerCase() !== row.seller) {
      setActionMsg("仅卖家可改价。");
      return;
    }
    if (marketPaused) {
      setActionMsg("市场合约已暂停。");
      return;
    }
    const mkey = listingKey(row.collection, row.tokenId);
    const raw = (newPriceEthByKey[mkey] ?? "").trim();
    let newWei: bigint;
    try {
      newWei = parseEther(raw || "0");
    } catch {
      setActionMsg("新价格（ETH）格式无效。");
      return;
    }
    if (newWei <= 0n) {
      setActionMsg("新价格须大于 0。");
      return;
    }
    if (newWei === BigInt(row.priceWei)) {
      setActionMsg("新价格与当前标价相同。");
      return;
    }

    const busy = updatePriceBusyKey(mkey);
    setActionBusy(busy);
    setActionMsg(null);
    try {
      await ensureSepolia();
      const { request } = await publicClient.simulateContract({
        address: marketAddr,
        abi: nftMarketplaceAbi,
        functionName: "updateListingPrice",
        args: [row.collection, BigInt(row.tokenId), newWei],
        account: address,
      });
      const hash = await writeContractAsync(request);
      await publicClient.waitForTransactionReceipt({ hash });
      setActionMsg("改价成功。列表将随索引更新。");
      setNewPriceEthByKey((prev) => {
        const next = { ...prev };
        delete next[mkey];
        return next;
      });
      await loadListings();
    } catch (e) {
      setActionMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setActionBusy(null);
    }
  }

  async function handleList() {
    if (!address || !publicClient) {
      setActionMsg("请先连接钱包。");
      return;
    }
    const c = listCollection.trim();
    const tid = listTokenId.trim();
    const pe = listPriceEth.trim();
    if (!isAddress(c)) {
      setActionMsg("合集合约地址无效。");
      return;
    }
    if (!/^\d+$/.test(tid)) {
      setActionMsg("tokenId 须为非负整数。");
      return;
    }
    let priceWei: bigint;
    try {
      priceWei = parseEther(pe || "0");
    } catch {
      setActionMsg("价格（ETH）格式无效。");
      return;
    }
    if (priceWei <= 0n) {
      setActionMsg("价格须大于 0。");
      return;
    }
    if (marketPaused) {
      setActionMsg("市场合约已暂停。");
      return;
    }

    setListBusy(true);
    setActionMsg(null);
    try {
      await ensureSepolia();
      const coll = getAddress(c as Address);
      const tidBn = BigInt(tid);

      const ownerOnChain = await publicClient.readContract({
        address: coll,
        abi: erc721ApproveAbi,
        functionName: "ownerOf",
        args: [tidBn],
      });
      if ((ownerOnChain as string).toLowerCase() !== address.toLowerCase()) {
        setActionMsg("当前钱包不是该 token 的 owner，无法上架。");
        return;
      }

      const approved = await publicClient.readContract({
        address: coll,
        abi: erc721ApproveAbi,
        functionName: "isApprovedForAll",
        args: [address, marketAddr],
      });
      if (!approved) {
        const { request: r1 } = await publicClient.simulateContract({
          address: coll,
          abi: erc721ApproveAbi,
          functionName: "setApprovalForAll",
          args: [marketAddr, true],
          account: address,
        });
        const h1 = await writeContractAsync(r1);
        await publicClient.waitForTransactionReceipt({ hash: h1 });
      }

      const { request: r2 } = await publicClient.simulateContract({
        address: marketAddr,
        abi: nftMarketplaceAbi,
        functionName: "listFormatItem",
        args: [coll, tidBn, priceWei],
        account: address,
      });
      const h2 = await writeContractAsync(r2);
      await publicClient.waitForTransactionReceipt({ hash: h2 });
      setActionMsg("上架成功。索引延迟时列表可能稍后才出现。");
      setListCollection("");
      setListTokenId("");
      setListPriceEth("");
      await loadListings();
    } catch (e) {
      setActionMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setListBusy(false);
    }
  }

  const total = listData?.total ?? 0;

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
          to="/nft/me"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          我的 NFT
        </Link>
        <Link
          to="/nft/market/history"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          挂单 / 成交历史
        </Link>
      </div>

      {chainMismatch ? (
        <Alert>
          <AlertTitle>钱包网络与后端链不一致</AlertTitle>
          <AlertDescription>
            当前钱包 chainId={walletChainId}，列表接口 chain_id={listData?.chain_id}。请切换到 Sepolia 后再购买或上架。
          </AlertDescription>
        </Alert>
      ) : null}

      {listData?.data_source === "subgraph" && listData.subgraph_note ? (
        <Alert>
          <AlertTitle>子图数据说明</AlertTitle>
          <AlertDescription>{listData.subgraph_note}</AlertDescription>
        </Alert>
      ) : null}

      {listData?.data_source === "database" && listData.subgraph_fallback_error ? (
        <Alert variant="destructive">
          <AlertTitle>子图查询失败，当前为数据库兜底列表</AlertTitle>
          <AlertDescription className="break-words">{listData.subgraph_fallback_error}</AlertDescription>
        </Alert>
      ) : null}

      {marketPaused ? (
        <Alert variant="destructive">
          <AlertTitle>市场已暂停</AlertTitle>
          <AlertDescription>合约 paused=true，购买与上架交易会失败。</AlertDescription>
        </Alert>
      ) : null}

      {actionMsg ? (
        <Alert variant={actionMsg.includes("成功") ? "default" : "destructive"}>
          <AlertTitle>{actionMsg.includes("成功") ? "操作结果" : "提示"}</AlertTitle>
          <AlertDescription className="break-words">{actionMsg}</AlertDescription>
        </Alert>
      ) : null}

      {mediaFatal ? (
        <Alert variant="destructive">
          <AlertTitle>图片加载失败</AlertTitle>
          <AlertDescription>{mediaFatal}</AlertDescription>
        </Alert>
      ) : null}

      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">NFT 市场</CardTitle>
          <CardDescription className="space-y-2 text-[13px] leading-relaxed">
            <p>
              列表接口为{" "}
              <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/nft/listings/active</code>
              。已配置子图时先查子图（含空列表）；子图失败时用 PostgreSQL 活跃挂单兜底并返回错误说明。购买前会调用{" "}
              <code className="rounded bg-muted px-1 font-mono text-xs">verify-active</code>
              ，仅在库内存在对应活跃挂单且价格、卖家与列表一致时才发起链上购买。
            </p>
            <p className="text-muted-foreground">
              购买/撤单/上架/改价均为链上调用 <code className="rounded bg-muted px-1 font-mono text-xs">NFTMarketPlace</code>（
              {shortHash(marketAddr, 8, 6)}）。
            </p>
            <p className="text-muted-foreground">
              缩略图为 72px 展示 + 元数据 <code className="font-mono text-[11px]">image</code>；与「我的 NFT」相同，部分 IPFS 可能无法在浏览器中加载。
            </p>
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {!isConnected ? (
            <Alert>
              <AlertTitle>未连接钱包</AlertTitle>
              <AlertDescription>购买、撤单、上架需连接钱包（Sepolia）。</AlertDescription>
            </Alert>
          ) : null}

          {listErr ? (
            <Alert variant="destructive">
              <AlertTitle>加载失败</AlertTitle>
              <AlertDescription>{listErr}</AlertDescription>
            </Alert>
          ) : null}

          {loading && !listData ? <p className="text-sm text-muted-foreground">加载挂单…</p> : null}

          {!loading && listData && rows.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无活跃挂单。</p>
          ) : null}

          {listData ? (
            <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <Badge variant="secondary" className="font-mono text-[10px]">
                data_source: {listData.data_source}
              </Badge>
              <span>
                共 {total} 条 · 第 {page} 页
              </span>
              <Button type="button" variant="outline" size="sm" disabled={loading} onClick={() => void loadListings()}>
                刷新
              </Button>
            </div>
          ) : null}

          {rows.length > 0 ? (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {rows.map((row) => {
                const mkey = listingKey(row.collection, row.tokenId);
                const m = mediaByKey[mkey];
                const isSeller = address?.toLowerCase() === row.seller;
                const updateBusy = actionBusy === updatePriceBusyKey(mkey);
                const rowActionBusy = actionBusy === mkey || updateBusy;
                const buyDisabled =
                  !isConnected ||
                  !canReadChain ||
                  Boolean(marketPaused) ||
                  isSeller ||
                  rowActionBusy;

                return (
                  <Card
                    key={row.key}
                    className="group overflow-hidden rounded-xl border border-white/10 bg-gradient-to-b from-card/90 to-card/40 shadow-sm transition-[box-shadow,transform] duration-200 hover:-translate-y-0.5 hover:shadow-md hover:ring-1 hover:ring-primary/15"
                  >
                    <div className="flex flex-col gap-0">
                      <div className="flex justify-center bg-gradient-to-b from-muted/35 via-muted/20 to-transparent px-3 pb-2 pt-3">
                        <div className="rounded-xl border border-border/60 bg-background/40 p-1.5 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.04)] ring-1 ring-black/5 dark:ring-white/5">
                          {!canReadChain ? (
                            <div
                              className={`flex ${thumbFrame} items-center justify-center bg-muted/30 px-1 text-center text-[9px] text-muted-foreground`}
                            >
                              切 Sepolia 看图
                            </div>
                          ) : (
                            <MarketThumb
                              media={m}
                              onImageError={() =>
                                setMediaByKey((prev) => ({
                                  ...prev,
                                  [mkey]: { status: "error", detail: "图片无法显示" },
                                }))
                              }
                            />
                          )}
                        </div>
                      </div>
                      <div className="space-y-2 border-t border-border/50 px-3 pb-3 pt-2.5">
                        {m?.status === "ready" && m.displayName ? (
                          <p className="line-clamp-2 text-sm font-semibold leading-snug">{m.displayName}</p>
                        ) : (
                          <p className="text-sm font-semibold text-muted-foreground">Token #{row.tokenId}</p>
                        )}
                        <p className="font-mono text-[11px] text-muted-foreground">
                          <a
                            href={explorerToken(apiChainId, row.collection)}
                            target="_blank"
                            rel="noreferrer"
                            className="text-primary underline-offset-2 hover:underline"
                          >
                            {shortHash(row.collection, 8, 6)}
                          </a>
                          <span className="mx-1 text-muted-foreground/50">·</span>#{row.tokenId}
                        </p>
                        <p className="text-sm font-medium text-primary">
                          {formatEther(BigInt(row.priceWei))} ETH
                        </p>
                        <p className="font-mono text-[10px] text-muted-foreground">
                          卖家 {shortHash(row.seller, 8, 6)}
                        </p>
                        {row.listedTxHash ? (
                          <p className="font-mono text-[10px]">
                            <a
                              href={explorerTx(apiChainId, row.listedTxHash)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-2 hover:underline"
                            >
                              上架 tx {shortHash(row.listedTxHash)}
                            </a>
                          </p>
                        ) : null}
                        <div className="flex flex-wrap gap-2 pt-1">
                          <Button
                            type="button"
                            size="sm"
                            disabled={buyDisabled}
                            onClick={() => void handleBuy(row)}
                          >
                            {actionBusy === mkey ? "处理中…" : "购买"}
                          </Button>
                          {isSeller ? (
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              disabled={!canReadChain || rowActionBusy}
                              onClick={() => void handleCancel(row)}
                            >
                              撤单
                            </Button>
                          ) : null}
                        </div>
                        {isSeller ? (
                          <div className="space-y-1.5 rounded-md border border-border/50 bg-muted/20 p-2">
                            <p className="text-[10px] font-medium text-muted-foreground">卖家改价（ETH）</p>
                            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                              <Input
                                placeholder={`新价，当前 ${formatEther(BigInt(row.priceWei))}`}
                                value={newPriceEthByKey[mkey] ?? ""}
                                onChange={(e) =>
                                  setNewPriceEthByKey((prev) => ({ ...prev, [mkey]: e.target.value }))
                                }
                                className="h-8 font-mono text-xs"
                                spellCheck={false}
                              />
                              <Button
                                type="button"
                                size="sm"
                                variant="secondary"
                                className="shrink-0"
                                disabled={!canReadChain || Boolean(marketPaused) || rowActionBusy}
                                onClick={() => void handleUpdatePrice(row)}
                              >
                                {updateBusy ? "提交中…" : "改价"}
                              </Button>
                            </div>
                          </div>
                        ) : null}
                      </div>
                    </div>
                  </Card>
                );
              })}
            </div>
          ) : null}

          {listData && rows.length > 0 ? (
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

          <Card className="border-white/10 bg-muted/15">
            <CardHeader className="pb-2">
              <CardTitle className="text-base">上架 NFT</CardTitle>
              <CardDescription className="text-xs leading-relaxed">
                需为 token owner。首次上架会先请求 <code className="font-mono">setApprovalForAll(市场, true)</code>，再{" "}
                <code className="font-mono">listFormatItem</code>。仅支持本平台工厂部署的合集（合约会校验）。
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid gap-2 sm:grid-cols-3">
                <Input
                  placeholder="合集合约 0x…"
                  value={listCollection}
                  onChange={(e) => setListCollection(e.target.value)}
                  className="font-mono text-xs sm:col-span-1"
                  spellCheck={false}
                />
                <Input
                  placeholder="tokenId"
                  value={listTokenId}
                  onChange={(e) => setListTokenId(e.target.value)}
                  className="font-mono text-xs"
                  spellCheck={false}
                />
                <Input
                  placeholder="价格（ETH）"
                  value={listPriceEth}
                  onChange={(e) => setListPriceEth(e.target.value)}
                  className="font-mono text-xs"
                  spellCheck={false}
                />
              </div>
              <Button type="button" disabled={listBusy || !isConnected || !canReadChain} onClick={() => void handleList()}>
                {listBusy ? "提交中…" : "授权（如需）并上架"}
              </Button>
            </CardContent>
          </Card>
        </CardContent>
      </Card>
    </div>
  );
}
