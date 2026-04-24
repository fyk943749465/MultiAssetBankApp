import { Component, type ErrorInfo, type ReactNode } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { ModuleChainGate } from "@/components/ModuleChainGate";
import { NavButton } from "@/components/nav-spa";
import { L1_MODULE_CHAIN_ID } from "@/lib/chain-policy";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type NftOutletErrorBoundaryProps = { readonly children: ReactNode };
type NftOutletErrorBoundaryState = { readonly error: Error | null };

/** 避免 NFT 子树单次渲染异常拖垮整页白屏。 */
class NftOutletErrorBoundary extends Component<NftOutletErrorBoundaryProps, NftOutletErrorBoundaryState> {
  state: NftOutletErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): NftOutletErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[NFT route]", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="rounded-2xl border border-destructive/40 bg-destructive/10 p-6 text-sm shadow-sm">
          <p className="font-semibold text-destructive">NFT 内容区渲染出错</p>
          <p className="mt-2 text-muted-foreground">
            这通常是接口返回了非预期数据或浏览器扩展干扰。可把下面错误信息发给开发者排查。
          </p>
          <pre className="mt-3 max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-background/80 p-3 font-mono text-xs text-foreground">
            {this.state.error.message}
          </pre>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="mt-4"
            onClick={() => this.setState({ error: null })}
          >
            重试渲染
          </Button>
        </div>
      );
    }
    return this.props.children;
  }
}

function navClass({ isActive }: { isActive: boolean }) {
  return cn(
    "relative rounded-xl px-4 py-2 text-sm font-medium transition-all duration-300",
    isActive
      ? "text-foreground bg-primary/15 shadow-[0_0_15px_rgba(45,212,191,0.15)] ring-1 ring-primary/30"
      : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
  );
}

export function NftPage() {
  const { pathname } = useLocation();
  const homeActive = pathname === "/nft" || pathname === "/nft/";
  const meActive = pathname === "/nft/me" || pathname.startsWith("/nft/me/");
  const marketActive = pathname === "/nft/market" || pathname.startsWith("/nft/market/");
  const createActive = pathname === "/nft/create" || pathname.startsWith("/nft/create/");

  return (
    <div className="space-y-10">
      <div className="glass-card rounded-[24px] p-6 sm:p-8 animate-in fade-in slide-in-from-top-4 duration-300">
        <h2 className="mb-2 flex items-center gap-2 text-[10px] font-bold uppercase tracking-[0.3em] text-primary/80">
          <span className="h-2 w-2 shrink-0 animate-pulse rounded-full bg-primary" aria-hidden />
          <span>NFT Module</span>
        </h2>
        <p className="max-w-3xl text-[15px] leading-relaxed text-muted-foreground/90">
          NFT 链上合约为 <strong className="text-foreground/90">Ethereum Sepolia（L1）</strong>；已连接钱包时若不在该网络，下方主内容将锁定。工厂、模板与市场地址来自环境变量；列表由后端{" "}
          <code className="rounded bg-muted/80 px-1 text-[13px]">/api/nft/*</code> 提供，子图可用时优先子图。
        </p>
        <nav className="mt-8 flex flex-wrap gap-2 rounded-2xl bg-black/20 p-2 ring-1 ring-white/5" aria-label="NFT 子导航">
          <NavButton to="/nft" className={navClass({ isActive: homeActive })} aria-current={homeActive ? "page" : undefined}>
            概览
          </NavButton>
          <NavButton to="/nft/me" className={navClass({ isActive: meActive })} aria-current={meActive ? "page" : undefined}>
            我的 NFT
          </NavButton>
          <NavButton
            to="/nft/market"
            className={navClass({ isActive: marketActive })}
            aria-current={marketActive ? "page" : undefined}
          >
            市场
          </NavButton>
          <NavButton
            to="/nft/create"
            className={navClass({ isActive: createActive })}
            aria-current={createActive ? "page" : undefined}
          >
            创建合集
          </NavButton>
        </nav>
      </div>

      <div className="animate-in fade-in slide-in-from-bottom-4 duration-300 delay-75">
        <NftOutletErrorBoundary>
          <ModuleChainGate requiredChainId={L1_MODULE_CHAIN_ID} moduleName="NFT 平台">
            <Outlet />
          </ModuleChainGate>
        </NftOutletErrorBoundary>
      </div>
    </div>
  );
}
