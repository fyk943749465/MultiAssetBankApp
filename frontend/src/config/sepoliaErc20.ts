import { type Address } from "viem";

/** Sepolia 测试网常见 ERC20（部分为社区常用地址；无余额需 faucet / 自行领水） */
export type Erc20Preset = { symbol: string; address: Address; note?: string };

export const SEPOLIA_ERC20_PRESETS: readonly Erc20Preset[] = [
  { symbol: "LINK", address: "0x779877A7B0D9E8603169DdbD7836e478b4624789", note: "Chainlink" },
  { symbol: "USDC", address: "0x1c7D4B196Cb0C7B01D743Fbc6116a902379C7238", note: "Circle 测试 USDC" },
  { symbol: "WETH", address: "0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9", note: "Wrapped ETH（常见）" },
  { symbol: "WETH2", address: "0xfff9976782d46Cc0563061a30D76441dC0136f38", note: "另一 Sepolia WETH 地址" },
  { symbol: "DAI", address: "0xf5c142292B85253E4D071812C84f05Ec42828fdb", note: "测试 DAI" },
  { symbol: "USDT", address: "0xaA8E23Fb1079EA9d0A68C96913Df2b1B74EfBD57", note: "测试 USDT" },
  { symbol: "EURC", address: "0x644aA32988756B6070133d01cFde1a666E09Dd99", note: "Circle 测试 EURC" },
  { symbol: "PYUSD", address: "0xCaC524BcA292aa9472f60797e097Da5081AAAA5A", note: "测试 PYUSD" },
  { symbol: "AAVE", address: "0x5bB9D650fEa6BBc642c9d998703fA317A06D5183", note: "测试 AAVE" },
  { symbol: "GRT", address: "0x5c142E25BcB86B2FC042c3A2164245fE699aaf0F", note: "The Graph 测试" },
] as const;
