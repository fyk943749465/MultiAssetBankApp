/** 将 tokenURI / metadata.image 转为浏览器可请求的 URL（IPFS、Arweave、data:）。 */

export function httpUrlFromUri(uri: string): string {
  const u = uri.trim();
  if (u.startsWith("ipfs://")) {
    const path = u.slice("ipfs://".length).replace(/^ipfs\//, "");
    return `https://ipfs.io/ipfs/${path}`;
  }
  if (u.startsWith("ipfs/")) {
    return `https://ipfs.io/${u}`;
  }
  if (u.startsWith("ar://")) {
    return `https://arweave.net/${u.slice("ar://".length)}`;
  }
  return u;
}

function parseDataJson(tokenURI: string): unknown {
  if (tokenURI.startsWith("data:application/json;base64,")) {
    const b64 = tokenURI.slice("data:application/json;base64,".length);
    const json = atob(b64);
    return JSON.parse(json) as unknown;
  }
  if (tokenURI.startsWith("data:application/json,")) {
    const raw = decodeURIComponent(tokenURI.slice("data:application/json,".length));
    return JSON.parse(raw) as unknown;
  }
  throw new Error("unsupported data: scheme for metadata");
}

export async function fetchMetadataFromTokenUri(tokenURI: string): Promise<{ name?: string; image?: string }> {
  const trimmed = tokenURI.trim();
  if (trimmed.startsWith("data:application/json")) {
    const j = parseDataJson(trimmed) as Record<string, unknown>;
    return {
      name: typeof j.name === "string" ? j.name : undefined,
      image: typeof j.image === "string" ? j.image : undefined,
    };
  }
  const url = httpUrlFromUri(trimmed);
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`metadata HTTP ${res.status}`);
  }
  const j = (await res.json()) as Record<string, unknown>;
  return {
    name: typeof j.name === "string" ? j.name : undefined,
    image: typeof j.image === "string" ? j.image : undefined,
  };
}

export function imageUrlFromMetadataField(imageField: string): string {
  return httpUrlFromUri(imageField.trim());
}
