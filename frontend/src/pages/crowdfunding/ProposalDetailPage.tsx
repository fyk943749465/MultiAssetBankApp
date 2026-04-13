import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { NavButton } from "@/components/nav-spa";
import { useAccount } from "wagmi";
import { ActionFormCard } from "../../features/codepulse/action-ui";
import { fetchCodePulseProposalDetail, fetchCodePulseProposalTimeline } from "../../features/codepulse/api";
import {
  Callout,
  CampaignCard,
  EmptyState,
  ErrorState,
  InfoGrid,
  LoadingState,
  MilestoneList,
  PaginationControls,
  SectionIntro,
  TimelineList,
} from "../../features/codepulse/components";
import { formatDateTime, formatDuration, formatWei, shortHash, titleCaseStatus } from "../../features/codepulse/format";
import type { CPProposalMilestone, ProposalDetailResponse, TimelineResponse } from "../../features/codepulse/types";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";

export function ProposalDetailPage() {
  const { proposalId = "" } = useParams();
  const { address, isConnected } = useAccount();
  const [detail, setDetail] = useState<ProposalDetailResponse | null>(null);
  const [timeline, setTimeline] = useState<TimelineResponse | null>(null);
  const [timelinePage, setTimelinePage] = useState(1);
  const [loadingDetail, setLoadingDetail] = useState(true);
  const [loadingTimeline, setLoadingTimeline] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoadingDetail(true);
    setError(null);
    (async () => {
      try {
        const result = await fetchCodePulseProposalDetail(proposalId);
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
  }, [proposalId, refreshKey]);

  useEffect(() => {
    let cancelled = false;
    setLoadingTimeline(true);
    (async () => {
      try {
        const result = await fetchCodePulseProposalTimeline(proposalId, { page: timelinePage, page_size: 10 });
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
  }, [proposalId, refreshKey, timelinePage]);

  const milestoneGroups = useMemo(() => {
    const groups = new Map<number, CPProposalMilestone[]>();
    for (const milestone of detail?.milestones ?? []) {
      const list = groups.get(milestone.round_ordinal) ?? [];
      list.push(milestone);
      groups.set(milestone.round_ordinal, list);
    }
    return Array.from(groups.entries()).sort(([a], [b]) => a - b);
  }, [detail?.milestones]);

  if (loadingDetail && !detail) return <LoadingState message={`正在加载 Proposal #${proposalId}…`} />;
  if (error && !detail) return <ErrorState message={error} />;
  if (!detail) return <EmptyState title="提案不存在" description="请检查提案编号，或返回探索页重新选择。" />;

  const { proposal } = detail;
  const refresh = () => setRefreshKey((v) => v + 1);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap gap-2">
        <NavButton
          to="/crowdfunding/explore?view=proposals"
          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
        >
          返回探索页
        </NavButton>
        <NavButton to="/crowdfunding" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
          返回首页
        </NavButton>
      </div>

      <Card>
        <CardContent>
          <SectionIntro
            eyebrow={`Proposal #${proposal.proposal_id}`}
            title={proposal.github_url}
            description="提案详情聚合基础信息、里程碑定义、关联众筹轮次与事件时间线。"
          />
          <InfoGrid
            items={[
              { label: "状态", value: titleCaseStatus(proposal.status) },
              { label: "轮次审核", value: titleCaseStatus(proposal.round_review_state) },
              { label: "目标金额", value: formatWei(proposal.target_wei) },
              { label: "周期", value: formatDuration(proposal.duration_seconds) },
              { label: "发起人", value: <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">{shortHash(proposal.organizer_address)}</code> },
              { label: "当前轮次", value: proposal.current_round_count },
              { label: "提交时间", value: formatDateTime(proposal.submitted_at ?? proposal.created_at) },
              { label: "最后更新时间", value: formatDateTime(proposal.updated_at) },
              { label: "待审核轮次目标", value: proposal.pending_round_target_wei ? formatWei(proposal.pending_round_target_wei) : "无" },
            ]}
          />
        </CardContent>
      </Card>

      {error ? <ErrorState message={error} /> : null}

      <section className="space-y-4">
        <SectionIntro eyebrow="Action Console" title="提案动作" description="第二阶段已接入动作预检与交易构建。实际是否可执行以 `actions/check` 返回为准，能避免前端重复实现链上状态规则。" />
        {!isConnected || !address ? (
          <Callout tone="warn" title="请先连接钱包" description="连接钱包后，才能基于当前地址执行提案审核、轮次提交与发起等动作。" />
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            <ActionFormCard title="管理员通过提案" action="review_proposal" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id), approve: true }} description="适用于 `pending_review` 阶段的管理员审核。" onSuccess={() => refresh()} />
            <ActionFormCard title="管理员拒绝提案" action="review_proposal" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id), approve: false }} description="拒绝后提案会离开待审核阶段。" onSuccess={() => refresh()} />
            <ActionFormCard title="提交首轮 Funding Round 审核" action="submit_first_round_for_review" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id) }} description="适用于提案已通过、但还没有首轮 round 审核记录的阶段。" onSuccess={() => refresh()} />
            <ActionFormCard title="管理员审核 Funding Round" action="review_funding_round" wallet={address} proposalId={proposal.proposal_id} description="管理员审核当前待审批的 funding round。" fields={[{ key: "approve", label: "审核通过", kind: "boolean" }]} presetParams={{ proposal_id: String(proposal.proposal_id), approve: true }} onSuccess={() => refresh()} />
            <ActionFormCard title="发起已批准轮次" action="launch_approved_round" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id) }} description="当 `round_review_state=approved` 时，发起人可把这一轮正式 launch 为 campaign。" onSuccess={() => refresh()} />
            <ActionFormCard
              title="提交 Follow-on Round"
              action="submit_follow_on_round_for_review"
              wallet={address}
              proposalId={proposal.proposal_id}
              description="上一轮结算后，可提交下一轮的目标金额、时长和里程碑描述。金额以 ETH 输入。"
              fields={[
                { key: "target", label: "目标金额（ETH）", kind: "eth", required: true, placeholder: "0.1" },
                { key: "duration", label: "众筹时长（秒）", kind: "bigint", required: true, placeholder: "86400" },
                { key: "milestone_descs", label: "里程碑描述（每行一条）", kind: "multiline_list", required: true, rows: 4, placeholder: ["第二轮需求定义", "第二轮开发", "第二轮验收"].join("\n") },
              ]}
              presetParams={{ proposal_id: String(proposal.proposal_id) }}
              onSuccess={() => refresh()}
            />
          </div>
        )}
        <Callout tone="warn" title="提案通过不等于立即募资" description="设计文档要求在 UI 中明确区分 proposal 审核通过和 funding round 审核通过这两个阶段；即使 proposal 已 approved，在 funding round 审核前也还不能 launch。" />
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Milestones" title="提案里程碑" description="按 round ordinal 分组，展示提案每一轮对应的阶段目标。" />
        {milestoneGroups.length > 0 ? (
          <div className="space-y-4">
            {milestoneGroups.map(([roundOrdinal, milestones]) => (
              <Card key={roundOrdinal}>
                <CardContent>
                  <p className="mb-4 text-sm font-medium text-foreground">Round {roundOrdinal}</p>
                  <MilestoneList milestones={milestones} />
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <EmptyState title="暂无里程碑定义" description="该提案尚未同步到阶段数据。" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Related Campaigns" title="关联众筹轮次" description="展示该提案已发起过的 campaign，便于查看每轮筹资结果。" />
        {detail.campaigns.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {detail.campaigns.map((campaign) => (
              <CampaignCard key={campaign.campaign_id} campaign={campaign} />
            ))}
          </div>
        ) : (
          <EmptyState title="尚未发起任何轮次" description="当提案审核通过并 launch 后，会在这里出现关联 campaign。" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Timeline" title="提案时间线" description="这里汇总与提案相关的提交、审核、轮次发起等事件。" />
        {loadingTimeline && !timeline ? <LoadingState message="正在加载提案时间线…" /> : null}
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
