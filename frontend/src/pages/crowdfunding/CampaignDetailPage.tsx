import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { NavButton } from "@/components/nav-spa";
import { useAccount } from "wagmi";
import { ActionFormCard } from "../../features/codepulse/action-ui";
import {
  fetchCodePulseCampaignContributions,
  fetchCodePulseCampaignDetail,
  fetchCodePulseCampaignTimeline,
} from "../../features/codepulse/api";
import {
  Callout,
  ContributionTable,
  EmptyState,
  ErrorState,
  InfoGrid,
  LoadingState,
  MilestoneList,
  PaginationControls,
  SectionIntro,
  TimelineList,
} from "../../features/codepulse/components";
import { computeProgressPercent, formatDateTime, formatWei, shortHash, titleCaseStatus } from "../../features/codepulse/format";
import type { CampaignContributionResponse, CampaignDetailResponse, TimelineResponse } from "../../features/codepulse/types";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";

export function CampaignDetailPage() {
  const { campaignId = "" } = useParams();
  const { address, isConnected } = useAccount();
  const [detail, setDetail] = useState<CampaignDetailResponse | null>(null);
  const [timeline, setTimeline] = useState<TimelineResponse | null>(null);
  const [contributions, setContributions] = useState<CampaignContributionResponse | null>(null);
  const [timelinePage, setTimelinePage] = useState(1);
  const [contributionPage, setContributionPage] = useState(1);
  const [loadingDetail, setLoadingDetail] = useState(true);
  const [loadingTimeline, setLoadingTimeline] = useState(true);
  const [loadingContributions, setLoadingContributions] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoadingDetail(true);
    setError(null);
    (async () => {
      try {
        const result = await fetchCodePulseCampaignDetail(campaignId);
        if (cancelled) return;
        setDetail(result);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoadingDetail(false);
      }
    })();
    return () => { cancelled = true; };
  }, [campaignId, refreshKey]);

  useEffect(() => {
    let cancelled = false;
    setLoadingTimeline(true);
    (async () => {
      try {
        const result = await fetchCodePulseCampaignTimeline(campaignId, { page: timelinePage, page_size: 10 });
        if (cancelled) return;
        setTimeline(result);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoadingTimeline(false);
      }
    })();
    return () => { cancelled = true; };
  }, [campaignId, refreshKey, timelinePage]);

  useEffect(() => {
    let cancelled = false;
    setLoadingContributions(true);
    (async () => {
      try {
        const result = await fetchCodePulseCampaignContributions(campaignId, { page: contributionPage, page_size: 10, sort: "amount_desc" });
        if (cancelled) return;
        setContributions(result);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoadingContributions(false);
      }
    })();
    return () => { cancelled = true; };
  }, [campaignId, contributionPage, refreshKey]);

  if (loadingDetail && !detail) return <LoadingState message={`正在加载 Campaign #${campaignId}…`} />;
  if (error && !detail) return <ErrorState message={error} />;
  if (!detail) return <EmptyState title="活动不存在" description="请检查活动编号，或返回探索页重新选择。" />;

  const progress = computeProgressPercent(detail.campaign.amount_raised_wei, detail.campaign.target_wei);
  const refresh = () => setRefreshKey((v) => v + 1);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap gap-2">
        <NavButton
          to="/crowdfunding/explore?view=campaigns"
          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
        >
          返回探索页
        </NavButton>
        <NavButton
          to={`/crowdfunding/proposals/${detail.campaign.proposal_id}`}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
        >
          查看所属提案
        </NavButton>
      </div>

      <Card>
        <CardContent>
          <SectionIntro
            eyebrow={`Campaign #${detail.campaign.campaign_id}`}
            title={detail.campaign.github_url}
            description="活动详情展示募资进度、开发者名单、阶段进展、贡献排行和链上时间线。"
          />

          <div className="mb-6">
            <div className="mb-2 flex items-center justify-between text-sm text-muted-foreground">
              <span>{formatWei(detail.campaign.amount_raised_wei)}</span>
              <span>{progress.toFixed(2)}%</span>
              <span>{formatWei(detail.campaign.target_wei)}</span>
            </div>
            <Progress value={progress} />
          </div>

          <InfoGrid
            items={[
              { label: "状态", value: titleCaseStatus(detail.campaign.state) },
              { label: "所属提案", value: `#${detail.campaign.proposal_id}` },
              { label: "Round", value: detail.campaign.round_index },
              { label: "截止时间", value: formatDateTime(detail.campaign.deadline_at) },
              { label: "发起人", value: <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">{shortHash(detail.campaign.organizer_address)}</code> },
              { label: "捐助者数", value: detail.donor_count },
              { label: "开发者数", value: detail.developers.length },
              { label: "已提取金额", value: formatWei(detail.campaign.total_withdrawn_wei) },
              { label: "未认领退款池", value: formatWei(detail.campaign.unclaimed_refund_pool_wei) },
              { label: "Dormant Sweep", value: detail.campaign.dormant_funds_swept ? "已执行" : "未执行" },
              { label: "Launch 时间", value: formatDateTime(detail.campaign.launched_at) },
              { label: "Launch Tx", value: <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">{shortHash(detail.campaign.launched_tx_hash)}</code> },
            ]}
          />
        </CardContent>
      </Card>

      {error ? <ErrorState message={error} /> : null}

      <section className="space-y-4">
        <SectionIntro eyebrow="Action Console" title="活动动作" description="这里覆盖捐助、退款、结算、开发者管理、里程碑审批与领取等关键交互；按钮的可用性最终以后端 `actions/check` 与 `tx/build` 为准。" />
        {!isConnected || !address ? (
          <Callout tone="warn" title="请先连接钱包" description="活动动作会以当前钱包地址作为角色与状态检查依据，因此执行前需要先连接钱包。" />
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            <ActionFormCard title="Donate to Campaign" action="donate" wallet={address} campaignId={detail.campaign.campaign_id} presetParams={{ campaign_id: String(detail.campaign.campaign_id) }} description="输入捐助金额（ETH），前端会自动换算为 wei。" fields={[{ key: "value", label: "捐助金额（ETH）", kind: "eth", required: true, placeholder: "0.01" }]} onSuccess={() => refresh()} />
            <ActionFormCard title="Finalize Campaign" action="finalize_campaign" wallet={address} campaignId={detail.campaign.campaign_id} presetParams={{ campaign_id: String(detail.campaign.campaign_id) }} description="设计文档要求该动作对所有用户开放，具体是否满足 deadline / state 以后端预检结果为准。" onSuccess={() => refresh()} />
            <ActionFormCard title="Claim Refund" action="claim_refund" wallet={address} campaignId={detail.campaign.campaign_id} presetParams={{ campaign_id: String(detail.campaign.campaign_id) }} description="当活动进入 failed_refundable 且当前钱包是 donor 时，可领取退款。" onSuccess={() => refresh()} />
            <ActionFormCard title="Sweep Stale Funds" action="sweep_stale_funds" wallet={address} campaignId={detail.campaign.campaign_id} presetParams={{ campaign_id: String(detail.campaign.campaign_id) }} description="发起人在满足 stale 条件后可回收长期未认领资金。" onSuccess={() => refresh()} />
            <ActionFormCard title="Add Developer" action="add_developer" wallet={address} campaignId={detail.campaign.campaign_id} presetParams={{ campaign_id: String(detail.campaign.campaign_id) }} fields={[{ key: "account", label: "开发者地址", kind: "address", required: true, placeholder: "0x..." }]} onSuccess={() => refresh()} />
            <ActionFormCard title="Remove Developer" action="remove_developer" wallet={address} campaignId={detail.campaign.campaign_id} presetParams={{ campaign_id: String(detail.campaign.campaign_id) }} fields={[{ key: "account", label: "开发者地址", kind: "address", required: true, placeholder: "0x..." }]} onSuccess={() => refresh()} />
          </div>
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Milestones" title="阶段进度" description="展示本轮活动冻结下来的里程碑及其审批 / 领取状态。" />
        <MilestoneList milestones={detail.milestones} />
        {isConnected && address && detail.milestones.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {detail.milestones.map((milestone) => (
              <div key={`${milestone.campaign_id}-${milestone.milestone_index}`} className="space-y-4">
                <ActionFormCard title={`审批 Milestone ${milestone.milestone_index + 1}`} action="approve_milestone" wallet={address} campaignId={detail.campaign.campaign_id} milestoneIndex={milestone.milestone_index} presetParams={{ campaign_id: String(detail.campaign.campaign_id), milestone_index: String(milestone.milestone_index) }} description="管理员可在 milestone 尚未 approved 时执行审批。" onSuccess={() => refresh()} />
                <ActionFormCard title={`领取 Milestone ${milestone.milestone_index + 1} 份额`} action="claim_milestone_share" wallet={address} campaignId={detail.campaign.campaign_id} milestoneIndex={milestone.milestone_index} presetParams={{ campaign_id: String(detail.campaign.campaign_id), milestone_index: String(milestone.milestone_index) }} description="开发者在 milestone 已审批通过后，可尝试领取对应份额。" onSuccess={() => refresh()} />
              </div>
            ))}
          </div>
        ) : null}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Developers" title="开发者名单" description="当前活动中处于激活状态的开发者。" />
        {detail.developers.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {detail.developers.map((developer) => (
              <Card key={`${developer.campaign_id}-${developer.developer_address}`}>
                <CardContent>
                  <p className="text-sm font-medium text-foreground">{shortHash(developer.developer_address, 8, 6)}</p>
                  <p className="mt-2 text-xs text-muted-foreground">加入时间: {formatDateTime(developer.added_at ?? developer.created_at)}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <EmptyState title="暂无开发者" description="该活动还没有处于激活状态的开发者记录。" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Contributions" title="贡献排行与记录" description="按累计捐助金额倒序展示当前活动的聚合贡献数据。" />
        {loadingContributions && !contributions ? <LoadingState message="正在加载贡献列表…" /> : null}
        {contributions ? (
          <>
            <ContributionTable contributions={contributions.contributions} />
            <PaginationControls pagination={contributions.pagination} onPageChange={setContributionPage} />
          </>
        ) : null}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Timeline" title="活动时间线" description="这里可以看到 launch、donate、refund、milestone approval 等链上事件。" />
        {loadingTimeline && !timeline ? <LoadingState message="正在加载活动时间线…" /> : null}
        {timeline ? (
          <>
            <TimelineList events={timeline.events} />
            <PaginationControls pagination={timeline.pagination} onPageChange={setTimelinePage} />
          </>
        ) : null}
      </section>
    </div>
  );
}
