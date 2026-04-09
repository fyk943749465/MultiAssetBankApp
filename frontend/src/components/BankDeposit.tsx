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
import { btn, btnPrimary, input, sectionTitleAccent, surface, surfaceDanger, surfaceWarn } from "../ui/styles";

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
      <section className={surface}>
        <h2 className={sectionTitleAccent}>MultiAssetBank 存 / 取</h2>
        <p className="text-sm text-slate-400">请先连接钱包。</p>
      </section>
    );
  }

  if (!onSepolia) {
    return (
      <section className={surfaceWarn}>
        <h2 className="mb-2 text-[11px] font-semibold uppercase tracking-[0.2em] text-amber-400/90">
          MultiAssetBank 存 / 取
        </h2>
        <p className="text-sm text-amber-100/90">请切换到 Sepolia（{sepolia.id}）后再操作。</p>
      </section>
    );
  }

  return (
    <section className={surface}>
      <h2 className={sectionTitleAccent}>MultiAssetBank 存 / 取</h2>
      <p className="mb-4 font-mono text-xs text-slate-500">
        合约 <span className="text-emerald-300/90">{bank}</span>
      </p>
      <p className="mb-5 rounded-xl border border-slate-700/60 bg-slate-950/50 px-3.5 py-3 text-xs leading-relaxed text-slate-400">
        发起交易前请确认 MetaMask 网络为 <strong className="text-slate-200">Sepolia</strong>（链 ID {sepolia.id}）。
      </p>

      {withdrawDisabled && (
        <div className={`${surfaceDanger} mb-5`}>
          <p className="text-sm text-red-200/90">
            合约已开启 <span className="font-mono">withdrawPaused</span>，暂不可提现；存款仍可进行。
          </p>
        </div>
      )}

      <div className="mb-5 inline-flex rounded-xl border border-slate-700/80 bg-slate-950/40 p-1">
        <button
          type="button"
          onClick={() => {
            setFlow("deposit");
            setLog(null);
          }}
          className={
            flow === "deposit"
              ? "rounded-lg bg-emerald-600/90 px-4 py-2 text-sm font-semibold text-white shadow-md"
              : "rounded-lg px-4 py-2 text-sm font-medium text-slate-400 transition hover:text-slate-200"
          }
        >
          存款
        </button>
        <button
          type="button"
          onClick={() => {
            setFlow("withdraw");
            setLog(null);
          }}
          className={
            flow === "withdraw"
              ? "rounded-lg bg-sky-600/90 px-4 py-2 text-sm font-semibold text-white shadow-md"
              : "rounded-lg px-4 py-2 text-sm font-medium text-slate-400 transition hover:text-slate-200"
          }
        >
          提现
        </button>
      </div>

      <div className="mb-5 inline-flex rounded-xl border border-slate-700/80 bg-slate-950/40 p-1">
        <button
          type="button"
          onClick={() => setMode("eth")}
          className={
            mode === "eth"
              ? "rounded-lg bg-slate-600/90 px-4 py-2 text-sm font-semibold text-white shadow-md"
              : "rounded-lg px-4 py-2 text-sm font-medium text-slate-400 transition hover:text-slate-200"
          }
        >
          以太（ETH）
        </button>
        <button
          type="button"
          onClick={() => setMode("erc20")}
          className={
            mode === "erc20"
              ? "rounded-lg bg-slate-600/90 px-4 py-2 text-sm font-semibold text-white shadow-md"
              : "rounded-lg px-4 py-2 text-sm font-medium text-slate-400 transition hover:text-slate-200"
          }
        >
          ERC20
        </button>
      </div>

      {mode === "eth" && (
        <div className="space-y-4">
          <div className="grid gap-1.5 rounded-xl border border-slate-800/80 bg-slate-950/40 px-3 py-3 text-sm text-slate-400">
            <span>钱包 ETH：{walletEth ? `${formatUnits(walletEth.value, 18)} ETH` : "…"}</span>
            <span>
              合约内记账（ETH）：{bankEthBalance !== undefined ? `${formatUnits(bankEthBalance, 18)} ETH` : "…"}
            </span>
          </div>
          <label className="block text-xs font-medium text-slate-400">
            数量（ETH）
            <input value={ethAmount} onChange={(e) => setEthAmount(e.target.value)} className={input} />
          </label>
          {flow === "withdraw" && (
            <button type="button" onClick={fillMaxEth} className={btn} disabled={bankEthBalance === undefined}>
              填入全部可提（记账余额）
            </button>
          )}
          {flow === "deposit" ? (
            <button type="button" disabled={busy} onClick={() => void depositEth()} className={btnPrimary}>
              {busy ? "处理中…" : "depositETH 存入"}
            </button>
          ) : (
            <button
              type="button"
              disabled={busy || withdrawDisabled}
              onClick={() => void withdrawEth()}
              className="rounded-xl border border-sky-500/40 bg-gradient-to-b from-sky-500 to-sky-600 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-sky-950/30 transition hover:from-sky-400 hover:to-sky-500 disabled:cursor-not-allowed disabled:opacity-45"
            >
              {busy ? "处理中…" : "withdrawETH 提现"}
            </button>
          )}
        </div>
      )}

      {mode === "erc20" && (
        <div className="space-y-4">
          <label className="block text-xs font-medium text-slate-400">
            代币（10 种预设）
            <select
              value={tokenIdx}
              onChange={(e) => setTokenIdx(Number(e.target.value))}
              className={input}
            >
              {SEPOLIA_ERC20_PRESETS.map((t, i) => (
                <option key={t.address} value={i} className="bg-slate-900">
                  {t.symbol} — {t.note ?? t.address}
                </option>
              ))}
            </select>
          </label>
          <p className="text-xs text-slate-500">
            合约地址：<span className="font-mono text-slate-400">{token.address}</span>
          </p>
          <div className="grid gap-1.5 rounded-xl border border-slate-800/80 bg-slate-950/40 px-3 py-3 text-sm text-slate-400">
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
                <span className="text-slate-500">
                  当前授权：{formatUnits(allowance, decimals)} {needApprove ? "（将先 approve）" : ""}
                </span>
              )}
          </div>
          <label className="block text-xs font-medium text-slate-400">
            数量（{token.symbol}）
            <input value={erc20Amount} onChange={(e) => setErc20Amount(e.target.value)} className={input} />
          </label>
          {flow === "withdraw" && (
            <button
              type="button"
              onClick={fillMaxErc20}
              className={btn}
              disabled={bankTokenBal === undefined || decimals === undefined}
            >
              填入全部可提（记账余额）
            </button>
          )}
          {flow === "deposit" ? (
            <button
              type="button"
              disabled={busy || decimals === undefined || amountWei === undefined}
              onClick={() => void depositErc20()}
              className={btnPrimary}
            >
              {busy ? "处理中…" : needApprove ? "授权并 depositToken" : "depositToken"}
            </button>
          ) : (
            <button
              type="button"
              disabled={busy || withdrawDisabled || decimals === undefined || amountWei === undefined}
              onClick={() => void withdrawErc20()}
              className="rounded-xl border border-sky-500/40 bg-gradient-to-b from-sky-500 to-sky-600 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-sky-950/30 transition hover:from-sky-400 hover:to-sky-500 disabled:cursor-not-allowed disabled:opacity-45"
            >
              {busy ? "处理中…" : "withdrawToken 提现"}
            </button>
          )}
        </div>
      )}

      {log && (
        <p
          className={`mt-5 border-t border-slate-700/60 pt-4 text-sm ${log.startsWith("0x") || log.includes("成功") ? "text-emerald-400" : "text-red-400"}`}
        >
          {log}
        </p>
      )}
    </section>
  );
}
