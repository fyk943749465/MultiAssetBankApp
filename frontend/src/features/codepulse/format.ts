const WEI_DECIMALS = 18n;
const ONE_ETH = 10n ** WEI_DECIMALS;

export function shortHash(value?: string | null, left = 6, right = 4): string {
  if (!value) return "N/A";
  if (value.length <= left + right + 3) return value;
  return `${value.slice(0, left)}…${value.slice(-right)}`;
}

export function titleCaseStatus(value?: string | null): string {
  if (!value) return "unknown";
  return value
    .replace(/_/g, " ")
    .split(" ")
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(" ");
}

export function formatDateTime(value?: string | null): string {
  if (!value) return "N/A";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatDuration(seconds?: number | null): string {
  if (!seconds || seconds <= 0) return "N/A";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0 && hours > 0) return `${days} 天 ${hours} 小时`;
  if (days > 0) return `${days} 天`;
  return `${Math.max(1, Math.floor(seconds / 3600))} 小时`;
}

export function formatWei(wei?: string | null, precision = 4): string {
  if (!wei) return "0 ETH";
  try {
    const raw = BigInt(wei);
    const sign = raw < 0n ? "-" : "";
    const abs = raw < 0n ? -raw : raw;
    const whole = abs / ONE_ETH;
    const fraction = abs % ONE_ETH;
    if (fraction === 0n) return `${sign}${whole.toString()} ETH`;
    const padded = fraction.toString().padStart(Number(WEI_DECIMALS), "0");
    const trimmed = padded.slice(0, precision).replace(/0+$/, "");
    return trimmed ? `${sign}${whole.toString()}.${trimmed} ETH` : `${sign}${whole.toString()} ETH`;
  } catch {
    return `${wei} wei`;
  }
}

export function computeProgressPercent(raisedWei?: string | null, targetWei?: string | null): number {
  try {
    const raised = BigInt(raisedWei ?? "0");
    const target = BigInt(targetWei ?? "0");
    if (target <= 0n) return 0;
    const percent = Number((raised * 10000n) / target) / 100;
    return Math.min(100, Math.max(0, percent));
  } catch {
    return 0;
  }
}

export function formatMilestonePercent(raw?: string | null): string {
  if (!raw) return "N/A";
  const num = Number(raw);
  if (!Number.isFinite(num)) return raw;
  if (num > 100) return `${(num / 100).toFixed(num % 100 === 0 ? 0 : 2)}%`;
  return `${num}%`;
}

export function payloadToText(payload: unknown): string {
  try {
    return JSON.stringify(payload, null, 2);
  } catch {
    return String(payload);
  }
}

export function parseEthToWei(value: string): string {
  const input = value.trim();
  if (!input) {
    throw new Error("请输入金额");
  }
  if (!/^\d+(\.\d+)?$/.test(input)) {
    throw new Error("金额格式不正确");
  }

  const [wholePart, fractionPart = ""] = input.split(".");
  if (fractionPart.length > 18) {
    throw new Error("ETH 最多支持 18 位小数");
  }

  const whole = BigInt(wholePart || "0");
  const fraction = BigInt((fractionPart + "0".repeat(18)).slice(0, 18) || "0");
  return (whole * ONE_ETH + fraction).toString();
}
