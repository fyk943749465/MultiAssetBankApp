import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import {
  fetchCodePulseCampaigns,
  fetchCodePulseConfig,
  fetchCodePulseProposals,
  fetchCodePulseSummary,
} from "../../features/codepulse/api";
import {
  Callout,
  CampaignCard,
  EmptyState,
  ErrorState,
  InfoGrid,
  LoadingState,
  ProposalCard,
  SectionIntro,
  StatCard,
} from "../../features/codepulse/components";
import { formatDuration, formatWei, shortHash } from "../../features/codepulse/format";
import type { CPConfig, CPSummary, CPCampaign, CPProposal } from "../../features/codepulse/types";

type HomeState = {
  summary: CPSummary;
  config: CPConfig;
  liveCampaigns: CPCampaign[];
  approvedProposals: CPProposal[];
  refundableCampaigns: CPCampaign[];
};

export function CrowdfundingHomePage() {
  const [data, setData] = useState<HomeState | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    (async () => {
      try {
        const [summary, config, liveCampaigns, approvedProposals, refundableCampaigns] = await Promise.all([
          fetchCodePulseSummary(),
          fetchCodePulseConfig(),
          fetchCodePulseCampaigns({ state: "fundraising", page_size: 3, sort: "deadline_at_asc" }),
          fetchCodePulseProposals({ status: "approved", page_size: 3, sort: "submitted_at_desc" }),
          fetchCodePulseCampaigns({ state: "failed_refundable", page_size: 3, sort: "launched_at_desc" }),
        ]);
        if (cancelled) return;
        setData({
          summary,
          config,
          liveCampaigns: liveCampaigns.campaigns,
          approvedProposals: approvedProposals.proposals,
          refundableCampaigns: refundableCampaigns.campaigns,
        });
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, []);

  if (loading && !data) return <LoadingState message="正在加载 Code Pulse 首页…" />;
  if (error && !data) return <ErrorState message={error} />;
  if (!data) return <EmptyState title="暂无数据" description="请确认后端 Code Pulse API 已启动。" />;

  return (
    <div className="space-y-8">
      <Card>
        <CardContent>
          <SectionIntro
            eyebrow="Code Pulse"
            title="开源项目众筹首页"
            description="围绕提案审核、众筹轮次、贡献历史与个人工作台的只读 MVP。当前页面聚合后端 summary、config 与精选列表，方便先完成浏览与发现。"
            action={
              <>
                <Link to="/crowdfunding/explore" className={cn(buttonVariants())}>
                  浏览提案与活动
                </Link>
                <Link to="/crowdfunding/me" className={cn(buttonVariants({ variant: "outline" }))}>
                  我的工作台
                </Link>
              </>
            }
          />
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="提案总数" value={data.summary.proposal_total} hint={`待审核 ${data.summary.pending_review} · 待发起 ${data.summary.approved_waiting}`} />
            <StatCard label="活动总数" value={data.summary.campaign_total} hint={`募集中 ${data.summary.fundraising} · 可退款 ${data.summary.failed}`} />
            <StatCard label="累计募集" value={formatWei(data.summary.total_raised_wei)} hint={`成功活动 ${data.summary.successful}`} />
            <StatCard label="累计退款" value={formatWei(data.summary.total_refunded_wei)} hint="按贡献聚合表统计" />
          </div>
        </CardContent>
      </Card>

      <section>
        <SectionIntro eyebrow="Runtime" title="合约与规则配置" description="这些静态规则会直接影响后续的创建提案、审核与众筹流程。" />
        <InfoGrid
          items={[
            {
              label: "合约地址",
              value: data.config.contract_address ? (
                <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">
                  {shortHash(data.config.contract_address, 8, 6)}
                </code>
              ) : "未配置",
            },
            { label: "合约连接", value: data.config.contract_configured ? "已配置" : "未配置" },
            { label: "Subgraph", value: data.config.subgraph_configured ? "已接入" : "未接入" },
            { label: "里程碑数", value: `${data.config.milestone_num} 个固定阶段` },
            { label: "最小筹资目标", value: formatWei(data.config.min_campaign_target) },
            { label: "最短周期", value: formatDuration(data.config.min_campaign_duration) },
            { label: "最多开发者", value: `${data.config.max_developers_per_campaign} 人` },
            { label: "Stale Funds 延迟", value: formatDuration(data.config.stale_funds_sweep_delay) },
          ]}
        />
      </section>

      {error ? <ErrorState message={error} /> : null}

      <section className="space-y-4">
        <SectionIntro eyebrow="Featured" title="募集中活动" description="按截止时间升序展示，优先看到最需要及时跟进的进行中项目。" />
        {data.liveCampaigns.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {data.liveCampaigns.map((campaign) => (
              <CampaignCard key={campaign.campaign_id} campaign={campaign} />
            ))}
          </div>
        ) : (
          <EmptyState title="暂无募集中活动" description="当前没有 state=fundraising 的众筹轮次。" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Launch Queue" title="已通过、待发起的提案" description="这些提案已经通过审核，但还未进入正式募资轮次。" />
        {data.approvedProposals.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {data.approvedProposals.map((proposal) => (
              <ProposalCard key={proposal.proposal_id} proposal={proposal} />
            ))}
          </div>
        ) : (
          <EmptyState title="暂无待发起提案" description="当前没有 status=approved 的提案。" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Refundable" title="失败并可退款的活动" description="为捐助人提供快速进入退款相关活动详情的入口。" />
        {data.refundableCampaigns.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {data.refundableCampaigns.map((campaign) => (
              <CampaignCard key={campaign.campaign_id} campaign={campaign} compact />
            ))}
          </div>
        ) : (
          <Callout tone="warn" title="当前没有可退款活动" description="当活动最终失败并进入 failed_refundable 状态后，会出现在这里。" />
        )}
      </section>
    </div>
  );
}
