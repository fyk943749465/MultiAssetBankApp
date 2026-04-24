import { Outlet, useLocation } from "react-router-dom";
import { ModuleChainGate } from "@/components/ModuleChainGate";
import { NavButton } from "@/components/nav-spa";
import { L2_LENDING_CHAIN_ID } from "@/lib/chain-policy";
import { LendingBackendStrip } from "@/features/lending/LendingBackendStrip";
import { cn } from "@/lib/utils";

function navClass({ isActive }: { isActive: boolean }) {
  return cn(
    "relative rounded-xl px-4 py-2 text-sm font-medium transition-all duration-300",
    isActive
      ? "text-foreground bg-primary/15 shadow-[0_0_15px_rgba(45,212,191,0.15)] ring-1 ring-primary/30"
      : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
  );
}

export function LendingPage() {
  const { pathname } = useLocation();
  const homeActive = pathname === "/lending" || pathname === "/lending/";
  const poolActive = pathname.startsWith("/lending/pool");

  return (
    <div className="space-y-10">
      <div className="glass-card animate-in fade-in slide-in-from-top-4 duration-300 rounded-[24px] p-6 sm:p-8">
        <h2 className="mb-2 flex items-center gap-2 text-[10px] font-bold uppercase tracking-[0.3em] text-primary/80">
          <span className="h-2 w-2 shrink-0 animate-pulse rounded-full bg-primary" aria-hidden />
          <span>Lending Module</span>
        </h2>
        <p className="max-w-3xl text-[15px] leading-relaxed text-muted-foreground/90">
          借贷为 <strong className="text-foreground/90">L2（Base Sepolia）</strong> 部署，与银行 / 众筹 / NFT 使用的{" "}
          <strong className="text-foreground/90">L1（Ethereum Sepolia）</strong>{" "}
          分离；已连接钱包时若不在 Base Sepolia，下方主内容区将锁定直至切换网络。本页先提供模块规划与合约清单，池上交互后续接入。
        </p>
        <nav
          className="mt-8 flex flex-wrap gap-2 rounded-2xl bg-black/20 p-2 ring-1 ring-white/5"
          aria-label="借贷子导航"
        >
          <NavButton to="/lending" className={navClass({ isActive: homeActive })} aria-current={homeActive ? "page" : undefined}>
            概览与规划
          </NavButton>
          <NavButton to="/lending/pool" className={navClass({ isActive: poolActive })} aria-current={poolActive ? "page" : undefined}>
            池子交互
          </NavButton>
        </nav>
      </div>

      <LendingBackendStrip />

      <div className="animate-in fade-in slide-in-from-bottom-4 duration-300 delay-75 fill-mode-both">
        <ModuleChainGate requiredChainId={L2_LENDING_CHAIN_ID} moduleName="借贷模块">
          <Outlet />
        </ModuleChainGate>
      </div>
    </div>
  );
}
