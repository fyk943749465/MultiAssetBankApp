# subgraph 目录说明

本目录存放 **The Graph** 子图工程（与链上合约事件一一对应的索引与 GraphQL 查询层）。  
仓库整体是 **Go 后端 + React 前端 + 子图** 的 monorepo；**业务入账与对账以 PostgreSQL 为准**、子图侧重 **列表与读体验** 时，见根目录文档：

- [`docs/NFT-Platform-Architecture-PG-Subgraph.md`](../docs/NFT-Platform-Architecture-PG-Subgraph.md)（NFT 平台：PG / 扫块 / 子图分工）
- [`docs/api-data-source-rules.md`](../docs/api-data-source-rules.md)（API 数据来源约定）

---

## 1. 仓库里三件套的当前形态（摘要）

### 1.1 后端 `backend/`

- **语言与框架**：Go 1.26，HTTP 服务使用 **Gin**；ORM 为 **GORM**，数据库为 **PostgreSQL**。
- **链交互**：**go-ethereum**（`ethclient`、合约调用、日志过滤等）。
- **能力概览**（详见 [`backend/README.md`](../backend/README.md)）：
  - 健康检查、链状态、示例 Counter 读写；
  - **MultiAssetBank**：RPC 扫块将 `Deposited` / `Withdrawn` 等写入 PG，并提供 REST 查询；可选对接 **The Graph**（`SUBGRAPH_URL` 等环境变量）；
  - **Code Pulse 众筹**：默认以 **RPC 扫块** 写入 `cp_*` 表为权威读模型；可选子图 URL（`SUBGRAPH_CODE_PULSE_URL`）与可选「子图同步入库」开关。
- **文档**：Swagger / ReDoc / Scalar 等由 `swag` 生成，本地起服务后见 `backend/README.md` 中的路径表。

### 1.2 前端 `frontend/`

- **栈**：**React 19**、**Vite 6**、**TypeScript**；样式与组件侧使用 **Tailwind CSS v4**、**@base-ui/react** 等（与 `package.json` 一致）。
- **链上读写**：**wagmi v2** + **viem**（钱包连接、读合约、发交易）。
- **路由与业务模块**：`react-router-dom`；当前可见模块包含 **银行（Bank）**、**众筹（Crowdfunding）** 等页面树（具体路由见 `frontend/src/App.tsx`）。
- **ABI**：部分合约的最小 ABI 放在 `frontend/src/abi/` 等处，供前端直接调用；与子图 ABI 应对齐同一套部署。

### 1.3 子图 `subgraph/`（本目录）

| 子目录 | 用途 |
|--------|------|
| **`nft-platform/`** | **NFT 工厂 + 模板实现 + 市场**（Sepolia 上三份固定地址合约），事件写入 `schema.graphql` 中定义的实体；适合列表、详情、市场流水等 GraphQL 查询。 |
| **`multi-asset-bank-sepolia/`** | **MultiAssetBank**（Sepolia）存取款事件子图；与后端 `SUBGRAPH_URL`、银行相关 API 配套。 |
| **`code-pulse-advanced/`** | **Code Pulse 众筹**合约子图；与后端 `SUBGRAPH_CODE_PULSE_URL`、众筹模块配套。 |

根目录 [`.gitignore`](../.gitignore) 已对子图常见产物做了兜底（如部分路径下的 `node_modules/`、`build/`、`generated/`）；各子项目内另有 `.gitignore`，以子目录为准。

---

## 2. NFT 子图：`nft-platform/`

- **网络**：`sepolia`（`subgraph.yaml` / `networks.json`）。
- **三个数据源（固定地址）**：
  1. **NFTTemplate** — 模板/实现合约，索引 ERC721 相关事件与模板级配置事件；
  2. **NFTFactory** — 工厂（合集创建、费用、暂停、提现等）；
  3. **NFTMarketPlace** — 市场（挂单、改价、取消、成交、平台费与所有权等）；市场侧实体在 schema 中使用 **`NFTMarket*`** 前缀，避免与工厂等同名事件在查询语义上混淆。
- **常用命令**（在 `nft-platform` 目录下）：

```bash
npm install
npm run codegen   # 根据 subgraph.yaml + schema 生成 AssemblyScript 类型
npm run build     # 编译为 WASM，检查映射是否通过
# 部署：在 Subgraph Studio 或自建 Graph Node 上按官方流程 graph deploy（需本机 graph-cli 与密钥/端点）
```

- **合约 ABI**：位于 `nft-platform/abis/*.json`；**`startBlock`** 已在 manifest 中按部署块填写，用于加快同步。

---

## 3. NFT 静态资源（图片）与 Irys Devnet

子图只索引**链上事件**（如 `BaseURIUpdated`、`Transfer`），**不会**上传 PNG 或元数据 JSON。把本地 **`images`** 目录一次性上传到 Devnet（Windows 手工一条命令）见 **[`script/README.md`](../script/README.md)**。元数据生成与上传 `metadata` 为可选，同目录内另有 `generate-nft-metadata.js` 等脚本，需要时自用。

---

## 4. 银行子图：`multi-asset-bank-sepolia/`

- 监听 **MultiAssetBank** 的 **`Deposited`** / **`Withdrawn`** 等事件（以该目录下 `subgraph.yaml` 为准）。
- 与前端 `multiAssetBank` 相关 ABI、后端银行子图查询路径一致即可；部署与调试步骤与下文「通用流程」相同。

---

## 5. Code Pulse 子图：`code-pulse-advanced/`

- 与仓库内 **Code Pulse 众筹**合约验证版本对应；事件与实体以该子图内 `schema.graphql`、`subgraph.yaml` 为准。
- 后端是否只读子图、是否同步进 PG，由 `CODE_PULSE_SUBGRAPH_SYNC` 等环境变量控制（见 `backend/README.md`）。

---

## 6. 通用：初始化、构建与部署要点

1. **安装 CLI**：`npm install -g @graphprotocol/graph-cli`（版本以各子项目 `package.json` 中 `@graphprotocol/graph-cli` 对齐为宜）。
2. **Studio 部署**：在 [Subgraph Studio](https://thegraph.com/studio/) 创建子图 → `graph auth --studio <DEPLOY_KEY>` → 在子项目根目录 `graph deploy --studio <slug>`（具体参数以官方文档为准）。
3. **`startBlock`**：务必设为合约（或该数据源）**首次产生需索引日志的区块**，避免从 0 扫链。
4. **密钥**：Deploy Key、API Key 等仅放在环境变量或本地未提交配置中，勿写入仓库。

更细的「从零 init 到 Playground 验证」 checklist，仍可参考本文件历史版本中针对 **MultiAssetBank** 的逐步说明；当前仓库已包含可直接 `codegen` / `build` 的上述三个子项目，优先直接打开对应子目录的 `subgraph.yaml` 阅读。

---

## 7. 相关链接（外部）

- [The Graph 文档](https://thegraph.com/docs/en/subgraphs/developing/introduction/)
- [Subgraph Studio](https://thegraph.com/studio/)
