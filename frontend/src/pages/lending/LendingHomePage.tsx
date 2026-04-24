import { LENDING_CHAIN_ID, LENDING_CHAIN_NAME, getLendingContractRows } from "@/config/lending";
import { LendingHomeDataSection } from "@/features/lending/LendingHomeDataSection";
import { SectionIntro } from "@/features/codepulse/components";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const BASESCAN = "https://sepolia.basescan.org";

function explorerAddressUrl(address: string): string {
  return `${BASESCAN}/address/${address}`;
}

const ROADMAP_PHASES = [
  {
    phase: "Phase 1",
    title: "发现与透明度",
    items: [
      "本页：目标链、合约角色说明、Basescan 入口（已完成）。",
      "后端（可选）：只读 REST 聚合 reserves / 用户头寸，减轻钱包 RPC 压力。",
      "子图：The Graph 索引 Pool 事件 + 工厂策略创建 + 喂价侧关键事件，供历史流水与仪表盘。",
    ],
  },
  {
    phase: "Phase 2",
    title: "池子核心交互",
    items: [
      "Wagmi + viem：在 Base Sepolia 上调用 Pool（supply / withdraw / borrow / repay / liquidate）。",
      "HybridPriceOracle：按资产组装 priceProofs（stream / feed 路径），失败时明确提示。",
      "健康因子与限额：只读 getUserHealthFactor、reserve caps，写入前客户端预检。",
    ],
  },
  {
    phase: "Phase 3",
    title: "体验与风控",
    items: [
      "多资产仪表盘：每资产 scaled balance、利率曲线说明（链接工厂部署事件）。",
      "EMode / 清算 bonus 等高级参数只读展示；管理员动作单独入口（若协议开放）。",
      "错误与回滚文案对齐合约 revert（如 BorrowCapExceeded、Unhealthy）。",
    ],
  },
] as const;

export function LendingHomePage() {
  const rows = getLendingContractRows();

  return (
    <div className="space-y-10">
      <Alert className="border-primary/25 bg-primary/5">
        <AlertTitle className="text-foreground">目标网络：{LENDING_CHAIN_NAME}（L2）</AlertTitle>
        <AlertDescription className="text-sm leading-relaxed">
          借贷合约部署在 <Badge variant="secondary">chainId {LENDING_CHAIN_ID}</Badge>。与银行 / 众筹 / NFT 使用的{" "}
          <strong className="text-foreground">Ethereum Sepolia（L1）</strong> 隔离。已连接钱包且不在 Base Sepolia 时，下方主内容区会锁定，请用提示按钮切换网络。
        </AlertDescription>
      </Alert>

      <SectionIntro
        eyebrow="Product"
        title="借贷模块规划"
        description="以下三阶段按依赖顺序排列：先可读与可观测，再链上写入闭环，最后增强展示与风控提示。与 subgraph/lending、后端 API 可并行迭代。"
      />

      <LendingHomeDataSection />

      <div className="grid gap-4 md:grid-cols-3">
        {ROADMAP_PHASES.map((p) => (
          <Card key={p.phase} className="glass-card border-white/10">
            <CardHeader className="pb-2">
              <Badge variant="outline" className="mb-2 w-fit text-[10px] tracking-widest">
                {p.phase}
              </Badge>
              <CardTitle className="text-base">{p.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="list-disc space-y-2 pl-4 text-sm text-muted-foreground">
                {p.items.map((t, i) => (
                  <li key={`${p.phase}-${i}`}>{t}</li>
                ))}
              </ul>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="glass-card border-white/10">
        <CardHeader>
          <CardTitle className="text-lg">合约地址清单</CardTitle>
          <CardDescription>
            默认值与仓库 subgraph/lending 中 Base Sepolia 配置一致；可通过{" "}
            <code className="rounded bg-muted px-1 font-mono text-xs">VITE_LENDING_*</code> 环境变量覆盖，见{" "}
            <code className="rounded bg-muted px-1 font-mono text-xs">frontend/.env.example</code>。
          </CardDescription>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[140px]">角色</TableHead>
                <TableHead>说明</TableHead>
                <TableHead className="min-w-[280px]">地址</TableHead>
                <TableHead className="w-[100px]">浏览器</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((r) => (
                <TableRow key={r.key}>
                  <TableCell className="align-top font-medium">{r.label}</TableCell>
                  <TableCell className="align-top text-sm text-muted-foreground">{r.description}</TableCell>
                  <TableCell className="align-top font-mono text-xs text-foreground">{r.address}</TableCell>
                  <TableCell className="align-top">
                    <a
                      href={explorerAddressUrl(r.address)}
                      target="_blank"
                      rel="noreferrer"
                      className="text-sm font-medium text-primary underline-offset-4 hover:underline"
                    >
                      Basescan
                    </a>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
