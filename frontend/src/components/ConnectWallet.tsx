import { useAccount, useConnect, useDisconnect, useChainId, useSwitchChain } from "wagmi";
import type { Connector } from "wagmi";
import { sepolia } from "wagmi/chains";
import { btn, code, sectionTitleAccent, surface } from "../ui/styles";

const WALLET_BAR_HINT =
  "请用系统浏览器打开 http://localhost:5173 并安装 MetaMask 等扩展；IDE 内置预览通常没有 window.ethereum。链上读写可用 Wagmi 的 useReadContract / useWriteContract。";

const btnCompact =
  "rounded-lg border border-slate-600 bg-slate-800/90 px-2.5 py-1.5 text-xs font-medium text-slate-100 shadow-sm transition hover:border-slate-500 hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-45";

/** Wagmi v2 的 Connector 类型里不一定声明 `ready`；用 `!c.ready` 会把 undefined 当成不可用，导致全部按钮灰掉。 */
function connectorExplicitlyUnavailable(c: Connector): boolean {
  return "ready" in c && (c as Connector & { ready?: boolean }).ready === false;
}

function shortAddress(a: string): string {
  if (a.length < 12) return a;
  return `${a.slice(0, 6)}…${a.slice(-4)}`;
}

type ConnectWalletProps = {
  /** 顶栏紧凑模式：右上角条带，少占纵向空间 */
  readonly compact?: boolean;
};

export function ConnectWallet({ compact = false }: ConnectWalletProps) {
  const { address, isConnected, status } = useAccount();
  const chainId = useChainId();
  const { connect, connectors, isPending, error, reset } = useConnect();
  const { disconnect } = useDisconnect();
  const { switchChain, isPending: isSwitching } = useSwitchChain();

  const wrongNetwork = isConnected && chainId !== sepolia.id;

  if (compact) {
    return (
      <div
        className="flex max-w-full flex-col items-end gap-1"
        title={WALLET_BAR_HINT}
      >
        <section
          className="inline-flex max-w-full flex-wrap items-center justify-end gap-2 rounded-xl border border-slate-700/60 bg-slate-900/85 px-3 py-2 shadow-md shadow-black/20 backdrop-blur-md"
          aria-label="钱包 · Sepolia"
        >
          <div className="min-w-0 text-right">
            {status === "connecting" && <span className="text-xs text-slate-400">连接中…</span>}
            {status === "disconnected" && (
              <span className="text-xs text-slate-500">未连接</span>
            )}
            {isConnected && address && (
              <div className="flex flex-wrap items-center justify-end gap-x-2 gap-y-0.5">
                <span
                  className="font-mono text-xs font-medium text-emerald-300/95 sm:text-sm"
                  title={address}
                >
                  <span className="sm:hidden">{shortAddress(address)}</span>
                  <span className="hidden max-w-[200px] truncate sm:inline" title={address}>
                    {address}
                  </span>
                </span>
                <span
                  className={`rounded-md px-1.5 py-0.5 font-mono text-[10px] ${
                    wrongNetwork
                      ? "bg-amber-950/70 text-amber-200 ring-1 ring-amber-500/35"
                      : "bg-slate-800/90 text-slate-400 ring-1 ring-slate-600/60"
                  }`}
                >
                  {wrongNetwork ? "网络不对" : `Sepolia`}
                </span>
              </div>
            )}
          </div>

          <div className="flex flex-shrink-0 flex-wrap justify-end gap-1.5">
            {!isConnected &&
              connectors.map((c) => (
                <button
                  key={c.uid}
                  type="button"
                  disabled={isPending || connectorExplicitlyUnavailable(c)}
                  onClick={() => {
                    reset();
                    connect({ connector: c });
                  }}
                  className={btnCompact}
                >
                  {c.name}
                  {connectorExplicitlyUnavailable(c) ? " ×" : ""}
                </button>
              ))}
            {isConnected && wrongNetwork && (
              <button
                type="button"
                disabled={isSwitching}
                onClick={() => switchChain({ chainId: sepolia.id })}
                className="rounded-lg border border-amber-500/40 bg-amber-950/55 px-2.5 py-1.5 text-xs font-semibold text-amber-100 transition hover:bg-amber-900/50 disabled:opacity-45"
              >
                {isSwitching ? "…" : "切 Sepolia"}
              </button>
            )}
            {isConnected && (
              <button type="button" onClick={() => disconnect()} className={btnCompact}>
                断开
              </button>
            )}
          </div>
        </section>

        {error && (
          <p className="max-w-sm text-right text-xs text-red-400" role="alert">
            {error.message}
          </p>
        )}
      </div>
    );
  }

  return (
    <section className={surface}>
      <h2 className={sectionTitleAccent}>钱包 · Sepolia</h2>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          {status === "connecting" && <p className="text-sm text-slate-400">连接中…</p>}
          {status === "disconnected" && (
            <p className="text-sm text-slate-400">未连接。请使用浏览器扩展钱包或 WalletConnect。</p>
          )}
          {isConnected && address && (
            <div className="space-y-1.5">
              <p className="truncate font-mono text-sm font-medium text-emerald-300/95" title={address}>
                {address}
              </p>
              <p className="text-xs text-slate-500">
                链 ID: <span className="font-mono text-slate-400">{chainId}</span>
                {wrongNetwork && (
                  <span className="ml-2 font-medium text-amber-400">（需要切换到 Sepolia {sepolia.id}）</span>
                )}
              </p>
            </div>
          )}
        </div>

        <div className="flex flex-wrap gap-2">
          {!isConnected &&
            connectors.map((c) => (
              <button
                key={c.uid}
                type="button"
                disabled={isPending || connectorExplicitlyUnavailable(c)}
                onClick={() => {
                  reset();
                  connect({ connector: c });
                }}
                className={btn}
              >
                {c.name}
                {connectorExplicitlyUnavailable(c) ? "（不可用）" : ""}
              </button>
            ))}
          {isConnected && wrongNetwork && (
            <button
              type="button"
              disabled={isSwitching}
              onClick={() => switchChain({ chainId: sepolia.id })}
              className="rounded-xl border border-amber-500/40 bg-amber-950/50 px-3.5 py-2 text-sm font-semibold text-amber-100 shadow-md transition hover:bg-amber-900/50 disabled:opacity-45"
            >
              {isSwitching ? "切换中…" : "切换到 Sepolia"}
            </button>
          )}
          {isConnected && (
            <button type="button" onClick={() => disconnect()} className={btn}>
              断开
            </button>
          )}
        </div>
      </div>

      {error && (
        <p className="mt-4 text-sm text-red-400" role="alert">
          {error.message}
        </p>
      )}

      <p className="mt-5 border-t border-slate-700/60 pt-4 text-xs leading-relaxed text-slate-500">
        请用系统浏览器打开 <code className={code}>http://localhost:5173</code>，并安装 MetaMask 等扩展；IDE 内置预览往往没有<code className={code}>window.ethereum</code>。链上读写可使用 <code className={code}>useReadContract</code>
        {" / "}
        <code className={code}>useWriteContract</code> 等 Wagmi 接口。
      </p>
    </section>
  );
}
