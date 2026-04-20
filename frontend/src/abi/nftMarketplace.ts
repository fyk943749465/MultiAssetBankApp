/** 与 subgraph/nft-platform/abis/NFTMarketPlace.json 对齐的最小子集：浏览、购买、挂单、撤单、改价。 */
export const nftMarketplaceAbi = [
  {
    name: "buyItem",
    type: "function",
    stateMutability: "payable",
    inputs: [
      { name: "collection", type: "address", internalType: "address" },
      { name: "tokenId", type: "uint256", internalType: "uint256" },
    ],
    outputs: [],
  },
  {
    name: "cancelListing",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [
      { name: "collection", type: "address", internalType: "address" },
      { name: "tokenId", type: "uint256", internalType: "uint256" },
    ],
    outputs: [],
  },
  {
    name: "listFormatItem",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [
      { name: "collection", type: "address", internalType: "address" },
      { name: "tokenId", type: "uint256", internalType: "uint256" },
      { name: "price", type: "uint256", internalType: "uint256" },
    ],
    outputs: [],
  },
  {
    name: "updateListingPrice",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [
      { name: "collection", type: "address", internalType: "address" },
      { name: "tokenId", type: "uint256", internalType: "uint256" },
      { name: "newPrice", type: "uint256", internalType: "uint256" },
    ],
    outputs: [],
  },
  {
    name: "listings",
    type: "function",
    stateMutability: "view",
    inputs: [
      { name: "", type: "address", internalType: "address" },
      { name: "", type: "uint256", internalType: "uint256" },
    ],
    outputs: [
      { name: "seller", type: "address", internalType: "address" },
      { name: "price", type: "uint256", internalType: "uint256" },
      { name: "active", type: "bool", internalType: "bool" },
    ],
  },
  {
    name: "paused",
    type: "function",
    stateMutability: "view",
    inputs: [],
    outputs: [{ name: "", type: "bool", internalType: "bool" }],
  },
] as const;
