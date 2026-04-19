# 手工：把 `images` 目录一次性上传到 Irys Devnet（Windows）

子图**不会**上传图片；需要本机已安装 **Irys CLI**，终端里能执行 `irys`。

1. 把要上传的所有图片放在某个目录下，例如 **`images`**，且其中必须有 **`1.png`**（作为 `--index-file` 入口；若没有 `1.png`，把参数改成你目录里真实存在的文件名）。
2. 在 **PowerShell** 里先进入该目录的**上一级**（即 `images` 的父目录），例如：

```powershell
cd E:\project\go-chain\script
```

3. 执行下面**一条**命令（把私钥换成你自己的；**不要**把私钥提交到 Git）：

```powershell
irys upload-dir .\images -h https://devnet.irys.xyz -t ethereum -w "0x你的以太坊私钥" --index-file 1.png
```

若私钥已在环境变量里（推荐，避免出现在命令历史里）：

```powershell
$env:WALLET_PRIVATE_KEY="0x你的以太坊私钥"
irys upload-dir .\images -h https://devnet.irys.xyz -t ethereum -w $env:WALLET_PRIVATE_KEY --index-file 1.png
```

`images` 不在 `script` 下时，把 `cd` 改成你的路径，或把 `.\images` 改成绝对路径，例如 `E:\my-nft\images`。

上传成功后，终端会打印 **Manifest / 根地址**（以 CLI 输出为准，常见为 `Uploaded to https://gateway.irys.xyz/<id>`），记下来用于元数据里的 `image` 等字段。

### 已上传图片包根地址（备忘）

以下为本仓库一次上传 **200 张图** 后终端给出的根地址（Devnet / gateway），遗失时可在此查找：

- **根 URL**：<https://gateway.irys.xyz/JOk1n1ztQJHAoLL-57jxRt7cASQ_cX0PbmbgnbHiuU8>
- **单张示例**：`https://gateway.irys.xyz/JOk1n1ztQJHAoLL-57jxRt7cASQ_cX0PbmbgnbHiuU8/1.png`
- **供 `generate-nft-metadata.js` 使用的 Manifest ID**（路径中斜杠后这一段）：`JOk1n1ztQJHAoLL-57jxRt7cASQ_cX0PbmbgnbHiuU8`  
  脚本默认把每条 `image` 写成 `https://gateway.irys.xyz/<ManifestID>/<编号>.png`，与本次上传方式一致。

---

## 批量生成 metadata（一条命令）

在 **`script`** 目录下执行（Manifest ID、个数、名称、描述按你实际情况改；与上文备忘中的 ID 一致时可照抄）：

```powershell
cd E:\project\go-chain\script
node generate-nft-metadata.js JOk1n1ztQJHAoLL-57jxRt7cASQ_cX0PbmbgnbHiuU8 200 "My Pixel Monster" "This is a cool pixel NFT stored on Arweave"
```

前两个参数必填：**Manifest ID**、**NFT 个数**。第三、四个可选：合集名、描述。第五个可选：**图片 URL 前缀**（默认已是 `https://gateway.irys.xyz`，一般不用写）；若你希望 `image` 使用 `https://arweave.net/...`，在命令末尾再加一个参数 `https://arweave.net` 即可。

生成结果在 **`script\metadata\`** 下，文件名为 `1`、`2`、…（无后缀）。

可选：上传 `metadata` 目录仍可用 Irys，例如：

```powershell
irys upload-dir .\metadata -h https://devnet.irys.xyz -t ethereum -w "0x你的以太坊私钥"
```

成功后终端会类似：`Uploaded to https://gateway.irys.xyz/<元数据根ID>`。请把 **整段根 URL**（或至少 `<元数据根ID>`）保存好，下面验证与合约 `baseURI` 都要用到。

---

## 上传 metadata 后如何验证

以下把 **元数据根 URL** 记作 `https://gateway.irys.xyz/<ROOT>`（把 `<ROOT>` 换成你实际上传后得到的那一串，例如 `qyVYPmmDTTTeU1J4Gr6lb_txU4paM2XoRX6jxwYbUHU`）。

### 1. 浏览器

1. 打开 **`https://gateway.irys.xyz/<ROOT>`**  
   应看到 **JSON manifest**（含 `paths` 等），说明根资源已在网关上可访问。
2. 打开单份元数据（与本地无后缀文件名 `1`、`200` 对应）：  
   - `https://gateway.irys.xyz/<ROOT>/1`  
   - `https://gateway.irys.xyz/<ROOT>/200`  
   应返回 **HTTP 200**，正文为带 `name`、`description`、`image`、`attributes` 的 JSON。  
3. 在 JSON 里点开 **`image`** 的链接，应能打开对应 PNG。

若 **`/<ROOT>/1` 打不开**：回到第 1 步 manifest 里找到 `"paths"` → `"1"` → **`id`**，再试 **`https://gateway.irys.xyz/<该id>`**（部分网关对「目录路径」与「直接交易 id」解析不一致）。

### 2. PowerShell + curl（看状态码与正文）

只看 **1** 号元数据是否 **200**：

```powershell
curl.exe -sS -o NUL -w "%{http_code}" "https://gateway.irys.xyz/<ROOT>/1"
```

期望输出 **`200`**。查看 JSON 正文：

```powershell
curl.exe -sS "https://gateway.irys.xyz/<ROOT>/1"
```

### 3. 与合约 `baseURI` / `tokenURI` 对齐（上链后）

若合约里 **`baseURI`** 设为（注意是否带**末尾斜杠**，需与合约实现一致）：

`https://gateway.irys.xyz/<ROOT>/`

则在区块浏览器 **Read Contract** 里调用 **`tokenURI(1)`**，把返回的 URL 复制到浏览器打开，应能打开与上面 **`/1`** 相同的 JSON。

### 4. 说明

个别环境或自动请求工具可能对网关返回 **500**，以你本机 **浏览器直接访问** 为准；若长期异常，可对照 manifest 中的 **`paths`** 与官方 [Irys 文档](https://docs.irys.xyz) 排查。

### 5. MetaMask 提示「This website might be harmful」怎么办？

这是 **MetaMask 扩展**（结合 SEAL、ChainPatrol 等名单）对当前标签页 URL 做的 **钓鱼风险提示**，**不等于** Irys 官方网关一定在作恶；名单里常有 **误报**，尤其是带「连接钱包」联想的新域名或网关类地址。

**你只是只读打开 JSON / 图片时**，可以任选其一，避免被扩展拦在门外：

1. **用没有装 MetaMask 的浏览器**（如系统自带的 Edge、Firefox 单独配置）打开 `https://gateway.irys.xyz/...`。  
2. **Chrome 无痕窗口**里若仍装扩展，可先关掉 MetaMask 再访问，或用 **不带扩展的配置/另一用户配置**。  
3. **只用 PowerShell `curl.exe`** 拉 JSON（见上文 §2），不经过 MetaMask。  
4. 打开根地址 **`https://gateway.irys.xyz/<ROOT>`** 的 manifest，用 **`paths` → `"1"` → `id`** 拼 **`https://arweave.net/<id>`**（或官方文档推荐的其它 Arweave 网关）访问**同一份**上链数据，有时不会被同一规则拦截。

若你确认是**自己上传的合法资源**、希望 MetaMask 以后少误拦，可在拦截页点击 **「report a detection problem」** 向名单方反馈误报；**「Proceed anyway」** 仅在你**确认 URL 来源**（例如来自你自己终端 `Uploaded to` 输出）时使用，不要对陌生链接乱点。

**上链后**：钱包读 `tokenURI` 时也可能遇到同类拦截，处理方式相同；合约里用 `https://arweave.net/<txId>/` 形式的 `baseURI` 有时能避开对 `gateway.irys.xyz` 的名单（需在部署前与元数据实际所在网关一致，并自行测通）。
