import { useCallback, useEffect, useState } from "react";
import { useAccount, useConnect, useDisconnect, useChainId, useSwitchChain } from "wagmi";
import type { Connector } from "wagmi";
import { sepolia } from "wagmi/chains";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";

const WALLET_BAR_HINT =
  "请用系统浏览器打开 http://localhost:5173 并安装 MetaMask 等扩展；IDE 内置预览通常没有 window.ethereum。链上读写可用 Wagmi 的 useReadContract / useWriteContract。";

function connectorExplicitlyUnavailable(c: Connector): boolean {
  return "ready" in c && (c as Connector & { ready?: boolean }).ready === false;
}

function shortAddress(a: string): string {
  if (a.length < 12) return a;
  return `${a.slice(0, 6)}…${a.slice(-4)}`;
}

async function copyTextToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {
    /* fall through */
  }
  try {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.left = "-9999px";
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    const ok = document.execCommand("copy");
    ta.remove();
    return ok;
  } catch {
    return false;
  }
}

type ConnectWalletProps = {
  readonly compact?: boolean;
};

export function ConnectWallet({ compact = false }: ConnectWalletProps) {
  const { address, isConnected, status } = useAccount();
  const chainId = useChainId();
  const { connect, connectors, isPending, error, reset } = useConnect();
  const { disconnect } = useDisconnect();
  const { switchChain, isPending: isSwitching } = useSwitchChain();
  const [addressCopied, setAddressCopied] = useState(false);

  const copyAddress = useCallback(async () => {
    if (!address) return;
    const ok = await copyTextToClipboard(address);
    if (!ok) return;
    setAddressCopied(true);
    globalThis.setTimeout(() => setAddressCopied(false), 2000);
  }, [address]);

  useEffect(() => {
    if (!address) setAddressCopied(false);
  }, [address]);

  const wrongNetwork = isConnected && chainId !== sepolia.id;

  if (compact) {
    return (
      <div className="flex max-w-full flex-col items-end gap-1" title={WALLET_BAR_HINT}>
        <section
          className="inline-flex max-w-full flex-wrap items-center justify-end gap-2 rounded-xl border bg-card/80 px-3 py-2 shadow-lg shadow-black/5 backdrop-blur-md dark:shadow-black/20"
          aria-label="钱包 · Sepolia"
        >
          <div className="min-w-0 text-right">
            {status === "connecting" && (
              <span className="text-xs text-muted-foreground">连接中…</span>
            )}
            {status === "disconnected" && (
              <span className="text-xs text-muted-foreground">未连接</span>
            )}
            {isConnected && address && (
              <div className="flex flex-wrap items-center justify-end gap-x-2 gap-y-0.5">
                <button
                  type="button"
                  onClick={() => void copyAddress()}
                  className="group max-w-full rounded-md border border-transparent px-1 py-0.5 text-left font-mono text-xs font-medium text-primary transition-colors hover:border-primary/25 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 sm:text-sm"
                  title="点击复制完整地址"
                  aria-label={`复制地址 ${address}`}
                >
                  <span className="sm:hidden">
                    {addressCopied ? "已复制" : shortAddress(address)}
                  </span>
                  <span className="hidden max-w-[200px] truncate sm:inline">
                    {addressCopied ? "已复制" : address}
                  </span>
                </button>
                <Badge variant={wrongNetwork ? "destructive" : "secondary"}>
                  {wrongNetwork ? "网络不对" : "Sepolia"}
                </Badge>
              </div>
            )}
          </div>

          <div className="flex flex-shrink-0 flex-wrap justify-end gap-1.5">
            {!isConnected &&
              connectors.map((c) => (
                <Button
                  key={c.uid}
                  variant="outline"
                  size="xs"
                  disabled={isPending || connectorExplicitlyUnavailable(c)}
                  onClick={() => {
                    reset();
                    connect({ connector: c });
                  }}
                >
                  {c.name}
                  {connectorExplicitlyUnavailable(c) ? " ×" : ""}
                </Button>
              ))}
            {isConnected && wrongNetwork && (
              <Button
                variant="destructive"
                size="xs"
                disabled={isSwitching}
                onClick={() => switchChain({ chainId: sepolia.id })}
              >
                {isSwitching ? "…" : "切 Sepolia"}
              </Button>
            )}
            {isConnected && (
              <Button variant="outline" size="xs" onClick={() => disconnect()}>
                断开
              </Button>
            )}
          </div>
        </section>

        {error && (
          <p className="max-w-sm text-right text-xs text-destructive" role="alert">
            {error.message}
          </p>
        )}
      </div>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-primary">钱包 · Sepolia</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0">
            {status === "connecting" && (
              <p className="text-sm text-muted-foreground">连接中…</p>
            )}
            {status === "disconnected" && (
              <p className="text-sm text-muted-foreground">
                未连接。请使用浏览器扩展钱包或 WalletConnect。
              </p>
            )}
            {isConnected && address && (
              <div className="space-y-1.5">
                <button
                  type="button"
                  onClick={() => void copyAddress()}
                  className="w-full truncate rounded-md border border-transparent px-1 py-0.5 text-left font-mono text-sm font-medium text-primary transition-colors hover:border-primary/25 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                  title="点击复制完整地址"
                  aria-label={`复制地址 ${address}`}
                >
                  {addressCopied ? "已复制" : address}
                </button>
                <p className="text-xs text-muted-foreground">
                  链 ID: <span className="font-mono text-foreground">{chainId}</span>
                  {wrongNetwork && (
                    <span className="ml-2 font-medium text-warning">
                      （需要切换到 Sepolia {sepolia.id}）
                    </span>
                  )}
                </p>
              </div>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            {!isConnected &&
              connectors.map((c) => (
                <Button
                  key={c.uid}
                  variant="outline"
                  disabled={isPending || connectorExplicitlyUnavailable(c)}
                  onClick={() => {
                    reset();
                    connect({ connector: c });
                  }}
                >
                  {c.name}
                  {connectorExplicitlyUnavailable(c) ? "（不可用）" : ""}
                </Button>
              ))}
            {isConnected && wrongNetwork && (
              <Button
                variant="destructive"
                disabled={isSwitching}
                onClick={() => switchChain({ chainId: sepolia.id })}
              >
                {isSwitching ? "切换中…" : "切换到 Sepolia"}
              </Button>
            )}
            {isConnected && (
              <Button variant="outline" onClick={() => disconnect()}>
                断开
              </Button>
            )}
          </div>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        )}

        <CardDescription className="border-t pt-4 text-xs leading-relaxed">
          请用系统浏览器打开{" "}
          <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">
            http://localhost:5173
          </code>
          ，并安装 MetaMask 等扩展；IDE 内置预览往往没有
          <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">
            window.ethereum
          </code>
          。链上读写可使用{" "}
          <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">
            useReadContract
          </code>
          {" / "}
          <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">
            useWriteContract
          </code>{" "}
          等 Wagmi 接口。
        </CardDescription>
      </CardContent>
    </Card>
  );
}
