import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { ConnectWallet } from "@/components/ConnectWallet";
import { ModeToggle } from "@/components/mode-toggle";
import { cn } from "@/lib/utils";

function navClass({ isActive }: { isActive: boolean }): string {
  return cn(
    "relative cursor-pointer rounded-xl border-0 bg-transparent px-6 py-2.5 text-sm font-medium transition-all duration-300",
    isActive
      ? "text-foreground bg-primary/10 shadow-[0_0_15px_rgba(45,212,191,0.15)] ring-1 ring-primary/20"
      : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
  );
}

export function AppLayout() {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const bankActive = pathname === "/bank";
  const crowdfundingActive = pathname.startsWith("/crowdfunding");
  const nftActive = pathname.startsWith("/nft");
  const lendingActive = pathname.startsWith("/lending");

  return (
    <div className="relative flex min-h-screen flex-col selection:bg-primary/20">
      {/* Absolute Top-Right Controls */}
      <div className="absolute top-4 right-4 z-50 flex items-center gap-2">
        <ModeToggle />
        <ConnectWallet compact />
      </div>

      <main className="mx-auto w-full max-w-5xl px-4 flex-1 pb-24 pt-16 sm:px-6 sm:pt-20 lg:px-8">
        <div className="relative mb-16 text-center">
          {/* Subtle glow behind title */}
          <div className="absolute left-1/2 top-1/2 -z-10 h-[120px] w-[60%] -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary/20 opacity-50 blur-[80px]"></div>
          
          <p className="mb-4 text-[10px] font-bold uppercase tracking-[0.4em] text-primary/80">
            L1 Sepolia · L2 Base Sepolia · Wagmi · Viem
          </p>
          <h1 className="text-5xl font-extrabold tracking-tight sm:text-7xl">
            <span className="text-gradient-brand drop-shadow-sm">GO-CHAIN</span>
          </h1>
          <p className="mx-auto mt-6 max-w-lg text-[15px] leading-relaxed text-muted-foreground">
            多业务入口：银行 / 众筹 / NFT 仅 <span className="text-foreground font-medium">Ethereum Sepolia（L1）</span>；借贷仅{" "}
            <span className="text-foreground font-medium">Base Sepolia（L2）</span>。钱包在右上角，按当前页面自动提示应处网络。
          </p>

          <nav
            className="mx-auto mt-10 inline-flex flex-wrap items-center justify-center gap-2 rounded-[20px] glass-panel p-2.5"
            aria-label="业务模块"
          >
            <button
              type="button"
              className={navClass({ isActive: bankActive })}
              aria-current={bankActive ? "page" : undefined}
              onClick={() => navigate("/bank")}
            >
              银行与后端
            </button>
            <button
              type="button"
              className={navClass({ isActive: crowdfundingActive })}
              aria-current={crowdfundingActive ? "page" : undefined}
              onClick={() => navigate("/crowdfunding")}
            >
              众筹合约
            </button>
            <button
              type="button"
              className={navClass({ isActive: nftActive })}
              aria-current={nftActive ? "page" : undefined}
              onClick={() => navigate("/nft")}
            >
              NFT
            </button>
            <button
              type="button"
              className={navClass({ isActive: lendingActive })}
              aria-current={lendingActive ? "page" : undefined}
              onClick={() => navigate("/lending")}
            >
              借贷
            </button>
          </nav>
        </div>

        <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
          <Outlet />
        </div>
      </main>

      <footer className="mt-auto border-t border-white/5 bg-background/40 backdrop-blur-md py-8 text-center text-xs text-muted-foreground">
        <p className="opacity-80">
          GO-CHAIN · 本地开发请使用系统浏览器打开前端地址以使用{" "}
          <code className="rounded border border-border bg-muted/50 px-1.5 py-0.5 font-mono text-[10px]">
            window.ethereum
          </code>
        </p>
      </footer>
    </div>
  );
}
