import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAccount } from "wagmi";
import { ActionFormCard } from "../../features/codepulse/action-ui";
import { fetchCodePulseConfig, fetchCodePulseWalletOverview } from "../../features/codepulse/api";
import { Callout, EmptyState, ErrorState, InfoGrid, LoadingState, SectionIntro } from "../../features/codepulse/components";
import { formatDuration, formatWei } from "../../features/codepulse/format";
import type { CPConfig, WalletOverview } from "../../features/codepulse/types";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";

export function CrowdfundingNewProposalPage() {
  const { address, isConnected } = useAccount();
  const [config, setConfig] = useState<CPConfig | null>(null);
  const [overview, setOverview] = useState<WalletOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    (async () => {
      try {
        const cfg = await fetchCodePulseConfig();
        const wallet = isConnected && address ? await fetchCodePulseWalletOverview(address) : null;
        if (cancelled) return;
        setConfig(cfg);
        setOverview(wallet);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [address, isConnected]);

  if (loading && !config) return <LoadingState message="正在加载提案创建配置…" />;
  if (error && !config) return <ErrorState message={error} />;
  if (!config) return <EmptyState title="缺少配置" description="后端 `/api/code-pulse/config` 暂不可用。" />;

  const milestoneLabel = `里程碑描述（每行一条，共 ${config.milestone_num} 条）`;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap gap-2">
        <Link to="/crowdfunding/me" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
          返回我的工作台
        </Link>
        <Link to="/crowdfunding" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
          返回首页
        </Link>
      </div>

      <Card>
        <CardContent>
          <SectionIntro
            eyebrow="New Proposal"
            title="创建开源项目众筹提案"
            description="第二阶段已接入动作预检与交易构建。提交前会先调用 `actions/check`，然后通过 `tx/build` 进行模拟，最后再提交交易。"
          />
          <InfoGrid
            items={[
              { label: "最小目标金额", value: formatWei(config.min_campaign_target) },
              { label: "最短时长", value: formatDuration(config.min_campaign_duration) },
              { label: "里程碑数量", value: `${config.milestone_num} 个` },
              { label: "GitHub URL 长度限制", value: `${config.max_github_url_length}` },
            ]}
          />
        </CardContent>
      </Card>

      {!isConnected || !address ? (
        <div className="space-y-4">
          <Callout tone="warn" title="请先连接钱包" description="请在页面右上角连接钱包。创建提案需要当前钱包地址参与动作预检与后续交易提交。" />
        </div>
      ) : null}

      {overview && !overview.is_proposal_initiator ? (
        <Callout
          tone="warn"
          title="当前地址尚未被授予 proposal_initiator"
          description="你仍然可以填写表单并进行预检；若角色不满足，后端会在 `actions/check` 阶段返回原因。管理员可在 Admin 页面为地址开通提案发起权限。"
        />
      ) : null}

      {error ? <ErrorState message={error} /> : null}

      <ActionFormCard
        title="提交提案"
        action="submit_proposal"
        wallet={address}
        description="字段会根据后端 config 做前端校验，再进入预检 / 模拟 / 提交流程。目标金额以 ETH 输入，前端会自动换算为 wei。"
        fields={[
          { key: "github_url", label: "GitHub URL", kind: "text", required: true, placeholder: "https://github.com/owner/repo" },
          { key: "target", label: "目标金额（ETH）", kind: "eth", required: true, placeholder: "0.1", helpText: `不得低于 ${formatWei(config.min_campaign_target)}` },
          { key: "duration", label: "众筹时长（秒）", kind: "bigint", required: true, placeholder: String(config.min_campaign_duration), helpText: `最少 ${formatDuration(config.min_campaign_duration)}` },
          {
            key: "milestone_descs", label: milestoneLabel, kind: "multiline_list", required: true,
            rows: config.milestone_num + 1,
            placeholder: ["需求分析与排期", "核心功能开发", "测试与发布"].join("\n"),
            helpText: "每行输入一个里程碑描述；不需要填写百分比，合约会使用固定配置。",
          },
        ]}
        validate={(params) => {
          const githubURL = String(params.github_url ?? "").trim();
          const duration = BigInt(String(params.duration ?? "0"));
          const target = BigInt(String(params.target ?? "0"));
          const milestoneDescs = Array.isArray(params.milestone_descs) ? params.milestone_descs : [];

          if (!githubURL) throw new Error("GitHub URL 不能为空");
          if (githubURL.length > config.max_github_url_length) throw new Error(`GitHub URL 不能超过 ${config.max_github_url_length} 个字符`);
          if (target < BigInt(config.min_campaign_target)) throw new Error(`目标金额不能低于 ${formatWei(config.min_campaign_target)}`);
          if (duration < BigInt(config.min_campaign_duration)) throw new Error(`众筹时长不能低于 ${formatDuration(config.min_campaign_duration)}`);
          if (milestoneDescs.length !== config.milestone_num) throw new Error(`里程碑描述必须正好填写 ${config.milestone_num} 条`);
          if (milestoneDescs.some((item) => String(item).trim().length === 0)) throw new Error("里程碑描述不能为空");
        }}
      />
    </div>
  );
}
