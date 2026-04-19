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

上传成功后，终端会打印 **Manifest / 根交易 ID**（以 CLI 输出为准），记下来用于元数据里的 `https://arweave.net/<id>/1.png` 等。

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
