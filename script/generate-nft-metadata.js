/**
 * 根据 Irys 上传图片后得到的 Manifest ID，批量生成 ERC721 常用的链下 metadata 文件。
 * 在 script/ 目录执行: node generate-nft-metadata.js
 *
 * 环境变量:
 *   IMG_MANIFEST_ID  (必填)  步骤一 irys 上传 images 后得到的 Manifest ID
 *   TOTAL_COUNT        (可选)  默认 100
 *   COLLECTION_NAME    (可选)  默认 My Pixel Monster
 *   METADATA_DESCRIPTION (可选) 默认描述文案
 *   METADATA_DIR       (可选)  输出目录，默认 ./metadata
 */
const fs = require("fs");
const path = require("path");

const IMG_MANIFEST_ID = process.env.IMG_MANIFEST_ID || "";
const TOTAL_COUNT = parseInt(process.env.TOTAL_COUNT || "100", 10);
const COLLECTION_NAME = process.env.COLLECTION_NAME || "My Pixel Monster";
const METADATA_DESCRIPTION =
  process.env.METADATA_DESCRIPTION ||
  "This is a cool pixel NFT stored on Arweave";
const METADATA_DIR = process.env.METADATA_DIR || path.join(__dirname, "metadata");

if (!IMG_MANIFEST_ID) {
  console.error(
    "错误: 请设置环境变量 IMG_MANIFEST_ID（上传 images 后得到的 Manifest ID）。",
  );
  process.exit(1);
}

if (!Number.isFinite(TOTAL_COUNT) || TOTAL_COUNT < 1) {
  console.error("错误: TOTAL_COUNT 须为正整数。");
  process.exit(1);
}

if (!fs.existsSync(METADATA_DIR)) {
  fs.mkdirSync(METADATA_DIR, { recursive: true });
}

for (let i = 1; i <= TOTAL_COUNT; i++) {
  const json = {
    name: `${COLLECTION_NAME} #${i}`,
    description: METADATA_DESCRIPTION,
    image: `https://arweave.net/${IMG_MANIFEST_ID}/${i}.png`,
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
