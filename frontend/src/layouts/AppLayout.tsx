import { NavLink, Outlet } from "react-router-dom";
import { ConnectWallet } from "../components/ConnectWallet";
import { btnGhost, code } from "../ui/styles";

function navClass({ isActive }: { isActive: boolean }): string {
  const base = btnGhost;
  if (isActive) {
    return `${base} border-emerald-500/45 bg-emerald-950/35 text-emerald-100`;
  }
  return `${base} text-slate-400`;
}

export function AppLayout() {
  return (
    <div className="mx-auto min-h-screen max-w-5xl px-4 pb-16 pt-6 sm:px-6 sm:pt-8 lg:px-8">
      <header className="mb-8 sm:mb-10">
        <div className="mb-4 flex justify-end sm:mb-5">
          <ConnectWallet compact />
        </div>

        <div className="text-center">
          <p className="mb-2 text-xs font-semibold uppercase tracking-[0.35em] text-emerald-500/80">
            Sepolia · Wagmi · Viem
          </p>
          <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
            <span className="text-gradient-brand">GO-CHAIN</span>
          </h1>
          <p className="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-slate-400">
            多业务入口：各模块页面相互独立；<span className="text-slate-300">钱包</span>在右上角，全站共用。
          </p>

          <nav
            className="mx-auto mt-8 flex max-w-md flex-wrap items-center justify-center gap-2"
            aria-label="业务模块"
          >
            <NavLink to="/bank" className={navClass}>
              银行与后端
            </NavLink>
            <NavLink to="/crowdfunding" className={navClass}>
              众筹合约
            </NavLink>
          </nav>
        </div>
      </header>

      <main>
        <Outlet />
      </main>

      <footer className="mt-14 border-t border-slate-800/80 pt-8 text-center text-xs text-slate-600">
        GO-CHAIN · 本地开发请使用系统浏览器打开前端地址以使用 <code className={code}>window.ethereum</code>
      </footer>
    </div>
  );
}
