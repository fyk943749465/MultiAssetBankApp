# MultiAssetBank · The Graph 子图说明

本目录用于存放 **The Graph（子图 / Subgraph）** 相关说明与后续子图工程。合约参考：

- [Sepolia 上的 MultiAssetBank（Etherscan）](https://sepolia.etherscan.io/address/0x668a7a8372c41ee0be46a4ea34e6eafeaa4e9748#code)

仓库内对应的最小 ABI 见：`frontend/src/abi/multiAssetBank.ts`（含 `Deposited` / `Withdrawn` 事件）。

---

## 1. The Graph 在做什么

- **Subgraph**：配置「监听哪些合约、哪些事件、如何写成可查询的数据」；**索引器**扫链，通过 **GraphQL** 对外查询。
- **与当前仓库的关系**：`backend` 已有 **Go + PostgreSQL** 索引账本；The Graph 是另一套索引与查询方式。两者可并存：前端可继续调 `/api`，也可增加 GraphQL 查询。

---

## 2. 部署子图前准备

1. **合约地址**：`0x668A7A8372C41EE0be46a4eA34e6eafeaA4E9748`（Sepolia）。
2. **完整 ABI**：在 Etherscan **Contract → Code** 复制 **Contract ABI**（JSON）。子图建议使用**完整 ABI**，避免漏事件；本地最小 ABI 仅够前端调用。
3. **部署区块 `startBlock`**：在 Etherscan 查看合约 **创建区块**（或首次出现合约的区块）。`subgraph.yaml` 中填写该值，可明显加快同步（不要从 0 开始）。
4. **稳定的 Sepolia RPC**：Studio 或自建 Graph Node 都需要稳定 JSON-RPC（如 Infura、Alchemy 等）。

---

## 3. 推荐路径：Subgraph Studio

### 3.1 创建子图

1. 打开 [Subgraph Studio](https://thegraph.com/studio/)，使用钱包登录。
2. **Create a Subgraph**，命名（例如 `multi-asset-bank-sepolia`），记下 **subgraph slug**（形如 `account/subgraph-name`）。
3. 在子图详情页复制 **Deploy Key**（勿提交到公开仓库）。

### 3.2 安装 CLI 并登录

```bash
npm install -g @graphprotocol/graph-cli
graph auth --studio <你的_DEPLOY_KEY>
```

官方文档以最新为准：[Deploying using Subgraph Studio](https://thegraph.com/docs/en/subgraphs/developing/deploying/using-subgraph-studio/)。

### 3.3 初始化子图工程

在本目录或新目录执行 `graph init`（交互式），选择：

- **Product**：Subgraph Studio  
- **Network**：Ethereum **Sepolia**（manifest 中网络标识多为 `sepolia`，以 CLI 提示为准）  
- **Contract**：`0x668A7A8372C41EE0be46a4eA34e6eafeaA4E9748`  
- 导入 Etherscan 导出的 ABI  

也可使用 `graph init` 的非交互参数，以本机 `graph init --help` 为准。

典型生成结构：

| 文件 | 作用 |
|------|------|
| `subgraph.yaml` | 网络、合约地址、事件、`startBlock`、映射入口 |
| `schema.graphql` | 实体（GraphQL 可查询的「表」） |
| `src/mapping.ts` | 事件处理（AssemblyScript） |

### 3.4 `subgraph.yaml` 要点

在 `dataSources` 中监听与 ABI 一致的事件：

- `Deposited(indexed address token, indexed address user, uint256 amount)`
- `Withdrawn(indexed address token, indexed address user, uint256 amount)`

将 `startBlock` 设为合约部署块。

### 3.5 `schema.graphql` 设计思路

按需定义实体，例如：

- **`Deposit`** / **`Withdraw`**：字段含 user、token、amount、blockNumber、timestamp、txHash 等；  
- 或使用统一 **`LedgerEntry`**，增加 `type` 区分存/取。

**实体 id 必须全局唯一**，常用：`${transaction.hash}-${logIndex}`。

### 3.6 `mapping.ts` 要点

为 `Deposited`、`Withdrawn` 各写一个 handler：

- 从 `event.params` 读取 `token`、`user`、`amount`  
- 使用 `event.block.number`、`event.block.timestamp`、`event.transaction.hash`  
- 调用 `entity.save()`  

若需将 ETH 与 ERC20 统一标识，可与链上 `ETH_ADDRESS()` 比较（需在 ABI 中包含该函数，并在 mapping 中通过合约模板调用），或仅在子图中存原始 token 地址，由前端解析。

### 3.7 构建与部署

```bash
cd <子图项目根目录>
graph codegen
graph build
graph deploy --studio <subgraph-slug>
```

在 Studio 的 Playground 中用 GraphQL 验证查询。

### 3.8 前端调用

使用 `fetch`、`graphql-request` 等向 Studio 提供的 **GraphQL HTTPS 端点** 发送查询。注意浏览器 **CORS** 与 Studio 对密钥的要求；生产环境常见做法是由后端转发 GraphQL，以官方说明为准。

---

## 4. 发布到去中心化网络（可选）

在 Studio 验证通过后，可按文档将子图 **Publish** 到 The Graph Network（涉及 GRT、版本与策展等）。测试阶段可仅在 Studio 使用。

文档：[Publishing a subgraph](https://thegraph.com/docs/en/subgraphs/developing/publishing/publishing-a-subgraph/)

---

## 5. 自建 Graph Node（可选）

若托管方案不满足需求，可用 Docker 运行 [graph-node](https://github.com/graphprotocol/graph-node)，并连接自有 Sepolia RPC。运维成本更高，但链与部署方式更可控。

---

## 6. 与仓库内 ABI 的对应关系

前端 `multiAssetBankAbi` 中的事件（与链上验证合约应对齐）：

- **`Deposited`**：`token`（indexed）、`user`（indexed）、`amount`  
- **`Withdrawn`**：同上  

子图中为上述两个事件各配置 `eventHandlers` 即可。

---

## 7. 实操检查清单

| 步骤 | 内容 |
|------|------|
| 1 | Etherscan 复制完整 ABI + 合约创建区块号 |
| 2 | Studio 创建子图并复制 Deploy Key |
| 3 | 安装 `graph-cli` 并执行 `graph auth --studio` |
| 4 | `graph init` 生成工程（Sepolia + 合约地址 + ABI） |
| 5 | 编辑 `schema.graphql`、`mapping.ts`，设置 `startBlock` |
| 6 | `graph codegen` → `graph build` → `graph deploy --studio` |
| 7 | Studio Playground 验证后接入前端或后端 |

---

## 8. 后续在本仓库中的建议

- 使用 `graph init` 在本目录生成正式子图代码时，请将 **Deploy Key** 放在环境变量或本地未提交的配置中。  
- 在根目录 `.gitignore` 中忽略子图目录下的 `node_modules/`、`build/` 等（若尚未忽略）。
