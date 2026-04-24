import type { ReactNode } from "react";
import { useAccount, useChainId, useSwitchChain } from "wagmi";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { chainIdLabel } from "@/lib/chain-policy";

type ModuleChainGateProps = {
  /** 本模块允许的 chainId（Sepolia 或 Base Sepolia）。 */
  readonly requiredChainId: number;
  /** 用于提示文案，如「借贷模块」「众筹合约」。 */
  readonly moduleName: string;
  readonly children: ReactNode;
};

/**
 * 已连接钱包且链不匹配时，拦截子内容并引导切换；
 * 未连接时仍渲染子内容（只读浏览等），链上操作由各组件自行依赖钱包。
 */
export function ModuleChainGate({ requiredChainId, moduleName, children }: ModuleChainGateProps) {
  const { isConnected } = useAccount();
  const chainId = useChainId();
  const { switchChain, isPending } = useSwitchChain();

  const mismatch = isConnected && chainId !== requiredChainId;
  if (!mismatch) {
    return <>{children}</>;
  }

  const need = chainIdLabel(requiredChainId);

  return (
    <Alert variant="destructive" className="border-destructive/50">
      <AlertTitle>网络与当前模块不一致</AlertTitle>
      <AlertDescription className="mt-2 space-y-3 text-sm leading-relaxed">
        <p>
          <strong className="text-foreground">{moduleName}</strong> 仅允许在{" "}
          <strong className="text-foreground">{need}</strong>
          <span className="font-mono text-foreground">（{requiredChainId}）</span>
          下使用。当前钱包在 <strong className="text-foreground">{chainIdLabel(chainId)}</strong>
          <span className="font-mono text-foreground">（{chainId}）</span>，已隐藏下方链上相关界面。
        </p>
        <Button type="button" size="sm" disabled={isPending} onClick={() => switchChain({ chainId: requiredChainId })}>
          {isPending ? "切换中…" : `切换到 ${need}`}
        </Button>
      </AlertDescription>
    </Alert>
  );
}
