/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string;
  readonly VITE_SEPOLIA_RPC?: string;
  readonly VITE_BASE_SEPOLIA_RPC?: string;
  readonly VITE_WALLETCONNECT_PROJECT_ID?: string;
  readonly VITE_BANK_ADDRESS?: string;
  readonly VITE_NFT_TEMPLATE_ADDRESS?: string;
  readonly VITE_NFT_FACTORY_ADDRESS?: string;
  readonly VITE_NFT_MARKETPLACE_ADDRESS?: string;
  readonly VITE_LENDING_POOL_ADDRESS?: string;
  readonly VITE_LENDING_HYBRID_ORACLE_ADDRESS?: string;
  readonly VITE_LENDING_CHAINLINK_ORACLE_ADDRESS?: string;
  readonly VITE_LENDING_REPORTS_VERIFIER_ADDRESS?: string;
  readonly VITE_LENDING_IR_STRATEGY_FACTORY_ADDRESS?: string;
  readonly VITE_LENDING_IR_STRATEGY_ADDRESS?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
