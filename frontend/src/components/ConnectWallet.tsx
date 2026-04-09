import { useAccount, useConnect, useDisconnect, useChainId, useSwitchChain } from "wagmi";
import type { Connector } from "wagmi";
import { sepolia } from "wagmi/chains";
import { btn, code, sectionTitleAccent, surface } from "../ui/styles";

/** Wagmi v2 的 Connector 类型里不一定声明 `ready`；用 `!c.ready` 会把 undefined 当成不可用，导致全部按钮灰掉。 */
function connectorExplicitlyUnavailable(c: Connector): boolean {
  return "ready" in c && (c as Connector & { ready?: boolean }).ready === false;
}

export function ConnectWallet() {
  const { address, isConnected, status } = useAccount();
  const chainId = useChainId();
  const { connect, connectors, isPending, error, reset } = useConnect();
  const { disconnect } = useDisconnect();
  const { switchChain, isPending: isSwitching } = useSwitchChain();

  const wrongNetwork = isConnected && chainId !== sepolia.id;

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
        请用系统浏览器打开 <code className={code}>http://localhost:5173</code>
        ，并安装 MetaMask 等扩展；IDE 内置预览往往没有 <code className={code}>window.ethereum</code>
        。后续可用 <code className={code}>useReadContract</code> / <code className={code}>useWriteContract</code>
        ；业务仍走 <code className={code}>/api</code>。
      </p>
    </section>
  );
}
