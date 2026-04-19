/**
 * 批量生成 ERC721 链下 metadata 文件（无 .json 后缀），输出到本脚本同目录下的 metadata/。
 *
 * 用法（在 script 目录执行一条即可）:
 *   node generate-nft-metadata.js <图片ManifestID> <NFT总数> [合集名称] [描述]
 *
 * 示例:
 *   node generate-nft-metadata.js abcdef1234567890 100 "My Pixel Monster" "cool pixel NFT"
 */
const fs = require("fs");
const path = require("path");

const [, , manifestId, totalStr, collectionNameArg, ...descriptionParts] =
  process.argv;

if (!manifestId || !totalStr) {
  console.error(
    "用法: node generate-nft-metadata.js <图片ManifestID> <NFT总数> [合集名称] [描述]",
  );
  process.exit(1);
}

const TOTAL_COUNT = parseInt(totalStr, 10);
const COLLECTION_NAME = collectionNameArg || "My Pixel Monster";
const METADATA_DESCRIPTION =
  descriptionParts.length > 0
    ? descriptionParts.join(" ")
    : "This is a cool pixel NFT stored on Arweave";

if (!Number.isFinite(TOTAL_COUNT) || TOTAL_COUNT < 1) {
  console.error("错误: <NFT总数> 须为正整数。");
  process.exit(1);
}

const METADATA_DIR = path.join(__dirname, "metadata");

if (!fs.existsSync(METADATA_DIR)) {
  fs.mkdirSync(METADATA_DIR, { recursive: true });
}

for (let i = 1; i <= TOTAL_COUNT; i++) {
  const json = {
    name: `${COLLECTION_NAME} #${i}`,
    description: METADATA_DESCRIPTION,
    image: `https://arweave.net/${manifestId}/${i}.png`,
    attributes: [
      { trait_type: "Format", value: "Pixel Art" },
      { trait_type: "Size", value: "36x36" },
    ],
  };
  const outPath = path.join(METADATA_DIR, String(i));
  fs.writeFileSync(outPath, JSON.stringify(json, null, 2), "utf8");
}

console.log(
  `JSON 批量生成完毕: ${TOTAL_COUNT} 个文件 -> ${path.resolve(METADATA_DIR)}`,
);
