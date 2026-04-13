import { useCallback, useEffect, useState } from "react";
import { NavButton } from "@/components/nav-spa";
import { useAccount } from "wagmi";
import { ActionFormCard } from "../../features/codepulse/action-ui";
import {
  addCodePulseInitiator,
  fetchCodePulseAdminDashboard,
  fetchCodePulseInitiators,
  fetchCodePulsePlatformFunds,
  fetchCodePulseSyncStatus,
  fetchCodePulseWalletOverview,
  removeCodePulseInitiator,
} from "../../features/codepulse/api";
import {
  CampaignCard,
  Callout,
  EmptyState,
  ErrorState,
  PaginationControls,
  ProposalCard,
  SectionIntro,
  StatCard,
} from "../../features/codepulse/components";
import { formatDateTime, formatWei, shortHash } from "../../features/codepulse/format";
import type {
  AdminDashboard,
  InitiatorListResponse,
  PlatformFundsResponse,
  SyncStatusResponse,
  WalletOverview,
} from "../../features/codepulse/types";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type AdminState = {
  overview: WalletOverview;
  dashboard: AdminDashboard;
  initiators: InitiatorListResponse;
  platformFunds: PlatformFundsResponse;
  sync: SyncStatusResponse;
};

export function CrowdfundingAdminPage() {
  const { address, isConnected } = useAccount();
  const [page, setPage] = useState(1);
  const [data, setData] = useState<AdminState | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [initiatorInput, setInitiatorInput] = useState("");
  const [initiatorMutationLoading, setInitiatorMutationLoading] = useState(false);

  const load = useCallback(async () => {
    if (!isConnected || !address) { setData(null); setLoading(false); setError(null); return; }
    setLoading(true);
    setError(null);
    try {
      const overview = await fetchCodePulseWalletOverview(address);
      const [dashboard, initiators, platformFunds, sync] = await Promise.all([
        fetchCodePulseAdminDashboard(),
        fetchCodePulseInitiators(),
        fetchCodePulsePlatformFunds(page, 10),
        fetchCodePulseSyncStatus(),
      ]);
      setData({ overview, dashboard, initiators, platformFunds, sync });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [address, isConnected, page]);

  useEffect(() => { void load(); }, [load]);

  async function handleAddInitiator() {
    if (!initiatorInput.trim()) return;
    setInitiatorMutationLoading(true);
    setError(null);
    try {
      await addCodePulseInitiator(initiatorInput.trim());
      setInitiatorInput("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setInitiatorMutationLoading(false);
    }
  }

  async function handleRemoveInitiator(addressToRemove: string) {
    setInitiatorMutationLoading(true);
    setError(null);
    try {
      await removeCodePulseInitiator(addressToRemove);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setInitiatorMutationLoading(false);
    }
  }

  if (!isConnected || !address) {
    return (
      <div className="space-y-6">
        <Callout tone="warn" title="请先连接管理员钱包" description="请在页面右上角连接钱包。Admin 工作台需要当前钱包地址先完成角色识别，随后再拉取 dashboard、白名单与平台资金数据。" />
      </div>
    );
  }

  if (loading && !data) return <Card><CardContent><p className="text-sm text-muted-foreground">正在加载管理员工作台…</p></CardContent></Card>;
  if (error && !data) return <ErrorState message={error} />;
  if (!data) return <EmptyState title="暂无管理员数据" description="请确认后端管理接口可用。" />;

  if (!data.overview.is_admin) {
    return (
      <div className="space-y-6">
        <Callout tone="warn" title="当前地址不是管理员" description="后端 `wallet overview` 返回当前钱包没有 admin 角色，因此不应执行管理类动作。如果需要切换账号，请在右上方连接钱包处操作。" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap gap-2">
        <NavButton to="/crowdfunding/me" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
          返回我的工作台
        </NavButton>
        <NavButton to="/crowdfunding" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
          返回首页
        </NavButton>
      </div>

      <Card>
        <CardContent>
          <SectionIntro eyebrow="Admin" title="Code Pulse 管理台" description="第二阶段接入管理员工作台：审核提案与 funding round、审批里程碑、管理 proposal initiator、查看平台资金与同步状态，并支持核心管理动作。" />
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="待审核提案" value={data.dashboard.pending_proposals.length} />
            <StatCard label="待审核轮次" value={data.dashboard.pending_rounds.length} />
            <StatCard label="待审批里程碑" value={data.dashboard.pending_milestones.length} />
            <StatCard label="事件总数" value={data.sync.event_count} />
          </div>
        </CardContent>
      </Card>

      {error ? <ErrorState message={error} /> : null}

      <section className="space-y-4">
        <SectionIntro eyebrow="Pending Proposals" title="提案审核队列" description="管理员可对 pending_review 的 proposal 执行 approve / reject。" />
        {data.dashboard.pending_proposals.length > 0 ? (
          data.dashboard.pending_proposals.map((proposal) => (
            <div key={proposal.proposal_id} className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]">
              <ProposalCard proposal={proposal} />
              <div className="space-y-4">
                <ActionFormCard title="审核通过提案" action="review_proposal" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id), approve: true }} description="通过后，发起人即可进入首轮 funding round 提交阶段。" onSuccess={() => void load()} />
                <ActionFormCard title="拒绝提案" action="review_proposal" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id), approve: false }} description="拒绝会让提案离开待审核队列。" onSuccess={() => void load()} />
              </div>
            </div>
          ))
        ) : (
          <EmptyState title="没有待审核提案" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Pending Rounds" title="Funding Round 审核队列" description="对于 round_review_state=pending 的提案，管理员可批准或拒绝新的 funding round。" />
        {data.dashboard.pending_rounds.length > 0 ? (
          data.dashboard.pending_rounds.map((proposal) => (
            <div key={proposal.proposal_id} className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]">
              <ProposalCard proposal={proposal} />
              <div className="space-y-4">
                <ActionFormCard title="批准 funding round" action="review_funding_round" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id), approve: true }} onSuccess={() => void load()} />
                <ActionFormCard title="拒绝 funding round" action="review_funding_round" wallet={address} proposalId={proposal.proposal_id} presetParams={{ proposal_id: String(proposal.proposal_id), approve: false }} onSuccess={() => void load()} />
              </div>
            </div>
          ))
        ) : (
          <EmptyState title="没有待审核 funding round" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Milestones" title="待审批里程碑" description="管理员可针对每个 milestone 执行链上审批。" />
        {data.dashboard.pending_milestones.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {data.dashboard.pending_milestones.map((milestone) => (
              <Card key={`${milestone.campaign_id}-${milestone.milestone_index}`}>
                <CardContent>
                  <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">Campaign #{milestone.campaign_id}</p>
                  <h3 className="text-base font-medium text-foreground">{milestone.github_url}</h3>
                  <p className="mt-2 text-sm text-muted-foreground">{milestone.description}</p>
                  <div className="mt-4">
                    <ActionFormCard title={`审批 Milestone ${milestone.milestone_index + 1}`} action="approve_milestone" wallet={address} campaignId={milestone.campaign_id} milestoneIndex={milestone.milestone_index} presetParams={{ campaign_id: String(milestone.campaign_id), milestone_index: String(milestone.milestone_index) }} onSuccess={() => void load()} />
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <EmptyState title="没有待审批里程碑" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Initiators" title="Proposal Initiator 白名单" description="先使用管理 API 更新数据库角色，再根据需要发链上 `set_proposal_initiator` 交易同步合约权限。" />
        <Card>
          <CardContent className="space-y-4">
            <div className="flex flex-col gap-3 md:flex-row">
              <Input value={initiatorInput} onChange={(e) => setInitiatorInput(e.target.value)} placeholder="0x..." />
              <Button disabled={initiatorMutationLoading} onClick={() => void handleAddInitiator()}>
                添加到数据库白名单
              </Button>
            </div>
            <div className="flex flex-wrap gap-3">
              {data.initiators.initiators.length > 0 ? (
                data.initiators.initiators.map((entry) => (
                  <div key={entry} className="flex items-center gap-2 rounded-lg border bg-card/50 px-3 py-2">
                    <span className="font-mono text-sm text-foreground">{shortHash(entry, 8, 6)}</span>
                    <Button variant="outline" size="xs" disabled={initiatorMutationLoading} onClick={() => void handleRemoveInitiator(entry)}>
                      移除
                    </Button>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">当前没有数据库白名单记录。</p>
              )}
            </div>
          </CardContent>
        </Card>

        <div className="grid gap-4 xl:grid-cols-2">
          <ActionFormCard title="链上设置 Proposal Initiator" action="set_proposal_initiator" wallet={address} description="该动作走合约 `setProposalInitiator(account, allowed)`，适合把数据库角色同步到链上。" fields={[{ key: "account", label: "地址", kind: "address", required: true, placeholder: "0x..." }, { key: "allowed", label: "允许提案发起", kind: "boolean" }]} />
          <ActionFormCard title="转移所有权" action="transfer_ownership" wallet={address} fields={[{ key: "new_owner", label: "新管理员地址", kind: "address", required: true, placeholder: "0x..." }]} />
        </div>
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Platform Funds" title="平台资金与提现" description="展示平台捐赠 / 提现总额与资金流水，并支持管理员发起提现交易。" />
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="平台捐赠总额" value={formatWei(data.platformFunds.total_donations)} />
          <StatCard label="平台提现总额" value={formatWei(data.platformFunds.total_withdrawals)} />
          <StatCard label="Dashboard Donation" value={formatWei(data.dashboard.platform_donations)} />
          <StatCard label="Dashboard Withdrawal" value={formatWei(data.dashboard.platform_withdrawals)} />
        </div>
        <ActionFormCard title="提现平台资金" action="withdraw_platform_funds" wallet={address} description="输入 ETH 金额，前端会自动换算为 wei。" fields={[{ key: "amount", label: "提现金额（ETH）", kind: "eth", required: true, placeholder: "0.05" }]} />
        <Card>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>方向</TableHead>
                  <TableHead>钱包</TableHead>
                  <TableHead>金额</TableHead>
                  <TableHead>时间</TableHead>
                  <TableHead>Tx</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.platformFunds.movements.map((row) => (
                  <TableRow key={`${row.tx_hash}-${row.log_index}`}>
                    <TableCell>{row.direction}</TableCell>
                    <TableCell className="font-mono">{shortHash(row.wallet_address)}</TableCell>
                    <TableCell>{formatWei(row.amount_wei)}</TableCell>
                    <TableCell>{formatDateTime(row.block_timestamp)}</TableCell>
                    <TableCell className="font-mono">{shortHash(row.tx_hash)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <PaginationControls pagination={data.platformFunds.pagination} onPageChange={setPage} />
          </CardContent>
        </Card>
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="System" title="系统状态与管理动作" description="同步状态来自只读 API；pause / unpause / renounce 等控制则通过交易提交流程执行。" />
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Sync Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {data.sync.cursors.length > 0 ? (
                data.sync.cursors.map((cursor) => (
                  <div key={cursor.sync_name} className="rounded-lg border bg-card/50 p-4">
                    <p className="text-sm font-medium text-foreground">{cursor.sync_name}</p>
                    <p className="mt-2 text-xs text-muted-foreground">区块: {cursor.last_block_number ?? "N/A"}</p>
                    <p className="mt-1 text-xs text-muted-foreground">更新时间: {formatDateTime(cursor.updated_at)}</p>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">暂无同步游标。</p>
              )}
            </div>
          </CardContent>
        </Card>

        <div className="grid gap-4 xl:grid-cols-2">
          <ActionFormCard title="Pause 合约" action="pause" wallet={address} />
          <ActionFormCard title="Unpause 合约" action="unpause" wallet={address} />
          <ActionFormCard title="Renounce Ownership" action="renounce_ownership" wallet={address} />
        </div>
      </section>

      <section className="space-y-4">
        <SectionIntro eyebrow="Live Campaigns" title="当前募资中的活动" description="管理员可以从这里快速跳转到活动详情，继续执行 donate / finalize / developer / milestone 等动作。" />
        {data.dashboard.live_campaigns.length > 0 ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {data.dashboard.live_campaigns.map((campaign) => (
              <CampaignCard key={campaign.campaign_id} campaign={campaign} />
            ))}
          </div>
        ) : (
          <EmptyState title="没有募资中活动" />
        )}
      </section>
    </div>
  );
}
