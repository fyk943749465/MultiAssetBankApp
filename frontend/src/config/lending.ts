import { type Address, isAddress } from "viem";

/** 借贷池与周边合约默认地址（Base Sepolia）；可用 VITE_* 覆盖。 */
/** 与 subgraph/lending 当前 networks 默认一致（Base Sepolia 新部署） */
const DEFAULTS = {
  pool: "0x3f0248e6fF7E414485A146C18d6B72Dc9E317E5F",
  hybridPriceOracle: "0xE72Ac9c1D557d65094aE92739e409cA56AE12B11",
  chainlinkPriceOracle: "0x3100b1FD5A2180dAc11820106579545D0f1C439b",
  reportsVerifier: "0x960e004f33566D0B56863F54532F1785923d2799",
  interestRateStrategyFactory: "0xB44D1c69EaF762441d6762e094B18d2614CF1617",
  interestRateStrategy: "0x0f4C88d757e370016B5cfC1ac48d013378BE4a27",
} as const;

export const LENDING_CHAIN_ID = 84532 as const;
export const LENDING_CHAIN_NAME = "Base Sepolia";

function readAddr(raw: string | undefined, fallback: (typeof DEFAULTS)[keyof typeof DEFAULTS]): Address {
  if (raw && isAddress(raw)) return raw;
  return fallback;
}

export function getLendingPoolAddress(): Address {
  return readAddr(import.meta.env.VITE_LENDING_POOL_ADDRESS, DEFAULTS.pool);
}

export function getLendingHybridPriceOracleAddress(): Address {
  return readAddr(import.meta.env.VITE_LENDING_HYBRID_ORACLE_ADDRESS, DEFAULTS.hybridPriceOracle);
}

export function getLendingChainlinkPriceOracleAddress(): Address {
  return readAddr(import.meta.env.VITE_LENDING_CHAINLINK_ORACLE_ADDRESS, DEFAULTS.chainlinkPriceOracle);
}

export function getLendingReportsVerifierAddress(): Address {
  return readAddr(import.meta.env.VITE_LENDING_REPORTS_VERIFIER_ADDRESS, DEFAULTS.reportsVerifier);
}

export function getLendingInterestRateStrategyFactoryAddress(): Address {
  return readAddr(import.meta.env.VITE_LENDING_IR_STRATEGY_FACTORY_ADDRESS, DEFAULTS.interestRateStrategyFactory);
}

export function getLendingInterestRateStrategyAddress(): Address {
  return readAddr(import.meta.env.VITE_LENDING_IR_STRATEGY_ADDRESS, DEFAULTS.interestRateStrategy);
}

export type LendingContractRow = {
  readonly key: string;
  readonly label: string;
  readonly address: Address;
  readonly description: string;
};

export function getLendingContractRows(): LendingContractRow[] {
  return [
    {
      key: "pool",
      label: "Pool",
      address: getLendingPoolAddress(),
      description: "供应 / 借款 / 还款 / 清算、储备与利率指数。",
    },
    {
      key: "hybridOracle",
      label: "HybridPriceOracle",
      address: getLendingHybridPriceOracleAddress(),
      description: "混合喂价（Stream + Chainlink 兜底）。",
    },
    {
      key: "chainlinkOracle",
      label: "ChainlinkPriceOracle",
      address: getLendingChainlinkPriceOracleAddress(),
      description: "AggregatorV3 纯链上喂价。",
    },
    {
      key: "reportsVerifier",
      label: "ReportsVerifier",
      address: getLendingReportsVerifierAddress(),
      description: "Chainlink VerifierProxy 验证报告。",
    },
    {
      key: "irFactory",
      label: "InterestRateStrategyFactory",
      address: getLendingInterestRateStrategyFactoryAddress(),
      description: "部署 kink 利率策略实例并发出 StrategyCreated。",
    },
    {
      key: "irStrategy",
      label: "InterestRateStrategy",
      address: getLendingInterestRateStrategyAddress(),
      description: "池子绑定的利率曲线（immutable 参数）。",
    },
  ];
}
