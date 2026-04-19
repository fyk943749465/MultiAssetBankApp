import type { Address } from "viem";

/** Sepolia 默认与 backend/migrations/005_nft_platform.sql 种子一致；可通过环境变量覆盖。 */
const DEFAULT_TEMPLATE = "0xcbbf6cd8d652289a91dc560944108ad962a69599" as const;
const DEFAULT_FACTORY = "0x32ccf2565f382519d6f8ca2fe12ba7a5ac1f8c90" as const;
const DEFAULT_MARKETPLACE = "0x7bf717d5c1262756e80b99f7eb6c8838cea5c9f6" as const;

function parseAddress(value: string | undefined, fallback: Address): Address {
  if (!value || !/^0x[a-fA-F0-9]{40}$/.test(value.trim())) return fallback;
  return value.trim() as Address;
}

export function getNftTemplateAddress(): Address {
  return parseAddress(import.meta.env.VITE_NFT_TEMPLATE_ADDRESS as string | undefined, DEFAULT_TEMPLATE);
}

export function getNftFactoryAddress(): Address {
  return parseAddress(import.meta.env.VITE_NFT_FACTORY_ADDRESS as string | undefined, DEFAULT_FACTORY);
}

export function getNftMarketplaceAddress(): Address {
  return parseAddress(import.meta.env.VITE_NFT_MARKETPLACE_ADDRESS as string | undefined, DEFAULT_MARKETPLACE);
}
