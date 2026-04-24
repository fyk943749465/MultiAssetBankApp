import { type Address, isAddress } from "viem";

/** 借贷池与周边合约默认地址（Base Sepolia）；可用 VITE_* 覆盖。 */
const DEFAULTS = {
  pool: "0x65213B004b54DeA6CB1096794CA3f1C24066B0ff",
  hybridPriceOracle: "0x37a8224BB0ea0828051adF9569967b4e8d0e1f49",
  chainlinkPriceOracle: "0xF48E792DdA21F978740DF4Acb999C22e84A9Ae6c",
  reportsVerifier: "0xDaAD54b34D4db3FdB0DDDF1aD37316fF862f9ab8",
  interestRateStrategyFactory: "0x7F3d525A1781e295a2AB9Aa74C18F28b984DFa74",
  interestRateStrategy: "0x9B91E7fa1E37d32C93f1bd1EcB7be991b53112A3",
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
