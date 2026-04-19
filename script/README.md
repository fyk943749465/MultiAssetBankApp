# NFT 图片与元数据：Irys Devnet 上传说明

本目录脚本用于 **把 NFT 图片和 JSON 元数据上传到 Irys（原 Bundlr）Devnet**，付费代币类型为 **`-t ethereum`**（使用以太坊钱包私钥支付 Devnet 存储费）。上传完成后，你会得到 **图片 Manifest ID** 与 **元数据 Manifest ID**，用于合约 `setBaseURI` 或前端拼接 `tokenURI`。

**安全提示**

- **永远不要**把真实私钥写进仓库或提交到 Git。请用环境变量（如 `WALLET_PRIVATE_KEY`），并确保 `script/.env` 已加入 `.gitignore`（本目录已忽略）。
- 以下命令中的 `-w` 仅读取环境变量，不要把私钥贴在聊天或 PR 里。

**前置条件**

1. 已安装 [Node.js](https://nodejs.org/)（用于生成元数据 JSON）。
2. 已安装 **Irys CLI**，且终端中能执行 `irys`（安装方式以 [Irys 官方文档](https://docs.irys.xyz) 为准）。
3. 准备目录 **`images/`**：内含 `1.png`、`2.png`、…、`N.png`（与下面 `TOTAL_COUNT` 一致）。首图 **`1.png`** 会作为目录索引的入口（`--index-file 1.png`）。

建议在 **`script/` 目录下** 放置 `images/`、`metadata/`，与下文命令一致；也可自行改路径。若本地素材在 **`pic/`** 下且命名已是 `1.png`…，可复制到 `images/`：`cp -r pic images`（Windows 可用资源管理器复制后把文件夹改名为 `images`）。

---

## 步骤一：上传图片目录到 Devnet

在 **`script/`** 目录执行（或先 `cd script`）：

```bash
export WALLET_PRIVATE_KEY="0x你的以太坊私钥"   # 勿提交仓库

irys upload-dir ./images \
  -h https://devnet.irys.xyz \
  -t ethereum \
  -w "$WALLET_PRIVATE_KEY" \
  --index-file 1.png
```

命令成功后会输出 **Manifest ID**（或交易/资源 ID，以 CLI 实际提示为准）。请记下 **图片 Manifest ID**，下一步生成 JSON 要用。

也可直接运行（需已 `chmod +x`）：

```bash
./upload-images.sh
```

脚本内通过环境变量读取私钥与节点地址，见 `upload-images.sh` 注释。

---

## 步骤二：批量生成本地元数据文件

1. 将上一步得到的 **图片 Manifest ID** 写入环境变量 **`IMG_MANIFEST_ID`**（或编辑 `generate-nft-metadata.js` 顶部的默认值，不推荐提交含真实 ID 的修改）。
2. 按需设置 **`TOTAL_COUNT`**、**`COLLECTION_NAME`**、**`METADATA_DESCRIPTION`** 等。

在 **`script/`** 目录执行：

```bash
export IMG_MANIFEST_ID="上一步得到的Manifest_ID"
export TOTAL_COUNT=100
export COLLECTION_NAME="My Pixel Monster"
export METADATA_DESCRIPTION="This is a cool pixel NFT stored on Arweave"

node generate-nft-metadata.js
```

会在 **`./metadata/`** 下生成名为 `1`、`2`、…、`N` 的文件（**无 `.json` 后缀**，便于合约里 `baseURI + tokenId` 直接拼接）。

每条元数据里的图片 URL 形如：

`https://arweave.net/${IMG_MANIFEST_ID}/${i}.png`

若你实际上传路径与 `i.png` 命名不一致，请自行修改 `generate-nft-metadata.js` 中的 `image` 字段逻辑。

---

## 步骤三：上传元数据目录到 Devnet

确认 **`metadata/`** 已生成后，在 **`script/`** 目录执行：

```bash
export WALLET_PRIVATE_KEY="0x你的以太坊私钥"

irys upload-dir ./metadata \
  -h https://devnet.irys.xyz \
  -t ethereum \
  -w "$WALLET_PRIVATE_KEY"
```

记下 **元数据 Manifest ID**。合约 `baseURI` 通常设为：

`https://arweave.net/<元数据Manifest_ID>/`

这样 `tokenURI(tokenId)` 可解析为 `.../<tokenId>` 对应的无后缀 JSON 文件。

也可运行：

```bash
./upload-metadata.sh
```

---

## 文件说明

| 文件 | 作用 |
|------|------|
| `upload-images.sh` | 封装「上传 `images/`」的 `irys upload-dir` 调用。 |
| `generate-nft-metadata.js` | 根据 `IMG_MANIFEST_ID` 等批量写入 `metadata/1` … `metadata/N`。 |
| `upload-metadata.sh` | 封装「上传 `metadata/`」的 `irys upload-dir` 调用。 |

Windows 用户若无 Bash，可在 **Git Bash** / **WSL** 中执行 `.sh`，或把脚本内 `irys` 那一行复制到 PowerShell 中手动替换变量执行。

---

## 与子图 / 合约的关系

- 链上 **ERC721 `tokenURI`** 只保存「指向元数据的 URL」；**子图**索引的是链上事件（如 `BaseURIUpdated`、Transfer 等），**不会**替你上传图片或 JSON。
- 上链资源与 **Sepolia 工厂 / 模板** 的配置流程，还可对照仓库内 [`docs/NFT-Platform-Architecture-PG-Subgraph.md`](../docs/NFT-Platform-Architecture-PG-Subgraph.md)。
