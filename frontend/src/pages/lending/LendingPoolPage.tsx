import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { SectionIntro } from "@/features/codepulse/components";
import { LENDING_CHAIN_ID, LENDING_CHAIN_NAME, getLendingPoolAddress } from "@/config/lending";

export function LendingPoolPage() {
  const pool = getLendingPoolAddress();

  return (
    <div className="space-y-8">
      <SectionIntro
        eyebrow="Phase 2"
        title="池子交互（开发中）"
        description="将在此页串联 supply / borrow / repay / withdraw / liquidate 与 Hybrid 喂价证明参数；需钱包处于 Base Sepolia 并持有目标资产。"
      />

      <Alert>
        <AlertTitle>尚未接链上表单</AlertTitle>
        <AlertDescription className="text-sm leading-relaxed">
          当前仓库已配置 Pool 地址 <span className="font-mono text-foreground">{pool}</span>（{LENDING_CHAIN_NAME}{" "}
          {LENDING_CHAIN_ID}）。下一步：补充 Pool ABI、价格证明构建流程与健康因子只读展示，再开放写入类交易按钮。
        </AlertDescription>
      </Alert>

      <Card className="glass-card border-white/10">
        <CardHeader>
          <CardTitle className="text-lg">计划中的用户动线</CardTitle>
          <CardDescription>与概览页的路线图 Phase 2 一致，便于产品拆分任务。</CardDescription>
        </CardHeader>
        <CardContent>
          <ol className="list-decimal space-y-2 pl-5 text-sm text-muted-foreground">
            <li>选择储备资产，展示 LTV、清算阈值、存款/借款 APY（只读）。</li>
            <li>供应 / 提现：ERC20 approve + Pool.supply / withdraw（ETH 走原生包装路径若池支持）。</li>
            <li>借款 / 还款：带 borrow index 提示；还款支持部分还。</li>
            <li>清算入口：对不健康头寸调用 liquidate，附带双资产价格证明。</li>
            <li>可选：子图展示最近 Supply/Borrow/Liquidation 流水，与链上校验一致。</li>
          </ol>
        </CardContent>
      </Card>
    </div>
  );
}
