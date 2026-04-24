import { http, createConfig } from "wagmi";
import { baseSepolia, sepolia } from "wagmi/chains";
import { injected, walletConnect } from "wagmi/connectors";

const sepoliaRpc = import.meta.env.VITE_SEPOLIA_RPC as string | undefined;
const baseSepoliaRpc = import.meta.env.VITE_BASE_SEPOLIA_RPC as string | undefined;
const wcProjectId = import.meta.env.VITE_WALLETCONNECT_PROJECT_ID as string | undefined;

const connectors = [
  injected(),
  ...(wcProjectId
    ? [
        walletConnect({
          projectId: wcProjectId,
        }),
      ]
    : []),
];

export const wagmiConfig = createConfig({
  chains: [sepolia, baseSepolia],
  connectors,
  transports: {
    [sepolia.id]: http(sepoliaRpc || undefined),
    [baseSepolia.id]: http(baseSepoliaRpc || undefined),
  },
});
