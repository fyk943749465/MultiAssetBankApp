import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { NavButton } from "@/components/nav-spa";
import { useAccount, useChainId } from "wagmi";
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

/** 路由里可能是缺失、JS undefined 被拼进 URL 的字面量 "undefined"、或非数字。 */
function normalizeCampaignRouteId(raw: string | undefined): string {
  const s = (raw ?? "").trim();
  if (!s || s === "undefined" || !/^\d+$/.test(s)) {
    return "";
  }
  return s;
}

export function CampaignDetailPage() {
  const params = useParams();
  const campaignId = useMemo(
    () => normalizeCampaignRouteId(params.campaignId),
    [params.campaignId],
  );
  const { address, isConnected } = useAccount();
  const chainId = useChainId();
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
    if (!campaignId) {
      setLoadingDetail(false);
      setDetail(null);
      setError(null);
      return;
    }
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
    if (!campaignId) {
      setLoadingTimeline(false);
      setTimeline(null);
      return;
    }
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
    if (!campaignId) {
      setLoadingContributions(false);
      setContributions(null);
      return;
    }
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

  if (!campaignId) {
    return (
      <div className="space-y-4">
        <NavButton
          to="/crowdfunding/explore?view=campaigns"
          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
        >
          返回探索页
        </NavButton>
        <EmptyState
          title="无效的活动链接"
          description="活动编号缺失、不是数字，或链接里出现了 “undefined”。请从探索页或列表重新进入活动详情。"
        />
      </div>
    );
  }

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
        <SectionIntro
          eyebrow="Action Console"
          title="活动动作"
          description="每张卡片内有该动作的用途、谁可以操作、以及大致何时可用。实际能否发送交易，以「预检并构建」与链上模拟为准（界面数据可能与链上或数据库有几分钟延迟）。"
        />
        {!isConnected || !address ? (
          <Callout tone="warn" title="请先连接钱包" description="活动动作会以当前钱包地址作为角色与状态检查依据，因此执行前需要先连接钱包。" />
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            <ActionFormCard
              title="向本轮活动捐助（Donate）"
              action="donate"
              wallet={address}
              campaignId={detail.campaign.campaign_id}
              presetParams={{ campaign_id: String(detail.campaign.campaign_id) }}
              description={`在链上执行 donate：向本活动合约转入 ETH 作为捐助；前端会把您填写的 ETH 数额换算为 wei 写入交易。通常仅在活动处于「募集中（fundraising）」时可成功；本页当前状态为「${titleCaseStatus(detail.campaign.state)}」。任何人都可以捐助，不要求您是发起人或管理员。截止时间 ${formatDateTime(detail.campaign.deadline_at)} 前一般仍可捐助（具体以合约与预检为准）。除 gas 外，交易需附带您输入的 ETH 作为捐款金额。`}
              fields={[{ key: "value", label: "捐助金额（ETH）", kind: "eth", required: true, placeholder: "0.01" }]}
              onSuccess={() => refresh()}
            />
            <ActionFormCard
              title="结算本轮众筹（Finalize）"
              action="finalize_campaign"
              wallet={address}
              campaignId={detail.campaign.campaign_id}
              presetParams={{ campaign_id: String(detail.campaign.campaign_id) }}
              description={`在链上执行 finalizeCampaign：在众筹「截止时间已过」且合约仍允许结算时，为本轮活动做正式收尾。结算后，若达到募集目标会进入成功路径（后续可走里程碑审批、开发者领取等）；若未达标会进入失败可退款路径（捐款人可使用「领取退款」）。合约上该函数对调用者无身份限制，任意地址均可发起，避免截止后无人操作导致活动一直卡在募集中。本活动截止时间：${formatDateTime(detail.campaign.deadline_at)}。是否已到可结算时点、以及链上是否允许，请点击「预检并构建」以服务端与模拟结果为准；未到截止时间时预检通常会失败。发起交易需支付 gas。`}
              onSuccess={() => refresh()}
            />
            <ActionFormCard
              title="领取失败项目的退款（Claim Refund）"
              action="claim_refund"
              wallet={address}
              campaignId={detail.campaign.campaign_id}
              presetParams={{ campaign_id: String(detail.campaign.campaign_id) }}
              description="在链上执行 claimRefund：当本轮众筹未达标且活动已进入「失败可退款（failed_refundable）」等可退款状态时，将您此前捐助中可退回的部分取回当前钱包。预检要求：当前钱包须被识别为本活动的捐款人（donor）——后端通常依据库中贡献记录判断；若您刚在链上捐款而库尚未同步，预检可能暂时失败。仅当活动状态允许退款时才能成功；请以「预检并构建」为准。发起交易需支付 gas。"
              onSuccess={() => refresh()}
            />
            <ActionFormCard
              title="清扫沉睡资金（Sweep Stale Funds）"
              action="sweep_stale_funds"
              wallet={address}
              campaignId={detail.campaign.campaign_id}
              presetParams={{ campaign_id: String(detail.campaign.campaign_id) }}
              description="在链上执行与「长期无人认领退款 / 沉睡资金」相关的清扫逻辑（合约中的 sweepStaleFunds）：一般由本活动发起人（organizer）在众筹已成功、且满足合约规定的宽限或超时等条件后发起，用于将仍滞留在合约内、无人认领的退款池资金按规定回收（去向以合约为准），避免资金长期锁定。您是否满足条件、是否为发起人，以「预检并构建」与链上模拟为准。发起交易需支付 gas。"
              onSuccess={() => refresh()}
            />
            <ActionFormCard
              title="添加开发者（Add Developer）"
              action="add_developer"
              wallet={address}
              campaignId={detail.campaign.campaign_id}
              presetParams={{ campaign_id: String(detail.campaign.campaign_id) }}
              description="在链上把某个地址加入本活动的开发者名单：开发者可在后续里程碑审批通过后，按规则领取对应阶段份额。仅本活动发起人（organizer）可添加；请填写对方完整钱包地址（0x…）。同一地址重复添加、非发起人调用、或活动状态不允许时，预检或链上会失败。添加成功后对方仍须在钱包侧配合后续「领取份额」等操作。发起交易需支付 gas。"
              fields={[{ key: "account", label: "开发者地址", kind: "address", required: true, placeholder: "0x..." }]}
              onSuccess={() => refresh()}
            />
            <ActionFormCard
              title="移除开发者（Remove Developer）"
              action="remove_developer"
              wallet={address}
              campaignId={detail.campaign.campaign_id}
              presetParams={{ campaign_id: String(detail.campaign.campaign_id) }}
              description="在链上将某个地址从本活动开发者名单中标记为移除（不再作为活跃开发者参与后续份额逻辑）。仅发起人（organizer）可操作。请填写要移除的开发者地址。若该地址本就不在名单中、或移除违反合约约束（例如仍有未结清阶段），预检或链上会失败。发起交易需支付 gas。"
              fields={[{ key: "account", label: "开发者地址", kind: "address", required: true, placeholder: "0x..." }]}
              onSuccess={() => refresh()}
            />
          </div>
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro
          eyebrow="Milestones"
          title="阶段进度"
          description="以下为众筹成功后按阶段释放的里程碑：管理员对某阶段点「审批」后，该阶段才允许对应开发者「领取份额」。每张卡片内有链上动作说明与角色要求。"
        />
        <MilestoneList milestones={detail.milestones} />
        {isConnected && address && detail.milestones.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {detail.milestones.map((milestone) => {
              const msLabel = `第 ${milestone.milestone_index + 1} 阶段`;
              const descTrim = milestone.description.trim();
              let msSnippet = "";
              if (descTrim) {
                const cut = descTrim.slice(0, 120);
                const more = descTrim.length > 120;
                msSnippet = `本阶段说明（节选）：${cut}${more ? "…" : ""} `;
              }
              return (
              <div key={`${milestone.campaign_id}-${milestone.milestone_index}`} className="space-y-4">
                <ActionFormCard
                  title={`审批里程碑（${msLabel}）`}
                  action="approve_milestone"
                  wallet={address}
                  campaignId={detail.campaign.campaign_id}
                  milestoneIndex={milestone.milestone_index}
                  presetParams={{ campaign_id: String(detail.campaign.campaign_id), milestone_index: String(milestone.milestone_index) }}
                  description={`在链上执行 approveMilestone：由合约管理员（admin）将「${msLabel}」标记为已审批，表示该阶段目标与解锁条件已被认可；审批通过后，该阶段才允许被列入开发者领取流程（具体顺序与比例以合约为准）。仅当该阶段在库中尚未标记为已审批时，预检才会通过；重复审批会失败。${msSnippet}预检会校验当前钱包是否为管理员角色。发起交易需支付 gas。`}
                  onSuccess={() => refresh()}
                />
                <ActionFormCard
                  title={`领取里程碑份额（${msLabel}）`}
                  action="claim_milestone_share"
                  wallet={address}
                  campaignId={detail.campaign.campaign_id}
                  milestoneIndex={milestone.milestone_index}
                  presetParams={{ campaign_id: String(detail.campaign.campaign_id), milestone_index: String(milestone.milestone_index) }}
                  description={`在链上执行 claimMilestoneShare：由本活动开发者名单中的开发者，在「${msLabel}」已被管理员审批通过、且满足合约中的时间或状态条件后，领取该阶段对应的本金或份额。预检要求：当前钱包须为在本活动中处于激活状态的开发者（developer）。若该阶段尚未审批、或未到可领取窗口、或您不是开发者，预检或链上会失败。${msSnippet}发起交易需支付 gas。`}
                  onSuccess={() => refresh()}
                />
              </div>
              );
            })}
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
        <SectionIntro eyebrow="Contributions" title="贡献排行与记录" description="子图可用时仅展示子图 Donated；子图不可用时再从数据库事件流水读取。按每一笔捐助展示，不做地址聚合。" />
        {loadingContributions && !contributions ? <LoadingState message="正在加载贡献列表…" /> : null}
        {contributions ? (
          <>
            {contributions.data_source ? (
              <p className="text-xs text-muted-foreground">
                数据来源：<span className="font-mono">{contributions.data_source}</span>
              </p>
            ) : null}
            <ContributionTable contributions={contributions.contributions} chainId={chainId} />
            <PaginationControls pagination={contributions.pagination} onPageChange={setContributionPage} />
          </>
        ) : null}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Timeline" title="活动时间线" description="子图可用时仅展示子图事件；子图不可用时再从 cp_event_log 读取。含 launch、donate、refund、milestone 等。" />
        {loadingTimeline && !timeline ? <LoadingState message="正在加载活动时间线…" /> : null}
        {timeline ? (
          <>
            {timeline.data_source ? (
              <p className="text-xs text-muted-foreground">
                数据来源：<span className="font-mono">{timeline.data_source}</span>
              </p>
            ) : null}
            <TimelineList events={timeline.events} />
            <PaginationControls pagination={timeline.pagination} onPageChange={setTimelinePage} />
          </>
        ) : null}
      </section>
    </div>
  );
}
