import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useAccount, useChainId, usePublicClient } from "wagmi";
import { getAddress, isAddress, type Address } from "viem";
import { sepolia } from "wagmi/chains";
import { ImageOff } from "lucide-react";
import { nftTemplateAbi } from "@/abi/nftTemplate";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { fetchNftHoldings, type NftHoldingsResponse, type NftHoldingRow } from "@/features/nft/api";
import { fetchMetadataFromTokenUri, imageUrlFromMetadataField } from "@/features/nft/metadata";
import { shortHash } from "@/features/codepulse/format";

function etherscanRoot(chainId: number): string {
  if (chainId === 11_155_111) return "https://sepolia.etherscan.io";
  return "https://etherscan.io";
}

function explorerAddress(chainId: number, addr: string): string {
  return `${etherscanRoot(chainId)}/address/${addr}`;
}

function explorerTx(chainId: number, hash: string): string {
  return `${etherscanRoot(chainId)}/tx/${hash}`;
}

function explorerTokenContract(chainId: number, contract: string): string {
  return `${etherscanRoot(chainId)}/token/${contract}`;
}

function explorerAddressNftTransfers(chainId: number, addr: string): string {
  return `${etherscanRoot(chainId)}/address/${addr}#nfttransfers`;
}

const pageSize = 30;
const META_CONCURRENCY = 4;

/** 元数据多为 36×36：用 2× 整数倍展示 + pixelated，居中不糊 */
const NFT_THUMB_PX = 72;

function readContractError(err: unknown): string {
  if (err && typeof err === "object") {
    const o = err as { shortMessage?: string; message?: string };
    if (typeof o.shortMessage === "string" && o.shortMessage) return o.shortMessage;
    if (typeof o.message === "string" && o.message) return o.message;
  }
  if (err instanceof Error) return err.message;
  return String(err);
}

function holdingKey(h: NftHoldingRow): string {
  return `${h.collection_contract_address.toLowerCase()}-${h.token_id}`;
}

type ChainVerifyState =
  | { status: "match"; chainOwner: string }
  | { status: "mismatch"; chainOwner: string }
  | { status: "error"; detail: string };

type MediaState =
  | { status: "loading" }
  | { status: "ready"; imageUrl: string; displayName?: string }
  | { status: "empty"; detail?: string }
  | { status: "error"; detail: string };

type HoldingThumbProps = {
  loadImages: boolean;
  canReadChain: boolean;
  media: MediaState | undefined;
  rowKey: string;
  onImageError: (rowKey: string) => void;
};

const thumbFrame = `shrink-0 overflow-hidden rounded-md [width:${NFT_THUMB_PX}px] [height:${NFT_THUMB_PX}px]`;

const thumbChecker =
  "bg-[repeating-conic-gradient(#e7e5e4_0%_25%,#d6d3d1_0%_50%)_50%_/_10px_10px] dark:bg-[repeating-conic-gradient(#292524_0%_25%,#44403c_0%_50%)_50%_/_8px_8px]";

/** 36×36 素材 → 2× 格内居中，object-contain + pixelated */
function HoldingThumb({ loadImages, canReadChain, media, rowKey, onImageError }: HoldingThumbProps) {
  if (!loadImages) {
    return (
      <div
        className={`flex ${thumbFrame} items-center justify-center bg-muted/35 text-[10px] leading-tight text-muted-foreground`}
      >
        关
      </div>
    );
  }
  if (!canReadChain) {
    return (
      <div
        className={`flex ${thumbFrame} items-center justify-center bg-muted/35 px-1 text-center text-[9px] leading-snug text-muted-foreground`}
      >
        切 Sepolia
      </div>
    );
  }
  if (media?.status === "ready") {
    return (
      <img
        src={media.imageUrl}
        alt=""
        width={NFT_THUMB_PX}
        height={NFT_THUMB_PX}
        className={`${thumbFrame} ${thumbChecker} object-contain [image-rendering:pixelated]`}
        loading="lazy"
        onError={() => onImageError(rowKey)}
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

export function NftMyHoldingsPage() {
  const { address, isConnected } = useAccount();
  const walletChainId = useChainId();
  const publicClient = usePublicClient({ chainId: sepolia.id });

  const [ownerInput, setOwnerInput] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [data, setData] = useState<NftHoldingsResponse | null>(null);

  const [verifyOnChain, setVerifyOnChain] = useState(true);
  const [loadImages, setLoadImages] = useState(true);
  const [verifyBusy, setVerifyBusy] = useState(false);
  const [chainVerify, setChainVerify] = useState<Record<string, ChainVerifyState>>({});
  const [mediaByKey, setMediaByKey] = useState<Record<string, MediaState>>({});
  const [mediaFatal, setMediaFatal] = useState<string | null>(null);

  const prevConnectedRef = useRef<string | undefined>(undefined);

  /**
   * 切换钱包账户时：输入框若为空、或仍等于「上一连接的链上地址」，则跟到新地址并回到第 1 页。
   * 断开连接时不清空 ref，以便重连后仍能识别旧地址并替换为新钱包。
   */
  useEffect(() => {
    if (!address) {
      return;
    }
    const prev = prevConnectedRef.current;
    prevConnectedRef.current = address;

    let shouldResetPage = false;
    setOwnerInput((curr) => {
      const t = curr.trim();
      if (!t) {
        shouldResetPage = true;
        return address;
      }
      if (
        prev !== undefined &&
        isAddress(t) &&
        getAddress(t as Address).toLowerCase() === prev.toLowerCase()
      ) {
        shouldResetPage = true;
        return address;
      }
      return curr;
    });
    if (shouldResetPage) setPage(1);
  }, [address]);

  const queryOwner = useMemo(() => ownerInput.trim(), [ownerInput]);
  const queryOwnerValid = isAddress(queryOwner);
  const queryOwnerNorm = queryOwnerValid ? getAddress(queryOwner as Address).toLowerCase() : "";

  const load = useCallback(async () => {
    if (!queryOwnerValid) {
      setData(null);
      setErr(null);
      return;
    }
    setLoading(true);
    setErr(null);
    try {
      const r = await fetchNftHoldings(queryOwner, { page, page_size: pageSize });
      setData(r);
    } catch (e) {
      setData(null);
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [queryOwner, queryOwnerValid, page]);

  useEffect(() => {
    void load();
  }, [load]);

  const chainId = data?.chain_id ?? walletChainId;
  const chainMismatch =
    data?.chain_id != null && walletChainId > 0 && walletChainId !== data.chain_id;

  const holdings: NftHoldingRow[] = data?.holdings ?? [];
  const total = data?.total ?? 0;

  const canReadChain = Boolean(publicClient) && walletChainId === chainId && chainId === sepolia.id;

  useEffect(() => {
    setChainVerify({});
    if (!verifyOnChain || holdings.length === 0 || !publicClient || !canReadChain) {
      setVerifyBusy(false);
      return;
    }

    let cancelled = false;
    (async () => {
      setVerifyBusy(true);
      try {
        const contracts = holdings.map((h) => ({
          address: h.collection_contract_address as Address,
          abi: nftTemplateAbi,
          functionName: "ownerOf" as const,
          args: [BigInt(h.token_id)] as const,
        }));
        const results = await publicClient.multicall({ contracts, allowFailure: true });
        if (cancelled) return;
        const next: Record<string, ChainVerifyState> = {};
        holdings.forEach((h, i) => {
          const key = holdingKey(h);
          const r = results[i];
          if (r.status === "success") {
            const oc = getAddress(r.result as Address).toLowerCase();
            next[key] =
              oc === queryOwnerNorm
                ? { status: "match", chainOwner: r.result as string }
                : { status: "mismatch", chainOwner: r.result as string };
          } else {
            next[key] = {
              status: "error",
              detail: r.error ? readContractError(r.error) : "ownerOf 调用失败",
            };
          }
        });
        setChainVerify(next);
      } catch (e) {
        if (!cancelled) {
          const msg = e instanceof Error ? e.message : String(e);
          setChainVerify(
            Object.fromEntries(holdings.map((h) => [holdingKey(h), { status: "error" as const, detail: msg }])),
          );
        }
      } finally {
        if (!cancelled) setVerifyBusy(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [verifyOnChain, holdings, publicClient, canReadChain, queryOwnerNorm]);

  useEffect(() => {
    setMediaFatal(null);
    setMediaByKey({});
    if (!loadImages || holdings.length === 0 || !publicClient || !canReadChain) {
      return;
    }

    const ac = new AbortController();
    const keys = holdings.map(holdingKey);
    setMediaByKey(Object.fromEntries(keys.map((k) => [k, { status: "loading" as const }])));

    (async () => {
      try {
        const uriResults = await publicClient.multicall({
          contracts: holdings.map((h) => ({
            address: h.collection_contract_address as Address,
            abi: nftTemplateAbi,
            functionName: "tokenURI" as const,
            args: [BigInt(h.token_id)] as const,
          })),
          allowFailure: true,
        });
        if (ac.signal.aborted) return;

        for (let i = 0; i < holdings.length; i += META_CONCURRENCY) {
          if (ac.signal.aborted) return;
          const batch = holdings.slice(i, i + META_CONCURRENCY);
          await Promise.all(
            batch.map(async (h, j) => {
              const ii = i + j;
              const key = holdingKey(h);
              const ur = uriResults[ii];
              if (ur.status !== "success" || !ur.result) {
                setMediaByKey((prev) => ({
                  ...prev,
                  [key]: {
                    status: "error",
                    detail: ur.status === "failure" ? readContractError(ur.error) : "无 tokenURI",
                  },
                }));
                return;
              }
              const tokenURI = ur.result as string;
              try {
                const meta = await fetchMetadataFromTokenUri(tokenURI);
                if (ac.signal.aborted) return;
                if (!meta.image?.trim()) {
                  setMediaByKey((prev) => ({
                    ...prev,
                    [key]: { status: "empty", detail: "元数据中无 image 字段" },
                  }));
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
  }, [loadImages, holdings, publicClient, canReadChain]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <Link
          to="/nft"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          ← 返回概览
        </Link>
      </div>

      {chainMismatch ? (
        <Alert>
          <AlertTitle>钱包网络与后端链不一致</AlertTitle>
          <AlertDescription>
            当前钱包 chainId={walletChainId}， holdings 接口按后端 RPC 解析为 chain_id={data?.chain_id}。列表以后端已索引数据为准。
          </AlertDescription>
        </Alert>
      ) : null}

      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">我的 NFT（库内索引）</CardTitle>
          <CardDescription className="space-y-2 text-[13px] leading-relaxed">
            <p>
              数据来自 <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/nft/holdings?owner=…</code>
              ，与合集详情里的 Token 表同源：仅包含<strong className="text-foreground/90">扫块已写入 PostgreSQL</strong>的记录。
            </p>
            <p className="text-muted-foreground">
              开启「链上校验」后，会对每条记录在本页用 RPC 调用 <code className="rounded bg-muted px-1 font-mono text-[11px]">ownerOf(tokenId)</code>
              与当前查询地址比对。开启「加载图片」会再读 <code className="rounded bg-muted px-1 font-mono text-[11px]">tokenURI</code> 并拉取元数据（IPFS 网关可能被浏览器策略拦截，失败时可看 Etherscan）。
            </p>
            <p className="text-muted-foreground">
              缩略图在卡片内<strong className="text-foreground/80"> 2×（72px）居中</strong>展示，配合像素风格渲染，小图更清晰；原图仍为 36×36 时细节有限，可点 Etherscan 查看链上媒体。
            </p>
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {!isConnected ? (
            <Alert>
              <AlertTitle>未连接钱包</AlertTitle>
              <AlertDescription>连接后会自动填入你的地址；也可在下方手动输入任意 0x 地址查询（只读）。</AlertDescription>
            </Alert>
          ) : null}

          <div className="flex flex-col gap-3 rounded-lg border border-border/60 bg-muted/20 p-3 sm:flex-row sm:flex-wrap sm:items-center">
            <label className="flex cursor-pointer items-center gap-2 text-sm">
              <Checkbox checked={verifyOnChain} onCheckedChange={(c) => setVerifyOnChain(c === true)} />
              <span>链上校验 ownerOf</span>
            </label>
            <label className="flex cursor-pointer items-center gap-2 text-sm">
              <Checkbox checked={loadImages} onCheckedChange={(c) => setLoadImages(c === true)} />
              <span>加载元数据图片</span>
            </label>
            {verifyOnChain && verifyBusy ? (
              <span className="text-xs text-muted-foreground">正在 multicall 校验…</span>
            ) : null}
          </div>

          {!canReadChain && holdings.length > 0 && (verifyOnChain || loadImages) ? (
            <Alert>
              <AlertTitle>链上读取不可用</AlertTitle>
              <AlertDescription>
                请在钱包中切换到 <strong className="text-foreground/90">Sepolia（chainId {sepolia.id}）</strong>
                ，与后端索引链一致后，方可进行 ownerOf / tokenURI 与图片加载。
              </AlertDescription>
            </Alert>
          ) : null}

          <div className="space-y-1.5">
            <label htmlFor="nft-holdings-owner" className="text-xs font-medium text-muted-foreground">
              持有人地址（owner）
            </label>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <Input
                id="nft-holdings-owner"
                placeholder="0x…"
                value={ownerInput}
                onChange={(e) => {
                  setOwnerInput(e.target.value);
                  setPage(1);
                }}
                className="font-mono text-xs"
                spellCheck={false}
              />
              <Button type="button" variant="outline" size="sm" disabled={loading} onClick={() => void load()}>
                {loading ? "加载中…" : "刷新"}
              </Button>
            </div>
            {!queryOwnerValid && queryOwner.length > 0 ? (
              <p className="text-xs text-destructive">地址格式不正确。</p>
            ) : null}
          </div>

          {queryOwnerValid ? (
            <p className="text-xs text-muted-foreground">
              <a
                href={explorerAddress(chainId, queryOwner)}
                target="_blank"
                rel="noreferrer"
                className="text-primary underline-offset-4 hover:underline"
              >
                Etherscan 地址
              </a>
              <span className="mx-2">·</span>
              <a
                href={explorerAddressNftTransfers(chainId, queryOwner)}
                target="_blank"
                rel="noreferrer"
                className="text-primary underline-offset-4 hover:underline"
              >
                NFT 转账记录
              </a>
            </p>
          ) : null}

          {err ? (
            <Alert variant="destructive">
              <AlertTitle>加载失败</AlertTitle>
              <AlertDescription>{err}</AlertDescription>
            </Alert>
          ) : null}

          {mediaFatal ? (
            <Alert variant="destructive">
              <AlertTitle>图片批量加载失败</AlertTitle>
              <AlertDescription>{mediaFatal}</AlertDescription>
            </Alert>
          ) : null}

          {loading && !data ? <p className="text-sm text-muted-foreground">加载中…</p> : null}

          {!loading && queryOwnerValid && data && holdings.length === 0 ? (
            <p className="text-sm text-muted-foreground">该地址在本平台库中暂无已索引的 NFT 持有记录。</p>
          ) : null}

          {holdings.length > 0 ? (
            <>
              <p className="text-sm text-muted-foreground">
                共 {total} 条 · 第 {page} 页，每页 {pageSize}
              </p>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {holdings.map((h) => {
                  const key = holdingKey(h);
                  const v = chainVerify[key];
                  const m = mediaByKey[key];
                  const collName = h.collection_name?.trim() || `合集 #${h.collection_id}`;

                  return (
                    <Card
                      key={h.id}
                      className="group overflow-hidden rounded-xl border border-white/10 bg-gradient-to-b from-card/90 to-card/40 shadow-sm transition-[box-shadow,transform] duration-200 hover:-translate-y-0.5 hover:shadow-md hover:ring-1 hover:ring-primary/15"
                    >
                      <div className="flex flex-col gap-0">
                        <div className="flex justify-center bg-gradient-to-b from-muted/35 via-muted/20 to-transparent px-3 pb-2 pt-3">
                          <div className="rounded-xl border border-border/60 bg-background/40 p-1.5 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.04)] ring-1 ring-black/5 dark:ring-white/5">
                            <HoldingThumb
                              loadImages={loadImages}
                              canReadChain={canReadChain}
                              media={m}
                              rowKey={key}
                              onImageError={(rk) =>
                                setMediaByKey((prev) => ({
                                  ...prev,
                                  [rk]: { status: "error", detail: "图片 URL 无法显示（跨域或资源失效）" },
                                }))
                              }
                            />
                          </div>
                        </div>
                        <div className="space-y-2 border-t border-border/50 px-3 pb-3 pt-2.5">
                          <div>
                            <Link
                              to={`/nft/collections/${h.collection_id}`}
                              className="line-clamp-2 text-sm font-semibold leading-snug tracking-tight text-primary underline-offset-2 transition-colors hover:text-primary/90"
                            >
                              {collName}
                            </Link>
                            {m?.status === "ready" && m.displayName ? (
                              <p className="mt-0.5 line-clamp-2 text-xs leading-snug text-muted-foreground">{m.displayName}</p>
                            ) : null}
                          </div>
                          <p className="font-mono text-[11px] leading-relaxed text-muted-foreground">
                            <span className="text-foreground/70">#{h.token_id}</span>
                            <span className="mx-1 text-border">·</span>
                            <a
                              href={explorerTokenContract(chainId, h.collection_contract_address)}
                              target="_blank"
                              rel="noreferrer"
                              className="text-primary underline-offset-2 hover:underline"
                              title={h.collection_contract_address}
                            >
                              {shortHash(h.collection_contract_address, 6, 4)}
                            </a>
                          </p>
                          {verifyOnChain && canReadChain ? (
                            <div className="flex flex-wrap items-center gap-1.5">
                              {v?.status === "match" ? (
                                <Badge variant="secondary" className="border border-primary/10 bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
                                  链上一致
                                </Badge>
                              ) : null}
                              {v?.status === "mismatch" ? (
                                <Badge variant="destructive" className="px-2 py-0.5 text-[10px] font-medium">
                                  owner 不一致
                                </Badge>
                              ) : null}
                              {v?.status === "error" ? (
                                <Badge variant="outline" className="max-w-full truncate px-2 py-0.5 text-[10px] text-destructive">
                                  校验失败
                                </Badge>
                              ) : null}
                              {v?.status === "mismatch" ? (
                                <span className="w-full break-all font-mono text-[10px] text-muted-foreground">
                                  链上 {shortHash(v.chainOwner, 6, 4)}
                                </span>
                              ) : null}
                              {v?.status === "error" ? (
                                <span className="line-clamp-2 break-words text-[10px] text-muted-foreground">{v.detail}</span>
                              ) : null}
                            </div>
                          ) : verifyOnChain && !canReadChain ? (
                            <span className="text-[10px] text-muted-foreground">连接 Sepolia 后可校验</span>
                          ) : null}
                          {h.mint_tx_hash ? (
                            <p className="font-mono text-[10px] text-muted-foreground">
                              <a
                                href={explorerTx(chainId, h.mint_tx_hash)}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary underline-offset-2 hover:underline"
                              >
                                mint {shortHash(h.mint_tx_hash)}
                              </a>
                            </p>
                          ) : null}
                        </div>
                      </div>
                    </Card>
                  );
                })}
              </div>
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
            </>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
