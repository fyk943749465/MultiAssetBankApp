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

---

可选：仓库里的 `generate-nft-metadata.js`、`upload-metadata.sh` 仍可用于批量生成 JSON 与上传 `metadata`；需要时再自行使用。
