import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  useAccount,
  useChainId,
  usePublicClient,
  useReadContract,
  useSwitchChain,
  useWriteContract,
} from "wagmi";
import { getAddress, isAddress } from "viem";
import type { Address } from "viem";
import { sepolia } from "wagmi/chains";
import { nftTemplateAbi } from "@/abi/nftTemplate";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { fetchNftCollectionByContractAddress, type NftCollectionDbRow } from "@/features/nft/api";
import { shortHash } from "@/features/codepulse/format";

function explorerAddress(chainId: number, addr: string): string {
  if (chainId === sepolia.id) return `https://sepolia.etherscan.io/address/${addr}`;
  return `https://etherscan.io/address/${addr}`;
}

function explorerTx(chainId: number, hash: string): string {
  if (chainId === sepolia.id) return `https://sepolia.etherscan.io/tx/${hash}`;
  return `https://etherscan.io/tx/${hash}`;
}

export function NftCollectionMintPage() {
  const { contractAddress: rawParam } = useParams<{ contractAddress: string }>();
  const contractParam = rawParam?.trim() ?? "";
  const contractValid = useMemo(() => isAddress(contractParam), [contractParam]);

  const { address, isConnected } = useAccount();
  const chainId = useChainId();
  const publicClient = usePublicClient({ chainId: sepolia.id });
  const { switchChainAsync } = useSwitchChain();
  const { writeContractAsync } = useWriteContract();

  const [collection, setCollection] = useState<NftCollectionDbRow | null>(null);
  const [loadErr, setLoadErr] = useState<string | null>(null);
  const [loadingDb, setLoadingDb] = useState(true);

  const [mintTo, setMintTo] = useState("");
  const [busy, setBusy] = useState(false);
  const [mintMsg, setMintMsg] = useState<string | null>(null);
  const [lastMintTx, setLastMintTx] = useState<string | null>(null);

  const prevMintWalletRef = useRef<string | undefined>(undefined);

  const collectionAddr = contractValid ? (contractParam as Address) : undefined;
  const onSepolia = chainId === sepolia.id;

  const loadCollection = useCallback(async () => {
    if (!contractValid) {
      setLoadingDb(false);
      setCollection(null);
      setLoadErr("无效的合集合约地址。");
      return;
    }
    setLoadingDb(true);
    setLoadErr(null);
    setCollection(null);
    try {
      const r = await fetchNftCollectionByContractAddress(contractParam);
      setCollection(r.collection);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (msg.includes("404")) {
        setLoadErr(
          "该地址在数据库中暂无合集记录。创建后需等待扫块同步入 PostgreSQL 后才可在此铸造；可稍后点「重新检测入库」。"
        );
      } else {
        setLoadErr(msg);
      }
    } finally {
      setLoadingDb(false);
    }
  }, [contractParam, contractValid]);

  useEffect(() => {
    void loadCollection();
  }, [loadCollection]);

  /** 切换账户：铸造给若为空或仍为上一连接地址则跟新钱包；手动填的第三方地址保留。 */
  useEffect(() => {
    if (!address) {
      return;
    }
    const prev = prevMintWalletRef.current;
    prevMintWalletRef.current = address;

    setMintTo((curr) => {
      const t = curr.trim();
      if (!t) return address;
      if (
        prev !== undefined &&
        isAddress(t) &&
        getAddress(t as Address).toLowerCase() === prev.toLowerCase()
      ) {
        return address;
      }
      return curr;
    });
  }, [address]);

  const {
    data: chainOwner,
    error: chainOwnerReadError,
    isPending: chainOwnerPending,
    isFetching: chainOwnerFetching,
  } = useReadContract({
    address: collectionAddr,
    abi: nftTemplateAbi,
    functionName: "owner",
    query: { enabled: Boolean(collectionAddr) && Boolean(collection) && onSepolia },
  });

  const { data: nextTokenId, refetch: refetchNext } = useReadContract({
    address: collectionAddr,
    abi: nftTemplateAbi,
    functionName: "nextTokenId",
    query: { enabled: Boolean(collectionAddr) && Boolean(collection) && onSepolia },
  });

  const { data: collectionName } = useReadContract({
    address: collectionAddr,
    abi: nftTemplateAbi,
    functionName: "name",
    query: { enabled: Boolean(collectionAddr) && Boolean(collection) && onSepolia },
  });

  const apiChainId = collection?.chain_id;
  const chainMismatch = apiChainId != null && chainId > 0 && chainId !== apiChainId;
  const isOwner =
    address && chainOwner != null
      ? address.toLowerCase() === (chainOwner as string).toLowerCase()
      : false;

  const chainOwnerLoading =
    onSepolia &&
    !chainOwnerReadError &&
    chainOwner === undefined &&
    (chainOwnerPending || chainOwnerFetching);

  const mintButtonDisabled =
    busy ||
    !address ||
    chainOwnerReadError != null ||
    chainOwnerLoading ||
    (chainOwner != null && !isOwner);

  const mintButtonLabel = (() => {
    if (busy) return "钱包确认中…";
    if (!address) return "请先连接钱包";
    if (chainOwnerReadError) return "无法读取链上 owner";
    if (chainOwnerLoading) return "正在读取链上 owner…";
    if (chainOwner != null && !isOwner) return "请使用 owner 钱包";
    return "发起 mint（在钱包中确认）";
  })();

  async function ensureSepolia() {
    if (switchChainAsync && chainId !== sepolia.id) {
      await switchChainAsync({ chainId: sepolia.id });
    }
  }

  async function handleMint() {
    setMintMsg(null);
    setLastMintTx(null);
    if (!collection || !collectionAddr || !address || !publicClient) {
      setMintMsg("请先连接钱包，并确认合集已入库。");
      return;
    }
    const to = mintTo.trim() as Address;
    if (!isAddress(to)) {
      setMintMsg("接收地址格式不正确。");
      return;
    }
    if (!isOwner) {
      setMintMsg("当前钱包不是链上 owner，合约通常会拒绝 mint。");
      return;
    }
    setBusy(true);
    try {
      await ensureSepolia();
      const { request } = await publicClient.simulateContract({
        address: collectionAddr,
        abi: nftTemplateAbi,
        functionName: "mint",
        args: [to],
        account: address,
      });
      const hash = await writeContractAsync(request);
      const receipt = await publicClient.waitForTransactionReceipt({ hash });
      if (receipt.status !== "success") {
        setMintMsg(`交易已上链但未成功（status=${receipt.status}）`);
        setLastMintTx(hash);
        return;
      }
      setLastMintTx(hash);
      setMintMsg("铸造成功。索引延迟时，库内 Token 列表可能稍后才更新。");
      await refetchNext();
    } catch (e) {
      setMintMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!contractValid) {
    return (
      <Alert variant="destructive">
        <AlertTitle>地址无效</AlertTitle>
        <AlertDescription>路径中的合集合约地址不是合法的 0x 地址。</AlertDescription>
      </Alert>
    );
  }

  if (loadingDb) {
    return <p className="text-sm text-muted-foreground">正在校验数据库中的合集…</p>;
  }

  if (loadErr || !collection) {
    return (
      <div className="space-y-4">
        <div className="flex flex-wrap gap-2">
          <Link
            to="/nft"
            className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
          >
            ← 返回概览
          </Link>
          <Button type="button" variant="outline" size="sm" onClick={() => void loadCollection()}>
            重新检测入库
          </Button>
        </div>
        <Alert variant="destructive">
          <AlertTitle>无法铸造</AlertTitle>
          <AlertDescription>{loadErr ?? "未知错误"}</AlertDescription>
        </Alert>
      </div>
    );
  }

  const cid = collection.chain_id;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <Link
          to="/nft"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          ← 概览
        </Link>
        <Link
          to={`/nft/collections/${collection.id}`}
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          库内详情 #{collection.id}
        </Link>
      </div>

      {chainMismatch ? (
        <Alert>
          <AlertTitle>网络不一致</AlertTitle>
          <AlertDescription>
            合集 chain_id={cid}，当前钱包 chainId={chainId}。请切换到与后端一致的链（本环境一般为 Sepolia）。
          </AlertDescription>
        </Alert>
      ) : null}

      {!isConnected ? (
        <Alert>
          <AlertTitle>请连接钱包</AlertTitle>
          <AlertDescription>铸造需由合集 owner 在链上发起 <code className="font-mono text-xs">mint</code> 交易。</AlertDescription>
        </Alert>
      ) : null}

      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">铸造 NFT</CardTitle>
          <CardDescription className="space-y-2 text-[13px] leading-relaxed">
            <p>
              合集合约与工厂部署的克隆一致，使用 NFTTemplate ABI 调用 <code className="font-mono text-xs">mint(address to)</code>。
            </p>
            <p>
              <strong className="text-foreground/90">是否允许铸造只由后端查询 PostgreSQL 决定</strong>（子图不参与放行；子图可能因重组与链短暂不一致）。
              打开本页时已请求 <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/nft/collections/by-contract/…</code>
              ，库中无该合集合约则不会展示链上操作区。
            </p>
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <div>
            <span className="text-muted-foreground">合集合约 </span>
            <a
              href={explorerAddress(cid, collection.contract_address)}
              target="_blank"
              rel="noreferrer"
              className="break-all font-mono text-primary underline-offset-4 hover:underline"
            >
              {collection.contract_address}
            </a>
          </div>
          <div>
            <span className="text-muted-foreground">链上名称 </span>
            <span className="font-mono text-xs">{collectionName ?? "读取中…"}</span>
            {collection.collection_name ? (
              <span className="ml-2 text-muted-foreground">（库：{collection.collection_name}）</span>
            ) : null}
          </div>
          <div>
            <span className="text-muted-foreground">下一枚 tokenId（nextTokenId） </span>
            <span className="font-mono text-xs">{nextTokenId != null ? String(nextTokenId) : onSepolia ? "—" : "请切 Sepolia"}</span>
          </div>
          <div>
            <span className="text-muted-foreground">链上 owner </span>
            {chainOwner ? (
              <a
                href={explorerAddress(cid, chainOwner as string)}
                target="_blank"
                rel="noreferrer"
                className="font-mono text-xs text-primary underline-offset-4 hover:underline"
              >
                {shortHash(chainOwner as string, 8, 6)}
              </a>
            ) : (
              <span className="text-muted-foreground">{onSepolia ? "读取中…" : "请切换到 Sepolia"}</span>
            )}
          </div>
          {chainOwnerReadError ? (
            <Alert variant="destructive">
              <AlertTitle>读取 owner 失败</AlertTitle>
              <AlertDescription>
                {chainOwnerReadError instanceof Error
                  ? chainOwnerReadError.message
                  : String(chainOwnerReadError)}
              </AlertDescription>
            </Alert>
          ) : null}

          {address && chainOwner && !isOwner ? (
            <Alert variant="destructive">
              <AlertTitle>非 owner 钱包</AlertTitle>
              <AlertDescription>
                已连接 {shortHash(address, 8, 6)}，与 owner 不一致，mint 很可能失败。
              </AlertDescription>
            </Alert>
          ) : null}

          <div className="space-y-1.5">
            <label htmlFor="nft-mint-to" className="text-xs font-medium text-muted-foreground">
              铸造给（to）
            </label>
            <Input
              id="nft-mint-to"
              placeholder="0x…"
              value={mintTo}
              onChange={(e) => setMintTo(e.target.value)}
              className="font-mono text-xs"
              spellCheck={false}
            />
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              此处为接收 NFT 的地址；能否点击「发起 mint」取决于<strong className="text-foreground/80">当前已连接钱包是否为链上 owner</strong>
              ，与这里填写的地址无关。
            </p>
          </div>

          {!onSepolia ? (
            <Button type="button" onClick={() => void switchChainAsync?.({ chainId: sepolia.id })}>
              切换到 Sepolia
            </Button>
          ) : (
            <Button type="button" disabled={mintButtonDisabled} onClick={() => void handleMint()}>
              {mintButtonLabel}
            </Button>
          )}

          {mintMsg ? (
            <Alert variant={lastMintTx && mintMsg.includes("成功") ? "default" : "destructive"}>
              <AlertTitle>{lastMintTx && mintMsg.includes("成功") ? "完成" : "提示"}</AlertTitle>
              <AlertDescription className="space-y-2">
                <p>{mintMsg}</p>
                {lastMintTx ? (
                  <a
                    href={explorerTx(cid, lastMintTx)}
                    target="_blank"
                    rel="noreferrer"
                    className="break-all font-mono text-primary underline-offset-4 hover:underline"
                  >
                    {lastMintTx}
                  </a>
                ) : null}
              </AlertDescription>
            </Alert>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
