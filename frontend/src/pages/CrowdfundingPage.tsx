import { Outlet, useLocation } from "react-router-dom";
import { ModuleChainGate } from "@/components/ModuleChainGate";
import { NavButton } from "@/components/nav-spa";
import { L1_MODULE_CHAIN_ID } from "@/lib/chain-policy";
import { cn } from "@/lib/utils";

function navClass({ isActive }: { isActive: boolean }) {
  return cn(
    "relative rounded-xl px-4 py-2 text-sm font-medium transition-all duration-300",
    isActive
      ? "text-foreground bg-primary/15 shadow-[0_0_15px_rgba(45,212,191,0.15)] ring-1 ring-primary/30"
      : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
  );
}

export function CrowdfundingPage() {
  const { pathname } = useLocation();
  const homeActive = pathname === "/crowdfunding";
  const exploreActive = pathname.startsWith("/crowdfunding/explore");
  const workspaceActive = pathname === "/crowdfunding/me";
  const newProposalActive = pathname.startsWith("/crowdfunding/me/proposals/new");
  const adminActive = pathname.startsWith("/crowdfunding/admin");

  return (
    <div className="space-y-10">
      <div className="glass-card rounded-[24px] p-6 sm:p-8 animate-in fade-in slide-in-from-top-4 duration-300">
        <h2 className="mb-2 text-[10px] font-bold uppercase tracking-[0.3em] text-primary/80 flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-primary animate-pulse"></span>
          Crowdfunding Module
        </h2>
        <p className="text-[15px] leading-relaxed text-muted-foreground/90 max-w-3xl">
          Code Pulse 运行在 <strong className="text-foreground/90">Ethereum Sepolia（L1）</strong>；已连接钱包时若不在该网络，下方主内容将锁定。借贷功能请从顶栏进入「借贷」模块（Base Sepolia / L2）。
        </p>
        <nav className="mt-8 flex flex-wrap gap-2 rounded-2xl bg-black/20 p-2 ring-1 ring-white/5" aria-label="Code Pulse 子导航">
          <NavButton to="/crowdfunding" className={navClass({ isActive: homeActive })} aria-current={homeActive ? "page" : undefined}>
            Home
          </NavButton>
          <NavButton to="/crowdfunding/explore" className={navClass({ isActive: exploreActive })} aria-current={exploreActive ? "page" : undefined}>
            Explore
          </NavButton>
          <NavButton to="/crowdfunding/me" className={navClass({ isActive: workspaceActive })} aria-current={workspaceActive ? "page" : undefined}>
            My Workspace
          </NavButton>
          <NavButton to="/crowdfunding/me/proposals/new" className={navClass({ isActive: newProposalActive })} aria-current={newProposalActive ? "page" : undefined}>
            New Proposal
          </NavButton>
          <NavButton to="/crowdfunding/admin" className={navClass({ isActive: adminActive })} aria-current={adminActive ? "page" : undefined}>
            Admin
          </NavButton>
        </nav>
      </div>

      <div className="animate-in fade-in slide-in-from-bottom-4 duration-300 delay-75 fill-mode-both">
        <ModuleChainGate requiredChainId={L1_MODULE_CHAIN_ID} moduleName="众筹合约（Code Pulse）">
          <Outlet />
        </ModuleChainGate>
      </div>
    </div>
  );
}
