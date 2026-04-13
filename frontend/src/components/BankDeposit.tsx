import { useMemo, useState } from "react";
import {
  useAccount,
  useBalance,
  useChainId,
  usePublicClient,
  useReadContract,
  useSwitchChain,
  useWriteContract,
} from "wagmi";
import { sepolia } from "wagmi/chains";
import { formatUnits, parseEther, parseUnits } from "viem";
import { multiAssetBankAbi } from "../abi/multiAssetBank";
import { erc20Abi } from "../abi/erc20";
import { getBankAddress } from "../config/bank";
import { SEPOLIA_ERC20_PRESETS } from "../config/sepoliaErc20";
import { notifyBankLedgerRefresh } from "./BankLedgerHistory";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { cn } from "@/lib/utils";

type Flow = "deposit" | "withdraw";

export function BankDeposit() {
  const { address, isConnected } = useAccount();
  const chainId = useChainId();
  const publicClient = usePublicClient({ chainId: sepolia.id });
  const { switchChainAsync } = useSwitchChain();
  const { writeContractAsync } = useWriteContract();

  const bank = getBankAddress();

  const [flow, setFlow] = useState<Flow>("deposit");
  const [mode, setMode] = useState<"eth" | "erc20">("eth");
  const [ethAmount, setEthAmount] = useState("0.01");
  const [erc20Amount, setErc20Amount] = useState("1");
  const [tokenIdx, setTokenIdx] = useState(0);
  const [busy, setBusy] = useState(false);
  const [log, setLog] = useState<string | null>(null);

  const token = SEPOLIA_ERC20_PRESETS[tokenIdx]!;
  const onSepolia = chainId === sepolia.id;

  const { data: walletEth } = useBalance({
    address,
    query: { enabled: !!address && onSepolia },
  });

  const { data: ethSentinel } = useReadContract({
    address: bank,
    abi: multiAssetBankAbi,
    functionName: "ETH_ADDRESS",
    query: { enabled: isConnected && onSepolia },
  });

  const { data: withdrawPaused } = useReadContract({
    address: bank,
    abi: multiAssetBankAbi,
    functionName: "withdrawPaused",
    query: { enabled: isConnected && onSepolia },
  });

  const { data: bankEthBalance, refetch: refetchBankEth } = useReadContract({
    address: bank,
    abi: multiAssetBankAbi,
    functionName: "tokenBalances",
    args: ethSentinel && address ? [address, ethSentinel] : undefined,
    query: { enabled: !!ethSentinel && !!address && onSepolia },
  });

  const { data: decimals } = useReadContract({
    address: token.address,
    abi: erc20Abi,
    functionName: "decimals",
    query: { enabled: isConnected && onSepolia && mode === "erc20" },
  });

  const { data: walletTokenBal } = useReadContract({
    address: token.address,
    abi: erc20Abi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: !!address && onSepolia && mode === "erc20" },
  });

  const { data: allowance, refetch: refetchAllowance } = useReadContract({
    address: token.address,
    abi: erc20Abi,
    functionName: "allowance",
    args: address ? [address, bank] : undefined,
    query: { enabled: !!address && onSepolia && mode === "erc20" && flow === "deposit" },
  });

  const { data: bankTokenBal, refetch: refetchBankToken } = useReadContract({
    address: bank,
    abi: multiAssetBankAbi,
    functionName: "tokenBalances",
    args: address ? [address, token.address] : undefined,
    query: { enabled: !!address && onSepolia && mode === "erc20" },
  });

  const amountWei = useMemo(() => {
    if (mode !== "erc20" || decimals === undefined) return undefined;
    try {
      return parseUnits(erc20Amount, Number(decimals));
    } catch {
      return undefined;
    }
  }, [mode, erc20Amount, decimals]);

  const needApprove =
    flow === "deposit" &&
    mode === "erc20" &&
    amountWei !== undefined &&
    allowance !== undefined &&
    allowance < amountWei;

  const withdrawDisabled = flow === "withdraw" && withdrawPaused === true;

  async function ensureSepoliaInWallet() {
    if (switchChainAsync) {
      await switchChainAsync({ chainId: sepolia.id });
    }
  }

  function fillMaxEth() {
    if (bankEthBalance === undefined) return;
    setEthAmount(formatUnits(bankEthBalance, 18));
  }

  function fillMaxErc20() {
    if (bankTokenBal === undefined || decimals === undefined) return;
    setErc20Amount(formatUnits(bankTokenBal, decimals));
  }

  async function depositEth() {
    if (!address || !publicClient) return;
    setLog(null);
    setBusy(true);
    try {
      await ensureSepoliaInWallet();
      const value = parseEther(ethAmount);
      const { request } = await publicClient.simulateContract({
        address: bank,
        abi: multiAssetBankAbi,
        functionName: "depositETH",
        account: address,
        value,
      });
      const hash = await writeContractAsync(request);
      await publicClient.waitForTransactionReceipt({ hash });
      await refetchBankEth();
      setLog(`ETH 存入成功：${hash}`);
      notifyBankLedgerRefresh();
    } catch (e) {
      setLog(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function withdrawEth() {
    if (!address || !publicClient) return;
    setLog(null);
    setBusy(true);
    try {
      await ensureSepoliaInWallet();
      const value = parseEther(ethAmount);
      const { request } = await publicClient.simulateContract({
        address: bank,
        abi: multiAssetBankAbi,
        functionName: "withdrawETH",
        args: [value],
        account: address,
      });
      const hash = await writeContractAsync(request);
      await publicClient.waitForTransactionReceipt({ hash });
      await refetchBankEth();
      setLog(`ETH 提现成功：${hash}`);
      notifyBankLedgerRefresh();
    } catch (e) {
      setLog(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function depositErc20() {
    if (!address || !publicClient || amountWei === undefined) return;
    setLog(null);
    setBusy(true);
    try {
      await ensureSepoliaInWallet();
      if (allowance === undefined || allowance < amountWei) {
        const h1 = await writeContractAsync({
          address: token.address,
          abi: erc20Abi,
          functionName: "approve",
          args: [bank, amountWei],
        });
        await publicClient.waitForTransactionReceipt({ hash: h1 });
        await refetchAllowance();
      }
      const { request: depReq } = await publicClient.simulateContract({
        address: bank,
        abi: multiAssetBankAbi,
        functionName: "depositToken",
        args: [token.address, amountWei],
        account: address,
      });
      const h2 = await writeContractAsync(depReq);
      await publicClient.waitForTransactionReceipt({ hash: h2 });
      await refetchBankToken();
      setLog(`ERC20 存入成功：${h2}`);
      notifyBankLedgerRefresh();
    } catch (e) {
      setLog(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function withdrawErc20() {
    if (!address || !publicClient || amountWei === undefined) return;
    setLog(null);
    setBusy(true);
    try {
      await ensureSepoliaInWallet();
      const { request } = await publicClient.simulateContract({
        address: bank,
        abi: multiAssetBankAbi,
        functionName: "withdrawToken",
        args: [token.address, amountWei],
        account: address,
      });
      const hash = await writeContractAsync(request);
      await publicClient.waitForTransactionReceipt({ hash });
      await refetchBankToken();
      setLog(`ERC20 提现成功：${hash}`);
      notifyBankLedgerRefresh();
    } catch (e) {
      setLog(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!isConnected) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-primary">MultiAssetBank 存 / 取</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">请先连接钱包。</p>
        </CardContent>
      </Card>
    );
  }

  if (!onSepolia) {
    return (
      <Alert>
        <AlertDescription>
          <strong>MultiAssetBank 存 / 取</strong> — 请切换到 Sepolia（{sepolia.id}）后再操作。
        </AlertDescription>
      </Alert>
    );
  }

  const tabClass = (active: boolean) =>
    cn(
      "rounded-md px-4 py-2 text-sm font-medium transition",
      active ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
    );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-primary">MultiAssetBank 存 / 取</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="font-mono text-xs text-muted-foreground">
          合约 <span className="text-primary">{bank}</span>
        </p>
        <p className="rounded-lg border bg-muted/50 px-3.5 py-3 text-xs leading-relaxed text-muted-foreground">
          发起交易前请确认 MetaMask 网络为 <strong className="text-foreground">Sepolia</strong>（链 ID {sepolia.id}）。
        </p>

        {withdrawDisabled && (
          <Alert variant="destructive">
            <AlertDescription>
              合约已开启 <span className="font-mono">withdrawPaused</span>，暂不可提现；存款仍可进行。
            </AlertDescription>
          </Alert>
        )}

        <div className="inline-flex rounded-lg border bg-muted/30 p-1">
          <button type="button" onClick={() => { setFlow("deposit"); setLog(null); }} className={tabClass(flow === "deposit")}>
            存款
          </button>
          <button type="button" onClick={() => { setFlow("withdraw"); setLog(null); }} className={tabClass(flow === "withdraw")}>
            提现
          </button>
        </div>

        <div className="inline-flex rounded-lg border bg-muted/30 p-1">
          <button type="button" onClick={() => setMode("eth")} className={tabClass(mode === "eth")}>
            以太（ETH）
          </button>
          <button type="button" onClick={() => setMode("erc20")} className={tabClass(mode === "erc20")}>
            ERC20
          </button>
        </div>

        {mode === "eth" && (
          <div className="space-y-4">
            <div className="grid gap-1.5 rounded-lg border bg-muted/30 px-3 py-3 text-sm text-muted-foreground">
              <span>钱包 ETH：{walletEth ? `${formatUnits(walletEth.value, 18)} ETH` : "…"}</span>
              <span>
                合约内记账（ETH）：{bankEthBalance !== undefined ? `${formatUnits(bankEthBalance, 18)} ETH` : "…"}
              </span>
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">数量（ETH）</label>
              <Input value={ethAmount} onChange={(e) => setEthAmount(e.target.value)} />
            </div>
            {flow === "withdraw" && (
              <Button variant="outline" onClick={fillMaxEth} disabled={bankEthBalance === undefined}>
                填入全部可提（记账余额）
              </Button>
            )}
            {flow === "deposit" ? (
              <Button disabled={busy} onClick={() => void depositEth()}>
                {busy ? "处理中…" : "depositETH 存入"}
              </Button>
            ) : (
              <Button variant="secondary" disabled={busy || withdrawDisabled} onClick={() => void withdrawEth()}>
                {busy ? "处理中…" : "withdrawETH 提现"}
              </Button>
            )}
          </div>
        )}

        {mode === "erc20" && (
          <div className="space-y-4">
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">代币（10 种预设）</label>
              <select
                value={tokenIdx}
                onChange={(e) => setTokenIdx(Number(e.target.value))}
                className="mt-1 w-full rounded-lg border bg-background px-3.5 py-2.5 font-mono text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
              >
                {SEPOLIA_ERC20_PRESETS.map((t, i) => (
                  <option key={t.address} value={i}>
                    {t.symbol} — {t.note ?? t.address}
                  </option>
                ))}
              </select>
            </div>
            <p className="text-xs text-muted-foreground">
              合约地址：<span className="font-mono text-foreground">{token.address}</span>
            </p>
            <div className="grid gap-1.5 rounded-lg border bg-muted/30 px-3 py-3 text-sm text-muted-foreground">
              <span>
                钱包余额：
                {decimals !== undefined && walletTokenBal !== undefined
                  ? `${formatUnits(walletTokenBal, decimals)} ${token.symbol}`
                  : decimals === undefined
                    ? "无法读取 decimals"
                    : "…"}
              </span>
              <span>
                合约内记账：
                {decimals !== undefined && bankTokenBal !== undefined
                  ? `${formatUnits(bankTokenBal, decimals)} ${token.symbol}`
                  : "…"}
              </span>
              {flow === "deposit" &&
                allowance !== undefined &&
                amountWei !== undefined &&
                decimals !== undefined && (
                  <span>
                    当前授权：{formatUnits(allowance, decimals)} {needApprove ? "（将先 approve）" : ""}
                  </span>
                )}
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">数量（{token.symbol}）</label>
              <Input value={erc20Amount} onChange={(e) => setErc20Amount(e.target.value)} />
            </div>
            {flow === "withdraw" && (
              <Button
                variant="outline"
                onClick={fillMaxErc20}
                disabled={bankTokenBal === undefined || decimals === undefined}
              >
                填入全部可提（记账余额）
              </Button>
            )}
            {flow === "deposit" ? (
              <Button
                disabled={busy || decimals === undefined || amountWei === undefined}
                onClick={() => void depositErc20()}
              >
                {busy ? "处理中…" : needApprove ? "授权并 depositToken" : "depositToken"}
              </Button>
            ) : (
              <Button
                variant="secondary"
                disabled={busy || withdrawDisabled || decimals === undefined || amountWei === undefined}
                onClick={() => void withdrawErc20()}
              >
                {busy ? "处理中…" : "withdrawToken 提现"}
              </Button>
            )}
          </div>
        )}

        {log && (
          <p
            className={cn(
              "border-t pt-4 text-sm",
              log.startsWith("0x") || log.includes("成功") ? "text-primary" : "text-destructive"
            )}
          >
            {log}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
