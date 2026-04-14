import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { NavButton } from "@/components/nav-spa";
import { useAccount } from "wagmi";
import { ActionFormCard } from "../../features/codepulse/action-ui";
import { fetchCodePulseConfig, fetchCodePulseProposalDetail, fetchCodePulseProposalTimeline } from "../../features/codepulse/api";
import {
  computeProposalFlow,
  isContractOwnerWallet,
  isProposalOrganizer,
  type AdminActionKey,
  type OrganizerActionKey,
} from "../../features/codepulse/proposal-action-flow";
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
import type { CPConfig, CPProposalMilestone, ProposalDetailResponse, TimelineResponse } from "../../features/codepulse/types";
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
  const [cpConfig, setCpConfig] = useState<CPConfig | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchCodePulseConfig()
      .then((c) => {
        if (!cancelled) setCpConfig(c);
      })
      .catch(() => {
        if (!cancelled) setCpConfig(null);
      });
    return () => {
      cancelled = true;
    };
  }, []);

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

  const flow = useMemo(
    () => (detail ? computeProposalFlow(detail.proposal, detail.campaigns) : null),
    [detail],
  );

  if (loadingDetail && !detail) return <LoadingState message={`正在加载 Proposal #${proposalId}…`} />;
  if (error && !detail) return <ErrorState message={error} />;
  if (!detail) return <EmptyState title="提案不存在" description="请检查提案编号，或返回探索页重新选择。" />;

  const { proposal } = detail;
  const refresh = () => setRefreshKey((v) => v + 1);

  const showOrganizer = Boolean(isConnected && address && isProposalOrganizer(address, proposal));
  const showAdmin = Boolean(isConnected && address && isContractOwnerWallet(address, cpConfig?.owner_address));
  const organizerKeys: OrganizerActionKey[] = showOrganizer && flow ? flow.organizerActions : [];
  const adminKeys: AdminActionKey[] = showAdmin && flow ? flow.adminActions : [];
  const adminStepWaiting =
    Boolean(flow && flow.adminActions.length > 0 && adminKeys.length === 0 && isConnected && address);

  const renderOrganizerAction = (key: OrganizerActionKey) => {
    const pid = proposal.proposal_id;
    const common = { wallet: address!, proposalId: pid, onSuccess: refresh };
    switch (key) {
      case "submit_first_round_for_review":
        return (
          <ActionFormCard
            key={key}
            title="提交首轮众筹审核"
            action="submit_first_round_for_review"
            description="将提案中的目标、周期与里程碑正式提交给管理员审核本轮；通过后还需「发起已批准轮次」才会开捐。"
            presetParams={{ proposal_id: String(pid) }}
            {...common}
          />
        );
      case "launch_approved_round":
        return (
          <ActionFormCard
            key={key}
            title="发起已批准轮次（上线众筹）"
            action="launch_approved_round"
            description="管理员已通过本轮参数审核后，发送此交易创建 campaign 并开始接受捐款。"
            presetParams={{ proposal_id: String(pid) }}
            {...common}
          />
        );
      case "submit_follow_on_round_for_review":
        return (
          <ActionFormCard
            key={key}
            title="提交下一轮众筹审核"
            action="submit_follow_on_round_for_review"
            description="上一轮链上已结清后，提交新一轮目标、周期与三条里程碑说明供管理员审核。"
            fields={[
              { key: "target", label: "目标金额（ETH）", kind: "eth", required: true, placeholder: "0.1" },
              { key: "duration", label: "众筹时长（秒）", kind: "bigint", required: true, placeholder: "86400" },
              { key: "milestone_descs", label: "里程碑描述（每行一条）", kind: "multiline_list", required: true, rows: 4, placeholder: ["第二轮需求定义", "第二轮开发", "第二轮验收"].join("\n") },
            ]}
            presetParams={{ proposal_id: String(pid) }}
            {...common}
          />
        );
      default:
        return null;
    }
  };

  const renderAdminAction = (key: AdminActionKey) => {
    const pid = proposal.proposal_id;
    const common = { wallet: address!, proposalId: pid, onSuccess: refresh };
    switch (key) {
      case "review_proposal_approve":
        return (
          <ActionFormCard
            key={key}
            title="通过提案"
            action="review_proposal"
            description="审核通过发起人提交的项目草案（GitHub、目标、周期与里程碑说明）。"
            presetParams={{ proposal_id: String(pid), approve: true }}
            {...common}
          />
        );
      case "review_proposal_reject":
        return (
          <ActionFormCard
            key={key}
            title="拒绝提案"
            action="review_proposal"
            description="拒绝后该提案不再进入后续众筹流程。"
            presetParams={{ proposal_id: String(pid), approve: false }}
            {...common}
          />
        );
      case "review_funding_round":
        return (
          <ActionFormCard
            key={key}
            title="审核本轮众筹参数"
            action="review_funding_round"
            description="通过：发起人可上线本轮；拒绝：清空待审参数，发起人可重新提交。"
            fields={[
              {
                key: "approve",
                label: "审核通过本轮众筹参数",
                kind: "boolean",
                helpText:
                  "默认勾选并发送：本轮审核通过，发起人可上线众筹。若本轮已在链上通过、但发起人尚未发起（未上线），取消勾选并发送：可撤回这次「通过」，回到未通过状态，发起人需重新提交本轮后再审。若仍在待审中，取消勾选并发送：为拒绝本轮。",
              },
            ]}
            presetParams={{ proposal_id: String(pid), approve: true }}
            {...common}
          />
        );
      default:
        return null;
    }
  };

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
        <SectionIntro
          eyebrow="Action Console"
          title="提案动作"
          description="按当前进度只展示与你钱包角色相关的操作；预检与发交易仍以 `actions/check` 与钱包为准。"
        />
        {flow ? (
          <div className="rounded-lg border border-border bg-muted/30 p-4">
            <p className="text-sm font-medium text-foreground">{flow.phaseHeadline}</p>
            <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{flow.phaseDetail}</p>
          </div>
        ) : null}
        {!isConnected || !address ? (
          <Callout tone="warn" title="请先连接钱包" description="连接钱包后，系统才能判断你是发起人还是合约管理员（owner），并显示对应操作。" />
        ) : (
          <div className="space-y-6">
            {adminStepWaiting ? (
              <Callout
                tone="info"
                title="当前步骤：需要管理员"
                description="此阶段需合约 owner 在链上操作。若你正是管理员，请确认后端 `/api/code-pulse/config` 能返回正确的 `owner_address`（与当前钱包一致）后刷新页面。"
              />
            ) : null}
            {organizerKeys.length > 0 ? (
              <div>
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">发起人操作</p>
                <div className="grid gap-4 xl:grid-cols-2">{organizerKeys.map(renderOrganizerAction)}</div>
              </div>
            ) : null}
            {adminKeys.length > 0 ? (
              <div>
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">管理员操作（合约 owner）</p>
                <div className="grid gap-4 xl:grid-cols-2">{adminKeys.map(renderAdminAction)}</div>
              </div>
            ) : null}
            {organizerKeys.length === 0 && adminKeys.length === 0 && !adminStepWaiting ? (
              <Callout
                tone="info"
                title="当前无需你执行提案动作"
                description="你已连接的钱包既不是本提案发起人，也不是当前步骤所需的管理员；或链上进度尚不需要提案页上的交易。仍可浏览下方里程碑与关联众筹。"
              />
            ) : null}
          </div>
        )}
        <Callout tone="warn" title="提案通过不等于立即募资" description="管理员通过提案后，仍需「提交本轮众筹审核 → 管理员审本轮 → 发起人上线」后，捐款入口才会在 campaign 上开放。" />
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
