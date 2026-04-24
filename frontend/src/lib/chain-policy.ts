import { baseSepolia, sepolia } from "wagmi/chains";

/** Code Pulse、银行链上交互、NFT 市场等：Ethereum Sepolia（L1 测试网）。 */
export const L1_MODULE_CHAIN_ID = sepolia.id;

/** 借贷 Pool / 喂价等：Base Sepolia（L2 测试网）。 */
export const L2_LENDING_CHAIN_ID = baseSepolia.id;

export function isLendingRoute(pathname: string): boolean {
  return pathname === "/lending" || pathname.startsWith("/lending/");
}

/** 当前路径下，钱包应处于的链（用于 UI 限制与切换提示）。 */
export function getRouteRequiredChainId(pathname: string): number {
  return isLendingRoute(pathname) ? L2_LENDING_CHAIN_ID : L1_MODULE_CHAIN_ID;
}

export function chainIdLabel(chainId: number): string {
  if (chainId === sepolia.id) return "Ethereum Sepolia";
  if (chainId === baseSepolia.id) return "Base Sepolia";
  return `链 ${chainId}`;
}
