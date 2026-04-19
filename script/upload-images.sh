#!/usr/bin/env bash
# 在 script/ 目录执行: ./upload-images.sh
# 依赖: 已安装 irys CLI；./images/ 下存在 1.png …（与 --index-file 一致）
set -euo pipefail

cd "$(dirname "$0")"

IRYS_HOST="${IRYS_HOST:-https://devnet.irys.xyz}"
TOKEN="${IRYS_TOKEN:-ethereum}"

if [[ -z "${WALLET_PRIVATE_KEY:-}" ]]; then
  echo "请设置环境变量 WALLET_PRIVATE_KEY（以太坊私钥，勿提交仓库）" >&2
  exit 1
fi

if [[ ! -d ./images ]]; then
  echo "未找到 ./images 目录，请将 1.png、2.png … 放在 script/images/ 下" >&2
  exit 1
fi

exec irys upload-dir ./images \
  -h "$IRYS_HOST" \
  -t "$TOKEN" \
  -w "$WALLET_PRIVATE_KEY" \
  --index-file 1.png
