# go-chain frontend

React 19 + Vite 6 + TypeScript，链交互使用 **wagmi v2** 与 **viem**，样式为 **Tailwind CSS v4** 与 **Base UI** 等（依赖见 `package.json`）。

## 运行

在 **`frontend`** 目录：

```bash
npm install
npm run dev
```

生产构建：

```bash
npm run build
```

开发模式下，`src/features/nft/api.ts` 将 API 前缀设为相对路径 **`""`**，由 Vite 将 **`/api`**、**`/health`** 代理到 **`http://127.0.0.1:8080`**（见 `vite.config.ts`）。生产构建默认使用 **`VITE_API_BASE`**，未设置时回退为 `http://127.0.0.1:8080`。

## 路由与功能（节选）

入口路由定义在 **`src/App.tsx`**。

| 路径前缀 | 说明 |
|----------|------|
| `/bank` | 多资产银行 |
| `/crowdfunding/*` | Code Pulse 众筹（探索、我的、管理、提案与活动等） |
| `/nft` | NFT 模块概览 |
| `/nft/me` | **我的 NFT**：按地址查询后端 **`GET /api/nft/holdings`**，并可链上 `ownerOf` / `tokenURI` 校验与展示图片 |
| `/nft/market` | **NFT 市场**：**`GET /api/nft/listings/active`** 列表（子图优先、失败时 PG 兜底并展示 `subgraph_fallback_error`）；链上购买、撤单、上架、改价（`NFTMarketPlace` + 合集 `ERC721` 授权） |
| `/nft/market/history` | **市场事件历史**：**`GET /api/nft/market/trade-events`**（上架 / 改价 / 撤单 / 成交），支持类型筛选与「仅与我相关」 |
| `/nft/create` | 创建合集 |
| `/nft/collections/:contractAddress/mint` | 铸造页 |
| `/nft/collections/:collectionId` | 合集详情 |

子导航在 **`src/pages/nft/NftPage.tsx`**；NFT 相关 REST 封装在 **`src/features/nft/api.ts`**，市场页 ABI 在 **`src/abi/nftMarketplace.ts`**、**`src/abi/erc721Approve.ts`** 等。

## 与后端的约定

- **列表展示**：市场挂单可来自子图或数据库，响应里的 **`data_source`** 与可选 **`subgraph_fallback_error`** 用于提示数据来源。
- **购买**：发起 **`buyItem`** 前会调用 **`GET /api/nft/listings/verify-active`**，仅在 PostgreSQL 中存在对应**活跃挂单**且价格、卖家与列表一致时才继续，避免仅子图可见却与索引不一致时误买。

更完整的 HTTP 路径、查询参数与环境变量见 **[`../backend/README.md`](../backend/README.md)** 中的 **NFT 平台** 一节。

## 延伸阅读

- [`../docs/NFT-Platform-Architecture-PG-Subgraph.md`](../docs/NFT-Platform-Architecture-PG-Subgraph.md) — NFT：PG 与子图分工  
- [`../backend/README.md`](../backend/README.md) — 后端接口与环境变量  
- [`../subgraph/README.md`](../subgraph/README.md) — 子图工程说明  
