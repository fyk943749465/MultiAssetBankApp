import { type Address, isAddress } from "viem";

const DEFAULT_BANK = "0x668A7A8372C41EE0be46a4eA34e6eafeaA4E9748" as const;

export function getBankAddress(): Address {
  const raw = import.meta.env.VITE_BANK_ADDRESS as string | undefined;
  if (raw && isAddress(raw)) return raw as Address;
  return DEFAULT_BANK;
}
