#!/usr/bin/env bash
# 在 script/ 目录执行: ./upload-metadata.sh
# 依赖: 已运行 node generate-nft-metadata.js 生成 ./metadata/
set -euo pipefail

cd "$(dirname "$0")"

IRYS_HOST="${IRYS_HOST:-https://devnet.irys.xyz}"
TOKEN="${IRYS_TOKEN:-ethereum}"

if [[ -z "${WALLET_PRIVATE_KEY:-}" ]]; then
  echo "请设置环境变量 WALLET_PRIVATE_KEY（以太坊私钥，勿提交仓库）" >&2
  exit 1
fi

if [[ ! -d ./metadata ]]; then
  echo "未找到 ./metadata，请先执行: node generate-nft-metadata.js" >&2
  exit 1
fi

exec irys upload-dir ./metadata \
  -h "$IRYS_HOST" \
  -t "$TOKEN" \
  -w "$WALLET_PRIVATE_KEY"
