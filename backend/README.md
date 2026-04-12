# go-chain backend

HTTP API（Gin）。默认监听地址由环境变量 **`SERVER_ADDR`** 决定，未设置时为 **`:8080`**。

下文以 **`http://127.0.0.1:8080`** 为例；若你改了端口或绑在其它主机上，请替换前缀。

## OpenAPI 与文档页

| 说明 | 地址 |
|------|------|
| OpenAPI JSON（Swagger 2） | `GET http://127.0.0.1:8080/swagger/doc.json` |
| Swagger UI | `http://127.0.0.1:8080/swagger/index.html` |
| ReDoc | `http://127.0.0.1:8080/docs` |
| Scalar | `http://127.0.0.1:8080/scalar` |

## HTTP 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/health` | 健康检查 |
| `GET` | `/api/info` | API 名称与版本 |
| `GET` | `/api/chain/status` | 链连接与 `chain_id`（需配置 `ETH_RPC_URL`） |
| `GET` | `/api/contract/counter/value` | 读 Counter 合约 `get()` |
| `POST` | `/api/contract/counter/count` | 发交易调用 `count()`（无请求体；需私钥等） |
| `GET` | `/api/bank/deposits` | 本地库中的充值记录 |
| `GET` | `/api/bank/withdrawals` | 本地库中的提现记录 |
| `GET` | `/api/bank/subgraph/deposits` | 子图充值（需 `SUBGRAPH_URL`） |
| `GET` | `/api/bank/subgraph/withdrawals` | 子图提现（需 `SUBGRAPH_URL`） |

### 查询参数（节选）

- **`GET /api/bank/deposits`**、**`GET /api/bank/withdrawals`**  
  - `limit`（可选）：默认 `50`，最大 `200`  
  - `user`（可选）：钱包地址 `0x...`，按用户过滤  

- **`GET /api/bank/subgraph/deposits`**、**`GET /api/bank/subgraph/withdrawals`**  
  - `user`（**必填**）：`0x` 前缀地址  
  - `limit`（可选）：默认 `50`，最大 `200`  

## 运行

在项目 **`backend`** 目录下：

```bash
go run ./cmd/server
```

更新 Swagger 注释后重新生成 OpenAPI：

```bash
go generate ./cmd/server/...
```

## 相关环境变量（常用）

| 变量 | 作用 |
|------|------|
| `SERVER_ADDR` | 监听地址，默认 `:8080` |
| `DATABASE_URL` | PostgreSQL（银行索引落库等） |
| `ETH_RPC_URL` | 以太坊 JSON-RPC |
| `COUNTER_CONTRACT_ADDRESS` | Counter 合约地址 |
| `ETH_PRIVATE_KEY` | 写链交易用私钥（勿提交仓库） |
| `BANK_CONTRACT_ADDRESS` | MultiAssetBank 合约（启动银行索引） |
| `SUBGRAPH_URL` | The Graph 查询 URL |
| `SUBGRAPH_API_KEY` | 子图 API Key（若托管方需要） |
