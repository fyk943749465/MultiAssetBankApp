import { NavLink, Outlet } from "react-router-dom";
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
  return (
    <div className="space-y-10">
      <div className="glass-card rounded-[24px] p-6 sm:p-8 animate-in fade-in slide-in-from-top-4 duration-700">
        <h2 className="mb-2 text-[10px] font-bold uppercase tracking-[0.3em] text-primary/80 flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-primary animate-pulse"></span>
          Crowdfunding Module
        </h2>
        <p className="text-[15px] leading-relaxed text-muted-foreground/90 max-w-3xl">
          Code Pulse 第二阶段已接入动作预检、交易构建/提交、提案创建与管理台。你可以在详情页、工作台和 Admin 页面直接执行链上业务动作。
        </p>
        <nav className="mt-8 flex flex-wrap gap-2 rounded-2xl bg-black/20 p-2 ring-1 ring-white/5" aria-label="Code Pulse 子导航">
          <NavLink end to="/crowdfunding" className={navClass}>
            Home
          </NavLink>
          <NavLink to="/crowdfunding/explore" className={navClass}>
            Explore
          </NavLink>
          <NavLink end to="/crowdfunding/me" className={navClass}>
            My Workspace
          </NavLink>
          <NavLink to="/crowdfunding/me/proposals/new" className={navClass}>
            New Proposal
          </NavLink>
          <NavLink to="/crowdfunding/admin" className={navClass}>
            Admin
          </NavLink>
        </nav>
      </div>

      <div className="animate-in fade-in slide-in-from-bottom-4 duration-700 delay-150 fill-mode-both">
        <Outlet />
      </div>
    </div>
  );
}
