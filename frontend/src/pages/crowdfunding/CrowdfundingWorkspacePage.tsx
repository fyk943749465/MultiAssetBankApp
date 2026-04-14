import { useEffect, useState } from "react";
import { NavButton, RoutePressable } from "@/components/nav-spa";
import { useAccount } from "wagmi";
import { ActionFormCard } from "../../features/codepulse/action-ui";
import {
  fetchCodePulseContributorDashboard,
  fetchCodePulseDeveloperDashboard,
  fetchCodePulseInitiatorDashboard,
  fetchCodePulseWalletOverview,
} from "../../features/codepulse/api";
import {
  CampaignCard,
  Callout,
  EmptyState,
  ErrorState,
  LoadingState,
  ProposalCard,
  SectionIntro,
  SmallMetaPill,
  StatCard,
  StatusPill,
} from "../../features/codepulse/components";
import { formatDateTime, formatWei, shortHash } from "../../features/codepulse/format";
import type { ContributorDashboard, DeveloperDashboard, InitiatorDashboard, WalletOverview } from "../../features/codepulse/types";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type WorkspaceState = {
  overview: WalletOverview;
  initiator: InitiatorDashboard | null;
  contributor: ContributorDashboard | null;
  developer: DeveloperDashboard | null;
};

function DashboardGroup({ title, children }: { readonly title: string; readonly children: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{title}</p>
      {children}
    </div>
  );
}

export function CrowdfundingWorkspacePage() {
  const { address, isConnected } = useAccount();
  const [data, setData] = useState<WorkspaceState | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    if (!isConnected || !address) { setData(null); setLoading(false); setError(null); return; }

    let cancelled = false;
    setLoading(true);
    setError(null);

    (async () => {
      try {
        const overview = await fetchCodePulseWalletOverview(address);
        if (cancelled) return;
        const [initiator, contributor, developer] = await Promise.all([
          overview.available_dashboards.includes("initiator") ? fetchCodePulseInitiatorDashboard(address) : Promise.resolve(null),
          overview.available_dashboards.includes("contributor") ? fetchCodePulseContributorDashboard(address) : Promise.resolve(null),
          overview.available_dashboards.includes("developer") ? fetchCodePulseDeveloperDashboard(address) : Promise.resolve(null),
        ]);
        if (cancelled) return;
        setData({ overview, initiator, contributor, developer });
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [address, isConnected, refreshKey]);

  if (!isConnected || !address) {
    return (
      <div className="space-y-6">
        <Callout tone="warn" title="请先连接钱包" description="请在页面右上角连接钱包。`/crowdfunding/me` 会根据当前钱包地址查询角色和工作台数据。连接后即可看到你的提案、参与记录和开发者视角。" />
      </div>
    );
  }

  if (loading && !data) return <LoadingState message="正在加载你的 Code Pulse 工作台…" />;
  if (error && !data) return <ErrorState message={error} />;
  if (!data) return <EmptyState title="暂无工作台数据" description="当前地址没有可展示的工作台信息。" />;

  const roles = Array.from(new Set(data.overview.roles.map((role) => role.role)));

  return (
    <div className="space-y-6">
      <Card>
        <CardContent>
          <SectionIntro
            eyebrow="My Workspace"
            title="我的 Code Pulse 工作台"
            description="根据当前连接的钱包地址，组合展示 initiator / contributor / developer 的只读视图。"
            action={
              <>
                <NavButton to="/crowdfunding/me/proposals/new" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
                  新建提案
                </NavButton>
                {data.overview.is_admin ? (
                  <NavButton to="/crowdfunding/admin" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
                    Admin
                  </NavButton>
                ) : null}
              </>
            }
          />
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">{address}</code>
            {roles.map((role) => <StatusPill key={role} status={role} />)}
            {roles.length === 0 ? <span className="text-sm text-muted-foreground">当前没有激活角色</span> : null}
          </div>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="提案数" value={data.overview.proposal_count} />
            <StatCard label="发起活动数" value={data.overview.campaign_as_organizer_count} />
            <StatCard label="参与捐助数" value={data.overview.donation_count} />
            <StatCard label="开发者活动数" value={data.overview.developer_campaign_count} />
          </div>
        </CardContent>
      </Card>

      {error ? <ErrorState message={error} /> : null}

      {data.overview.is_admin ? (
        <Callout title="检测到 Admin 角色" description="当前地址具备 admin 权限。第二阶段已开放独立的 `/crowdfunding/admin` 管理台，可继续审核提案、审批里程碑和管理平台资金。" />
      ) : null}

      <section className="space-y-4">
        <SectionIntro eyebrow="Quick Actions" title="全局快捷动作" description="工作台提供与角色无关或高频的动作入口，具体是否允许执行仍以后端预检为准。" />
        <div className="grid gap-4 xl:grid-cols-2">
          <ActionFormCard
            title="Donate to Platform"
            action="donate_to_platform"
            wallet={address}
            description="任何用户都可以向平台捐赠；金额以 ETH 输入。"
            fields={[{ key: "value", label: "捐赠金额（ETH）", kind: "eth", required: true, placeholder: "0.01" }]}
            onSuccess={() => setRefreshKey((v) => v + 1)}
          />
          {data.overview.is_admin ? (
            <ActionFormCard
              title="Pause / Unpause 可在 Admin 执行"
              action="pause"
              wallet={address}
              description="工作台保留一个管理员快捷入口；更多管理动作请进入 Admin 页面。"
              onSuccess={() => setRefreshKey((v) => v + 1)}
            />
          ) : (
            <Callout title="创建提案入口" description="如果你已被授予 proposal_initiator，可直接进入「新建提案」页面完成 submit_proposal 流程。" />
          )}
        </div>
      </section>

      {data.initiator ? (
        <section className="space-y-4">
          {data.initiator.view_data_source === "subgraph" ? (
            <Callout
              title="发起人视角：只读数据以子图为准"
              description="提案分组、状态与「募资中」活动列表按子图事件推导，与链上展示一致。执行预检、构建与发送交易时仍以 PostgreSQL 为准；若动作提示状态不符，请等待 RPC 索引同步或稍后重试。"
            />
          ) : null}
          <SectionIntro eyebrow="Initiator" title="发起人视角" description="按审核与轮次状态分组展示你的提案，并单独列出正在募资的活动。" />
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="提案总数" value={data.initiator.proposals_total} />
            <StatCard label="活动总数" value={data.initiator.campaigns_total} />
            <StatCard label="待审核提案" value={data.initiator.pending_review.length} />
            <StatCard label="募资中活动" value={data.initiator.fundraising_campaigns.length} />
          </div>
          <DashboardGroup title="待审核提案">
            {data.initiator.pending_review.length > 0 ? data.initiator.pending_review.map((p) => <ProposalCard key={p.proposal_id} proposal={p} compact />) : <EmptyState title="没有待审核提案" />}
          </DashboardGroup>
          <DashboardGroup title="已通过待发起">
            {data.initiator.approved_waiting.length > 0 ? data.initiator.approved_waiting.map((p) => <ProposalCard key={p.proposal_id} proposal={p} compact />) : <EmptyState title="没有待发起提案" />}
          </DashboardGroup>
          <DashboardGroup title="轮次审核中 / 已通过">
            <div className="grid gap-4 xl:grid-cols-2">
              <div className="space-y-3">
                {data.initiator.round_review_pending.length > 0 ? data.initiator.round_review_pending.map((p) => <ProposalCard key={p.proposal_id} proposal={p} compact />) : <EmptyState title="没有待审核轮次" />}
              </div>
              <div className="space-y-3">
                {data.initiator.round_review_approved.length > 0 ? data.initiator.round_review_approved.map((p) => <ProposalCard key={p.proposal_id} proposal={p} compact />) : <EmptyState title="没有待 launch 轮次" />}
              </div>
            </div>
          </DashboardGroup>
          <DashboardGroup title="被拒绝 / 已结算">
            <div className="grid gap-4 xl:grid-cols-2">
              <div className="space-y-3">
                {data.initiator.rejected.length > 0 ? data.initiator.rejected.map((p) => <ProposalCard key={p.proposal_id} proposal={p} compact />) : <EmptyState title="没有被拒绝提案" />}
              </div>
              <div className="space-y-3">
                {data.initiator.settled_can_follow_on.length > 0 ? data.initiator.settled_can_follow_on.map((p) => <ProposalCard key={p.proposal_id} proposal={p} compact />) : <EmptyState title="没有可 follow-on 的提案" />}
              </div>
            </div>
          </DashboardGroup>
          <DashboardGroup title="募资中活动">
            {data.initiator.fundraising_campaigns.length > 0 ? data.initiator.fundraising_campaigns.map((c) => <CampaignCard key={c.campaign_id} campaign={c} compact />) : <EmptyState title="没有募资中活动" />}
          </DashboardGroup>
        </section>
      ) : null}

      {data.contributor ? (
        <section className="space-y-4">
          <SectionIntro eyebrow="Contributor" title="捐助人视角" description="展示你的累计捐助、可退款项目和参与过的募资活动。" />
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="参与活动数" value={data.contributor.contributions_total} />
            <StatCard label="累计捐助" value={formatWei(data.contributor.total_donated_wei)} />
            <StatCard label="可退款项目" value={data.contributor.refundable.length} />
            <StatCard label="成功 / 进行中" value={data.contributor.successful.length + data.contributor.fundraising.length} />
          </div>
          <DashboardGroup title="可退款项目">
            {data.contributor.refundable.length > 0 ? (
              <div className="grid gap-4 xl:grid-cols-2">
                {data.contributor.refundable.map((item) => (
                  <RoutePressable key={`${item.campaign_id}-${item.contributor_address}`} to={`/crowdfunding/campaigns/${item.campaign_id}`} className="block">
                    <Card className="transition hover:ring-primary/30">
                      <CardContent>
                        <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">Campaign #{item.campaign_id}</p>
                        <h3 className="text-base font-medium text-foreground">{item.github_url}</h3>
                        <div className="mt-3 grid gap-3 sm:grid-cols-2">
                          <SmallMetaPill label="累计捐助" value={formatWei(item.total_contributed_wei)} />
                          <SmallMetaPill label="已退款" value={formatWei(item.refund_claimed_wei)} />
                        </div>
                      </CardContent>
                    </Card>
                  </RoutePressable>
                ))}
              </div>
            ) : (
              <EmptyState title="没有可退款项目" />
            )}
          </DashboardGroup>
          <DashboardGroup title="募资中项目">
            {data.contributor.fundraising.length > 0 ? (
              <div className="grid gap-4 xl:grid-cols-2">
                {data.contributor.fundraising.map((item) => (
                  <RoutePressable key={`${item.campaign_id}-${item.contributor_address}`} to={`/crowdfunding/campaigns/${item.campaign_id}`} className="block">
                    <Card className="transition hover:ring-primary/30">
                      <CardContent>
                        <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">Campaign #{item.campaign_id}</p>
                        <h3 className="text-base font-medium text-foreground">{item.github_url}</h3>
                        <p className="mt-2 text-sm text-muted-foreground">累计捐助 {formatWei(item.total_contributed_wei)}</p>
                      </CardContent>
                    </Card>
                  </RoutePressable>
                ))}
              </div>
            ) : (
              <EmptyState title="没有募资中参与记录" />
            )}
          </DashboardGroup>
          <DashboardGroup title="成功 / 已完成项目">
            {data.contributor.successful.length > 0 ? (
              <div className="grid gap-4 xl:grid-cols-2">
                {data.contributor.successful.map((item) => (
                  <RoutePressable key={`${item.campaign_id}-${item.contributor_address}`} to={`/crowdfunding/campaigns/${item.campaign_id}`} className="block">
                    <Card className="transition hover:ring-primary/30">
                      <CardContent>
                        <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">Campaign #{item.campaign_id}</p>
                        <h3 className="text-base font-medium text-foreground">{item.github_url}</h3>
                        <p className="mt-2 text-sm text-muted-foreground">累计捐助 {formatWei(item.total_contributed_wei)}</p>
                      </CardContent>
                    </Card>
                  </RoutePressable>
                ))}
              </div>
            ) : (
              <EmptyState title="没有成功项目记录" />
            )}
          </DashboardGroup>
        </section>
      ) : null}

      {data.developer ? (
        <section className="space-y-4">
          <SectionIntro eyebrow="Developer" title="开发者视角" description="展示你参与的活动、待审批阶段和历史领取记录。" />
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="参与活动数" value={data.developer.campaigns.length} />
            <StatCard label="待审批里程碑" value={data.developer.pending_milestones.length} />
            <StatCard label="领取记录数" value={data.developer.claims.length} />
            <StatCard label="累计领取" value={formatWei(data.developer.total_claimed_wei)} />
          </div>
          <DashboardGroup title="参与中的活动">
            {data.developer.campaigns.length > 0 ? (
              <div className="grid gap-4 xl:grid-cols-2">
                {data.developer.campaigns.map((c) => <CampaignCard key={c.campaign_id} campaign={c} compact />)}
              </div>
            ) : (
              <EmptyState title="没有开发者活动" />
            )}
          </DashboardGroup>
          <DashboardGroup title="待审批里程碑">
            {data.developer.pending_milestones.length > 0 ? (
              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {data.developer.pending_milestones.map((m) => (
                  <Card key={`${m.campaign_id}-${m.milestone_index}`}>
                    <CardContent>
                      <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">Campaign #{m.campaign_id}</p>
                      <h3 className="text-base font-medium text-foreground">{m.description}</h3>
                      <p className="mt-2 text-sm text-muted-foreground">Milestone {m.milestone_index + 1}</p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : (
              <EmptyState title="没有待审批里程碑" />
            )}
          </DashboardGroup>
          <DashboardGroup title="历史领取记录">
            {data.developer.claims.length > 0 ? (
              <Card>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Campaign</TableHead>
                        <TableHead>Milestone</TableHead>
                        <TableHead>领取金额</TableHead>
                        <TableHead>时间</TableHead>
                        <TableHead>Tx</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.developer.claims.map((claim) => (
                        <TableRow key={`${claim.campaign_id}-${claim.milestone_index}-${claim.claimed_tx_hash}`}>
                          <TableCell>#{claim.campaign_id}</TableCell>
                          <TableCell>{claim.milestone_index + 1}</TableCell>
                          <TableCell>{formatWei(claim.claimed_amount_wei)}</TableCell>
                          <TableCell>{formatDateTime(claim.claimed_at)}</TableCell>
                          <TableCell className="font-mono">{shortHash(claim.claimed_tx_hash)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            ) : (
              <EmptyState title="暂无领取记录" />
            )}
          </DashboardGroup>
        </section>
      ) : null}

      {!data.initiator && !data.contributor && !data.developer ? (
        <EmptyState title="当前地址暂无可用工作台" description="后端返回的 `available_dashboards` 为空，说明这个地址还没有发起人 / 捐助人 / 开发者视角的数据。" />
      ) : null}
    </div>
  );
}
