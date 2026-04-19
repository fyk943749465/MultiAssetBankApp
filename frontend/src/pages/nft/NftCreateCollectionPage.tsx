import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  useAccount,
  useChainId,
  usePublicClient,
  useReadContract,
  useSwitchChain,
  useWriteContract,
} from "wagmi";
import { sepolia } from "wagmi/chains";
import type { Address } from "viem";
import { formatUnits, toHex } from "viem";
import { nftFactoryAbi } from "@/abi/nftFactory";
import { getNftFactoryAddress } from "@/config/nft";
import { cn } from "@/lib/utils";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

function explorerAddress(addr: string): string {
  return `https://sepolia.etherscan.io/address/${addr}`;
}

function explorerTx(hash: string): string {
  return `https://sepolia.etherscan.io/tx/${hash}`;
}

type DeployMode = "standard" | "deterministic";

/** 解析为 bytes32：必须恰好 32 字节（64 个十六进制字符），可带或不带 0x。 */
function parseBytes32Salt(raw: string): `0x${string}` | null {
  const t = raw.trim();
  if (!t) return null;
  const h = /^0x/i.test(t) ? t.replace(/^0X/i, "0x") : `0x${t}`;
  if (!/^0x[0-9a-fA-F]{64}$/.test(h)) return null;
  return h.toLowerCase() as `0x${string}`;
}

function randomBytes32Hex(): `0x${string}` {
  const b = new Uint8Array(32);
  crypto.getRandomValues(b);
  return toHex(b) as `0x${string}`;
}

export function NftCreateCollectionPage() {
  const factory = getNftFactoryAddress();
  const { address, isConnected } = useAccount();
  const chainId = useChainId();
  const publicClient = usePublicClient({ chainId: sepolia.id });
  const { switchChainAsync } = useSwitchChain();
  const { writeContractAsync } = useWriteContract();

  const onSepolia = chainId === sepolia.id;

  const { data: creationFee, refetch: refetchFee } = useReadContract({
    address: factory,
    abi: nftFactoryAbi,
    functionName: "creationFee",
    query: { enabled: onSepolia },
  });

  const { data: paused } = useReadContract({
    address: factory,
    abi: nftFactoryAbi,
    functionName: "paused",
    query: { enabled: onSepolia },
  });

  const [deployMode, setDeployMode] = useState<DeployMode>("standard");
  const [saltInput, setSaltInput] = useState("");
  const [name, setName] = useState("");
  const [symbol, setSymbol] = useState("");
  const [baseUri, setBaseUri] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [lastCollection, setLastCollection] = useState<Address | null>(null);
  const [lastTxHash, setLastTxHash] = useState<string | null>(null);

  const saltParsed = useMemo(() => parseBytes32Salt(saltInput), [saltInput]);

  const { data: predictedAddress } = useReadContract({
    address: factory,
    abi: nftFactoryAbi,
    functionName: "predictCloneAddress",
    args: saltParsed ? [saltParsed] : undefined,
    query: { enabled: onSepolia && deployMode === "deterministic" && saltParsed != null },
  });

  const feeEth = useMemo(() => {
    if (creationFee === undefined) return null;
    try {
      return formatUnits(creationFee, 18);
    } catch {
      return null;
    }
  }, [creationFee]);

  const baseUriTrimmed = baseUri.trim();
  const trailingOk = baseUriTrimmed === "" || baseUriTrimmed.endsWith("/");

  async function ensureSepolia() {
    if (switchChainAsync && chainId !== sepolia.id) {
      await switchChainAsync({ chainId: sepolia.id });
    }
  }

  async function handleDeploy() {
    setMessage(null);
    setLastCollection(null);
    setLastTxHash(null);
    if (!address || !publicClient) {
      setMessage("请先连接钱包（右上角）。");
      return;
    }
    const n = name.trim();
    const s = symbol.trim();
    const b = baseUriTrimmed;
    if (!n || !s || !b) {
      setMessage("请填写合集名称、简称（symbol）和元数据根地址（base URI）。");
      return;
    }
    if (paused === true) {
      setMessage("工厂当前处于暂停状态，无法创建合集。");
      return;
    }
    if (creationFee === undefined) {
      setMessage("无法读取创建费，请确认网络为 Sepolia 后重试。");
      return;
    }
    if (deployMode === "deterministic" && !saltParsed) {
      setMessage(
        "确定性部署需要 Salt：请输入 64 位十六进制（32 字节），可带或不带 0x；或点「生成随机 Salt」。"
      );
      return;
    }

    setBusy(true);
    try {
      await ensureSepolia();
      let result: Address;
      let hash: `0x${string}`;
      if (deployMode === "deterministic" && saltParsed) {
        const { request, result: out } = await publicClient.simulateContract({
          address: factory,
          abi: nftFactoryAbi,
          functionName: "deployProxyDeterministic",
          args: [n, s, b, saltParsed],
          account: address,
          value: creationFee,
        });
        hash = await writeContractAsync(request);
        result = out as Address;
      } else {
        const { request, result: out } = await publicClient.simulateContract({
          address: factory,
          abi: nftFactoryAbi,
          functionName: "deployProxy",
          args: [n, s, b],
          account: address,
          value: creationFee,
        });
        hash = await writeContractAsync(request);
        result = out as Address;
      }
      const receipt = await publicClient.waitForTransactionReceipt({ hash });
      if (receipt.status !== "success") {
        setMessage(`交易已上链但未成功（status=${receipt.status}）：${hash}`);
        setLastTxHash(hash);
        return;
      }
      setLastTxHash(hash);
      setLastCollection(result as Address);
      setMessage(
        deployMode === "deterministic"
          ? "创建成功（确定性部署）！合集合约地址如下（应与部署前预览一致；已付创建费）。"
          : "创建成功！新合集合约地址如下（已付创建费）。"
      );
      await refetchFee();
    } catch (e) {
      setMessage(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!isConnected) {
    return (
      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">创建 NFT 合集</CardTitle>
          <CardDescription>通过平台工厂一键部署你的 ERC721 克隆合约（Sepolia）。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          <p>请先在页面右上角连接钱包，并切换到 Sepolia 网络。</p>
          <Link
            to="/nft"
            className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
          >
            返回概览
          </Link>
        </CardContent>
      </Card>
    );
  }

  if (!onSepolia) {
    return (
      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">创建 NFT 合集</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert>
            <AlertTitle>请切换到 Sepolia</AlertTitle>
            <AlertDescription>本功能仅在链 ID {sepolia.id} 上可用。</AlertDescription>
          </Alert>
          <Button
            onClick={() => {
              void switchChainAsync?.({ chainId: sepolia.id });
            }}
          >
            切换到 Sepolia
          </Button>
          <Link
            to="/nft"
            className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
          >
            返回概览
          </Link>
        </CardContent>
      </Card>
    );
  }

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

      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">创建 NFT 合集</CardTitle>
          <CardDescription>
            在链上调用工厂：可选 <code className="rounded bg-muted px-1 font-mono text-xs">deployProxy</code>（普通）
            或 <code className="rounded bg-muted px-1 font-mono text-xs">deployProxyDeterministic</code>
            （确定性，需 Salt）。均需支付一次创建费；无需编写 Solidity。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <p className="font-mono text-xs text-muted-foreground">
            工厂地址 <span className="text-primary">{factory}</span>
          </p>

          {paused === true ? (
            <Alert variant="destructive">
              <AlertTitle>工厂已暂停</AlertTitle>
              <AlertDescription>暂时无法创建新合集，请稍后再试或联系平台。</AlertDescription>
            </Alert>
          ) : null}

          <div className="rounded-lg border border-primary/20 bg-primary/5 px-4 py-3 text-sm leading-relaxed">
            <p className="font-medium text-foreground">创建费（将从钱包扣除 + gas）</p>
            <p className="mt-1 text-muted-foreground">
              {feeEth != null ? (
                <>
                  当前 <span className="font-mono text-foreground">{feeEth} ETH</span>
                </>
              ) : (
                "读取中…"
              )}
            </p>
          </div>

          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">部署方式</p>
            <div className="inline-flex flex-wrap rounded-lg border bg-muted/30 p-1">
              <button
                type="button"
                onClick={() => setDeployMode("standard")}
                className={cn(
                  "rounded-md px-4 py-2 text-sm font-medium transition",
                  deployMode === "standard"
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                普通（链上随机地址）
              </button>
              <button
                type="button"
                onClick={() => setDeployMode("deterministic")}
                className={cn(
                  "rounded-md px-4 py-2 text-sm font-medium transition",
                  deployMode === "deterministic"
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                确定性（Salt → 可预知地址）
              </button>
            </div>
            {deployMode === "deterministic" ? (
              <p className="max-w-2xl text-xs leading-relaxed text-muted-foreground">
                名称、符号、元数据根与 Salt 一致时，合集合约地址在部署前即可算出（与链上{" "}
                <code className="rounded bg-muted px-1 font-mono text-[11px]">predictCloneAddress</code> 一致）。若该地址上已有合约代码，部署会失败，请更换
                Salt。
              </p>
            ) : null}
          </div>

          {deployMode === "deterministic" ? (
            <div className="space-y-3 rounded-lg border border-white/10 bg-muted/15 p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
                <div className="min-w-0 flex-1 space-y-1.5">
                  <label htmlFor="nft-salt" className="text-xs font-medium text-muted-foreground">
                    Salt（32 字节，十六进制）
                  </label>
                  <Input
                    id="nft-salt"
                    placeholder="0x + 64 位 hex，或 64 位 hex 无前缀"
                    value={saltInput}
                    onChange={(e) => setSaltInput(e.target.value)}
                    className="font-mono text-xs"
                    spellCheck={false}
                    autoComplete="off"
                  />
                </div>
                <Button type="button" variant="outline" size="sm" className="shrink-0" onClick={() => setSaltInput(randomBytes32Hex())}>
                  生成随机 Salt
                </Button>
              </div>
              {saltInput.trim() && !saltParsed ? (
                <p className="text-xs text-destructive">
                  Salt 格式不正确：需要恰好 32 字节（64 个十六进制字符），总长度含 0x 为 66 字符。
                </p>
              ) : null}
              {saltParsed && predictedAddress ? (
                <div className="rounded-md border border-primary/20 bg-primary/5 px-3 py-2 text-sm leading-relaxed">
                  <span className="text-muted-foreground">部署前地址预览：</span>{" "}
                  <a
                    href={explorerAddress(predictedAddress)}
                    target="_blank"
                    rel="noreferrer"
                    className="break-all font-mono text-primary underline-offset-4 hover:underline"
                  >
                    {predictedAddress}
                  </a>
                </div>
              ) : saltParsed ? (
                <p className="text-xs text-muted-foreground">正在读取预测地址…</p>
              ) : null}
            </div>
          ) : null}

          <details className="rounded-lg border bg-muted/20 px-4 py-3 text-sm">
            <summary className="cursor-pointer font-medium text-foreground">什么是「元数据根地址」？（点开阅读）</summary>
            <ul className="mt-3 list-inside list-disc space-y-2 text-muted-foreground">
              <li>这是网络上存放每条 NFT 的 JSON 元数据的位置，不是单张图片链接。</li>
              <li>
                常见做法：把 <code className="rounded bg-muted px-1 text-xs">1</code>、<code className="rounded bg-muted px-1 text-xs">2</code>…
                等无后缀 JSON 文件上传到 Arweave / Irys 等，把上传后得到的<strong>根 URL</strong>填在这里。
              </li>
              <li>
                多数实现会把 <code className="rounded bg-muted px-1 text-xs">baseURI</code> 与编号直接拼接，因此根地址通常需要以{" "}
                <strong className="text-foreground">斜杠 / 结尾</strong>，例如{" "}
                <code className="break-all rounded bg-muted px-1 text-xs">https://gateway.irys.xyz/你的根ID/</code>
              </li>
              <li>更详细的说明见仓库内 `script/README.md` 中「合约 baseURI 用哪个 URL」一节。</li>
            </ul>
          </details>

          {!trailingOk && baseUriTrimmed ? (
            <Alert>
              <AlertTitle>建议检查末尾斜杠</AlertTitle>
              <AlertDescription>
                你填写的地址未以 <code className="font-mono">/</code> 结尾，可能导致{" "}
                <code className="font-mono">tokenURI</code> 拼错。若不确定，请按上传平台说明在末尾加上{" "}
                <code className="font-mono">/</code>。
              </AlertDescription>
            </Alert>
          ) : null}

          <div className="grid gap-4 sm:grid-cols-1">
            <div className="space-y-1.5">
              <label htmlFor="nft-create-name" className="text-xs font-medium text-muted-foreground">
                合集名称
              </label>
              <Input
                id="nft-create-name"
                placeholder="例如：我的像素小怪兽"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete="off"
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="nft-create-symbol" className="text-xs font-medium text-muted-foreground">
                简称（Symbol）
              </label>
              <Input
                id="nft-create-symbol"
                placeholder="例如：MONS（建议全大写、短一些）"
                value={symbol}
                onChange={(e) => setSymbol(e.target.value)}
                autoComplete="off"
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="nft-create-baseuri" className="text-xs font-medium text-muted-foreground">
                元数据根地址（Base URI）
              </label>
              <Textarea
                id="nft-create-baseuri"
                placeholder="https://gateway.irys.xyz/你的元数据根ID/"
                value={baseUri}
                onChange={(e) => setBaseUri(e.target.value)}
                rows={3}
                className="font-mono text-xs sm:text-sm"
              />
            </div>
          </div>

          <Button
            size="lg"
            className="w-full sm:w-auto"
            disabled={
              busy ||
              paused === true ||
              creationFee === undefined ||
              (deployMode === "deterministic" && !saltParsed)
            }
            onClick={() => void handleDeploy()}
          >
            {busy
              ? "钱包确认中…"
              : deployMode === "deterministic"
                ? "在钱包中确认并确定性创建"
                : "在钱包中确认并创建合集"}
          </Button>

          {message ? (
            <Alert variant={lastCollection ? "default" : "destructive"}>
              <AlertTitle>{lastCollection ? "完成" : "提示"}</AlertTitle>
              <AlertDescription className="space-y-3">
                <p>{message}</p>
                {lastCollection ? (
                  <p>
                    新合集合约：{" "}
                    <a
                      href={explorerAddress(lastCollection)}
                      target="_blank"
                      rel="noreferrer"
                      className="break-all font-mono text-primary underline-offset-4 hover:underline"
                    >
                      {lastCollection}
                    </a>
                  </p>
                ) : null}
                {lastTxHash ? (
                  <p>
                    交易：{" "}
                    <a
                      href={explorerTx(lastTxHash)}
                      target="_blank"
                      rel="noreferrer"
                      className="break-all font-mono text-primary underline-offset-4 hover:underline"
                    >
                      {lastTxHash}
                    </a>
                  </p>
                ) : null}
                {lastCollection ? (
                  <p className="text-xs text-muted-foreground">
                    网站「概览」里的列表来自数据库；新合集可能要等索引同步后才会出现，可先收藏上方合约地址。
                  </p>
                ) : null}
                {lastCollection ? (
                  <p>
                    <Link
                      to={`/nft/collections/${lastCollection}/mint`}
                      className="font-medium text-primary underline-offset-4 hover:underline"
                    >
                      前往铸造页（需已写入 PostgreSQL）
                    </Link>
                  </p>
                ) : null}
              </AlertDescription>
            </Alert>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
