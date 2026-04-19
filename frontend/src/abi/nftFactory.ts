/** 与 `subgraph/nft-platform/abis/NFTFactory.json` 对齐的最小子集：创建费、暂停、两种部署、`predictCloneAddress`、事件。 */
export const nftFactoryAbi = [
  {
    inputs: [],
    name: "creationFee",
    outputs: [{ internalType: "uint256", name: "", type: "uint256" }],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [],
    name: "paused",
    outputs: [{ internalType: "bool", name: "", type: "bool" }],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      { internalType: "string", name: "name", type: "string" },
      { internalType: "string", name: "symbol", type: "string" },
      { internalType: "string", name: "baseURI", type: "string" },
    ],
    name: "deployProxy",
    outputs: [{ internalType: "address", name: "", type: "address" }],
    stateMutability: "payable",
    type: "function",
  },
  {
    inputs: [
      { internalType: "string", name: "name", type: "string" },
      { internalType: "string", name: "symbol", type: "string" },
      { internalType: "string", name: "baseURI", type: "string" },
      { internalType: "bytes32", name: "salt", type: "bytes32" },
    ],
    name: "deployProxyDeterministic",
    outputs: [{ internalType: "address", name: "", type: "address" }],
    stateMutability: "payable",
    type: "function",
  },
  {
    inputs: [{ internalType: "bytes32", name: "salt", type: "bytes32" }],
    name: "predictCloneAddress",
    outputs: [{ internalType: "address", name: "", type: "address" }],
    stateMutability: "view",
    type: "function",
  },
  {
    anonymous: false,
    inputs: [
      { indexed: true, internalType: "address", name: "collection", type: "address" },
      { indexed: true, internalType: "address", name: "creator", type: "address" },
      { indexed: false, internalType: "uint256", name: "feePaid", type: "uint256" },
      { indexed: false, internalType: "bytes32", name: "salt", type: "bytes32" },
    ],
    name: "CollectionCreated",
    type: "event",
  },
] as const;
