# go-chain

以太坊 **Sepolia** 等环境下的 **全栈 DApp 单仓库**：**Go 后端**（REST + PostgreSQL 链上索引）、**React 前端**（钱包与页面）、以及 **The Graph 子图**（事件索引与 GraphQL）。  
业务上覆盖 **多资产银行（MultiAssetBank）**、**Code Pulse 众筹**，以及 **NFT 工厂 / 模板 / 二级市场** 等能力；**入账与对账以 PostgreSQL 为准**、子图侧重读查询时的说明见 [`docs/NFT-Platform-Architecture-PG-Subgraph.md`](docs/NFT-Platform-Architecture-PG-Subgraph.md)。

---

## 后端（`backend/`）

| 项 | 说明 |
|----|------|
| **语言 / 运行时** | Go 1.26（见 [`go.work`](go.work)） |
| **HTTP** | [Gin](https://github.com/gin-gonic/gin)，OpenAPI 由 [swag](https://github.com/swaggo/swag) 生成 |
| **数据库** | PostgreSQL，[GORM](https://gorm.io/) |
| **链** | [go-ethereum](https://github.com/ethereum/go-ethereum)：`ethclient`、合约调用、`eth_getLogs` 扫块索引 |

**主要职责**：提供 `/health`、`/api/chain/status` 等通用接口；**MultiAssetBank** 的充值/提现等事件经 RPC 扫块写入 PG 并对外查询；**Code Pulse 众筹**默认同样以 RPC 扫块写入 `cp_*` 表为权威读模型。可选配置 **The Graph** 查询地址（银行子图、众筹子图），用于接口或前端聚合展示。

**怎么跑、环境变量、接口表**：请直接看 [`backend/README.md`](backend/README.md)。

---

## 前端（`frontend/`）

| 项 | 说明 |
|----|------|
| **栈** | React 19、Vite 6、TypeScript |
| **链上** | [wagmi](https://wagmi.sh/) v2 + [viem](https://viem.sh/)（连接钱包、读合约、发交易） |
| **UI** | Tailwind CSS v4、Base UI 等（详见 `frontend/package.json`） |
| **路由** | React Router；当前包含 **银行**、**众筹** 等页面（入口见 `frontend/src/App.tsx`） |

**怎么跑**：在 `frontend/` 下执行 `npm install` 与 `npm run dev`（默认 Vite 开发服务器，端口以终端输出为准）。

---

## 子图（`subgraph/`）

基于 **The Graph** 的 **AssemblyScript 映射**，把链上 **事件** 写成 **GraphQL 实体**，便于列表与复杂筛选。

| 子目录 | 内容 |
|--------|------|
| **`nft-platform/`** | **NFTTemplate**、**NFTFactory**、**NFTMarketPlace** 三个固定数据源（Sepolia），覆盖铸造、工厂、挂单/成交等事件。 |
| **`multi-asset-bank-sepolia/`** | **MultiAssetBank** 存取款等事件，与后端 `SUBGRAPH_URL` 等配置配套。 |
| **`code-pulse-advanced/`** | **Code Pulse 众筹**合约事件，与后端 `SUBGRAPH_CODE_PULSE_URL` 等配套。 |

在对应子目录内通常执行：`npm install` → `npm run codegen` → `npm run build`；部署到 Studio 或自建 Graph Node 的步骤见 [`subgraph/README.md`](subgraph/README.md)。

---

## 仓库结构（节选）

```text
go-chain/
├── backend/          # Go API、扫块索引、Swagger
├── frontend/         # React + Vite 前端
├── subgraph/         # 多个子图工程（nft-platform、银行等）
├── docs/             # 架构与约定说明
├── go.work           # Go workspace（当前包含 backend 模块）
└── README.md         # 本文件
```

---

## 延伸阅读

- [`docs/NFT-Platform-Architecture-PG-Subgraph.md`](docs/NFT-Platform-Architecture-PG-Subgraph.md) — NFT：PostgreSQL 权威、扫块与子图分工  
- [`docs/api-data-source-rules.md`](docs/api-data-source-rules.md) — API 数据来源约定  
- [`subgraph/README.md`](subgraph/README.md) — 子图目录与各子工程说明  
