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
      // 与原先「先 overview 再并行」相比，五路同时发出，总耗时接近最慢的一路而非相加。
      const [overview, dashboard, initiators, platformFunds, sync] = await Promise.all([
        fetchCodePulseWalletOverview(address),
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
  }, [
    address,
    isConnected,
    page,
  ]);

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
            <StatCard
              label="事件总数"
              value={data.sync.event_count}
              hint={
                data.sync.event_count_source === "subgraph" &&
                data.sync.event_count_database != null &&
                data.sync.event_count_database !== data.sync.event_count
                  ? `子图口径；索引库 ${data.sync.event_count_database} 条（可能缺部署早期块）`
                  : data.sync.event_count_source === "subgraph"
                    ? "子图口径（与链上事件一致）"
                    : "PostgreSQL 索引库（子图未配置或计数失败时）"
              }
            />
          </div>
        </CardContent>
      </Card>

      {error ? <ErrorState message={error} /> : null}

      <section className="space-y-4">
        <SectionIntro eyebrow="Pending Proposals" title="提案审核队列" description="管理员可对 pending_review 的 proposal 执行 approve / reject。" />
        {data.dashboard.pending_proposals.length > 0 ? (
          data.dashboard.pending_proposals.map((proposal) => (
            <div key={proposal.proposal_id} className="space-y-3">
              <ProposalCard proposal={proposal} compact />
              <div className="grid gap-3 md:grid-cols-2">
                <ActionFormCard
                  title="审核通过"
                  action="review_proposal"
                  wallet={address}
                  proposalId={proposal.proposal_id}
                  presetParams={{ proposal_id: String(proposal.proposal_id), approve: true }}
                  description="在链上执行管理员对提案的审核（approve=true）：表示您认可该提案进入后续流程。通过后，发起人通常可继续提交首轮 funding round 等材料；具体下一状态以合约与索引为准。仅当前钱包被识别为管理员时预检会通过；若提案已不在待审状态，预检或链上会失败。请核对提案标题、目标与风险后再确认。发起交易需支付 gas。"
                  onSuccess={() => void load()}
                />
                <ActionFormCard
                  title="拒绝提案"
                  action="review_proposal"
                  wallet={address}
                  proposalId={proposal.proposal_id}
                  presetParams={{ proposal_id: String(proposal.proposal_id), approve: false }}
                  description="在链上执行管理员对提案的审核（approve=false）：表示您不批准该提案继续走后续募资路径。拒绝后提案会离开「待管理员审核」队列，发起人需按产品规则修改或放弃。仅管理员可调用；若提案状态已变更，预检可能失败。该操作具有治理含义，建议在后台或沟通中有记录后再执行。发起交易需支付 gas。"
                  onSuccess={() => void load()}
                />
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
            <div key={proposal.proposal_id} className="space-y-3">
              <ProposalCard proposal={proposal} compact />
              <div className="grid gap-3 md:grid-cols-2">
                <ActionFormCard
                  title="批准 funding round"
                  action="review_funding_round"
                  wallet={address}
                  proposalId={proposal.proposal_id}
                  presetParams={{ proposal_id: String(proposal.proposal_id), approve: true }}
                  description="在链上执行管理员对「新一轮/首轮 funding round」的审核（approve=true）：表示您同意发起人提交的募集参数（目标金额、时长等以合约为准）进入可启动状态。通过后发起人通常可执行 launch 等后续动作以真正创建链上众筹活动。仅管理员可审；若轮次不在 pending、或参数与链上状态不一致，预检会失败。发起交易需支付 gas。"
                  onSuccess={() => void load()}
                />
                <ActionFormCard
                  title="拒绝 funding round"
                  action="review_funding_round"
                  wallet={address}
                  proposalId={proposal.proposal_id}
                  presetParams={{ proposal_id: String(proposal.proposal_id), approve: false }}
                  description="在链上执行管理员对 funding round 的审核（approve=false）：表示您不同意本轮募集参数上线。拒绝后轮次需发起人调整后重新提交审核，或停留在不可启动状态（以合约为准）。仅管理员可操作；请在拒绝前与发起人同步原因。发起交易需支付 gas。"
                  onSuccess={() => void load()}
                />
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
            {data.dashboard.pending_milestones.map((milestone) => {
              const msLabel = `第 ${milestone.milestone_index + 1} 阶段`;
              const descTrim = milestone.description.trim();
              const ghTrim = milestone.github_url?.trim() ?? "";
              const parts: string[] = [];
              if (descTrim) {
                const cut = descTrim.slice(0, 160);
                const more = descTrim.length > 160;
                parts.push(`本阶段说明（节选）：${cut}${more ? "…" : ""}`);
              }
              if (ghTrim) {
                parts.push(`关联链接：${ghTrim}`);
              }
              const ctx = parts.length > 0 ? ` ${parts.join(" ")}` : "";
              return (
                <ActionFormCard
                  key={`${milestone.campaign_id}-${milestone.milestone_index}`}
                  title={`活动 #${milestone.campaign_id} — ${msLabel}`}
                  action="approve_milestone"
                  wallet={address}
                  campaignId={milestone.campaign_id}
                  milestoneIndex={milestone.milestone_index}
                  description={`在链上执行 approveMilestone：由管理员将活动 #${milestone.campaign_id} 的「${msLabel}」标记为已审批，使该阶段在合约侧满足「开发者可领取该阶段份额」等后续前提（具体顺序与比例以合约为准）。仅当当前钱包为管理员且该阶段尚未在链上审批时，预检通常才会通过；重复审批会失败。审批前请对照库内说明、活动状态与链上是否一致；列表数据可能有延迟，以「预检并构建」与模拟结果为准。${ctx}发起交易需支付 gas。`}
                  presetParams={{ campaign_id: String(milestone.campaign_id), milestone_index: String(milestone.milestone_index) }}
                  onSuccess={() => void load()}
                />
              );
            })}
          </div>
        ) : (
          <EmptyState title="没有待审批里程碑" />
        )}
      </section>

      <section className="space-y-4">
        <SectionIntro
          eyebrow="Initiators"
          title="可发起提案的地址"
          description={
            data.initiators.data_source === "subgraph"
              ? "下列地址来自链上（子图），即当前合约允许发起提案的钱包。链上已撤销或未授权的地址不会出现在这里。若要授权新人或撤销某人，请只用下方「链上设置 Proposal Initiator」发交易，不必在数据库里删地址来「对齐」列表。"
              : "子图暂时不可用，下列地址来自 PostgreSQL 中的活跃记录，仅供应急参考，与链上可能不一致；子图恢复后将自动改回只读链上列表。若你仍需在库里维护预检/运营用记录，可使用添加与移除。建议在服务端配置 CODE_PULSE_INITIATOR_RECONCILE_SECONDS（例如 600），由后台定时用子图或合约调用把数据库与链上白名单对齐，兜底时更接近真实。"
          }
        />
        <Card>
          <CardContent className="space-y-4">
            {data.initiators.data_source !== "subgraph" ? (
              <div className="flex flex-col gap-3 md:flex-row">
                <Input value={initiatorInput} onChange={(e) => setInitiatorInput(e.target.value)} placeholder="0x..." />
                <Button disabled={initiatorMutationLoading} onClick={() => void handleAddInitiator()}>
                  添加到数据库（预检/运营）
                </Button>
              </div>
            ) : null}
            <p className="text-xs text-muted-foreground">
              当前展示来源：
              <span className="font-medium text-foreground">
                {data.initiators.data_source === "subgraph" ? "子图（链上）" : "PostgreSQL"}
              </span>
              {data.initiators.data_source === "subgraph" ? " · 只读" : null}
            </p>
            {data.initiators.data_source !== "subgraph" ? (
              <p className="text-xs text-muted-foreground">
                当前为数据库兜底展示：「从数据库移除」只会更新库里的记录；子图恢复后列表将以链上为准。
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                此为链上真实允许集；撤销权限后子图索引更新，对应地址会从列表消失。
              </p>
            )}
            <div className="flex flex-wrap gap-3">
              {data.initiators.initiators.length > 0 ? (
                data.initiators.initiators.map((entry) => (
                  <div key={entry} className="flex items-center gap-2 rounded-lg border bg-card/50 px-3 py-2">
                    <span className="font-mono text-sm text-foreground" title={entry}>
                      {shortHash(entry, 8, 6)}
                    </span>
                    {data.initiators.data_source !== "subgraph" ? (
                      <Button variant="outline" size="xs" disabled={initiatorMutationLoading} onClick={() => void handleRemoveInitiator(entry)}>
                        从数据库移除
                      </Button>
                    ) : null}
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">
                  {data.initiators.data_source === "subgraph"
                    ? "链上当前没有处于「允许发起提案」状态的地址。"
                    : "数据库中暂无活跃的 initiator 记录。"}
                </p>
              )}
            </div>
          </CardContent>
        </Card>

        <div className="grid gap-4 xl:grid-cols-2">
          <ActionFormCard
            title="链上设置 Proposal Initiator"
            action="set_proposal_initiator"
            wallet={address}
            description="这张卡片用来直接改链上谁能发起「新」提案（setProposalInitiator）：在「地址」里填目标钱包，在「允许提案发起」里选是或否，预检通过后发交易。选「否」即撤销 initiator 白名单。说明：地址一旦已经提交过提案，在链上会成为该提案的 organizer，后续提交轮次、launch 募资等动作通常按 organizer 校验，而不是再次要求 initiator——因此撤销白名单一般不会让对方已发起的提案线彻底停摆，但会禁止其再 submit 新提案。预检若发现该地址在库里仍是多条非拒绝态提案的 organizer，会弹出提示；若部署侧开启 CODE_PULSE_BLOCK_INITIATOR_REVOKE_ORGANIZER，则会硬拦撤销直到你处理完相关提案。发交易需要 gas。"
            fields={[{ key: "account", label: "地址", kind: "address", required: true, placeholder: "0x..." }, { key: "allowed", label: "允许提案发起", kind: "boolean" }]}
          />
          <ActionFormCard
            title="转移合约所有权"
            action="transfer_ownership"
            wallet={address}
            description="在链上调用 transferOwnership(newOwner)：将本 Code Pulse 合约的 owner 角色移交给 new_owner 地址。完成后，原管理员钱包将失去 owner 权限（无法再执行 pause、设置 initiator、里程碑审批等 owner 专属操作），新地址成为唯一 owner。此操作不可逆且错误地址会导致合约失控，请务必核对新管理员地址完整、正确，并确认接收方钱包可控。发起交易需支付 gas。"
            fields={[{ key: "new_owner", label: "新管理员地址", kind: "address", required: true, placeholder: "0x..." }]}
          />
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
        <ActionFormCard
          title="提现平台资金"
          action="withdraw_platform_funds"
          wallet={address}
          description="在链上由管理员从合约中提取平台可支配资金到当前连接钱包（或合约实现所指向的收款方，以合约为准）。下方金额以 ETH 填写，前端会换算为 wei 写入交易。预检会校验您是否为管理员、以及可提余额是否足够；超过合约记录的平台余额会失败。该操作直接影响金库，请与账务记录一致后再确认。发起交易需支付 gas。"
          fields={[{ key: "amount", label: "提现金额（ETH）", kind: "eth", required: true, placeholder: "0.05" }]}
        />
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

            {data.sync.chain_heads ? (
              <div className="mt-6 rounded-lg border border-primary/20 bg-primary/5 p-4">
                <p className="text-xs font-semibold uppercase tracking-wide text-primary/90">RPC 链头（对照游标）</p>
                <p className="mt-2 text-xs text-muted-foreground">
                  索引器只扫到「确认上界」为止：当前采用{" "}
                  <span className="font-mono text-foreground">
                    {data.sync.chain_heads.confirmed_tip_source === "finalized"
                      ? "finalized"
                      : data.sync.chain_heads.confirmed_tip_source === "safe"
                        ? "safe"
                        : "latest − 12"}
                  </span>
                  ，上界块号{" "}
                  <span className="font-mono text-foreground">{data.sync.chain_heads.confirmed_tip_block}</span>
                  。若你的交易块号大于上界，需等确认追上后游标才会继续动。
                </p>
                <dl className="mt-3 grid gap-2 text-xs sm:grid-cols-2 lg:grid-cols-4">
                  <div>
                    <dt className="text-muted-foreground">latest</dt>
                    <dd className="font-mono text-foreground">{data.sync.chain_heads.latest_block}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">safe</dt>
                    <dd className="font-mono text-foreground">
                      {data.sync.chain_heads.safe_block ?? "—"}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">finalized</dt>
                    <dd className="font-mono text-foreground">
                      {data.sync.chain_heads.finalized_block ?? "—"}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">索引上界（confirmed_tip）</dt>
                    <dd className="font-mono text-foreground">{data.sync.chain_heads.confirmed_tip_block}</dd>
                  </div>
                </dl>
              </div>
            ) : null}
          </CardContent>
        </Card>

        <Callout
          tone="neutral"
          title="链上事件流水已移至首页 Home"
          description="所有访客在众筹模块「Home」Tab 可查看同一套事件列表（公开接口 /api/code-pulse/events）。此处仍保留「事件总数」统计（来自同步状态）。"
        />

        <div className="grid gap-4 xl:grid-cols-2">
          <ActionFormCard
            title="暂停合约（Pause）"
            action="pause"
            wallet={address}
            description="在链上调用 pause()：进入暂停状态后，依赖该开关的用户级操作（如捐助、结算等，以合约实现为准）通常会被拒绝，用于紧急止损或升级窗口。仅 owner 可执行；若已处于 pause，重复 pause 可能失败。暂停期间管理员仍可能保留部分管理操作，具体以合约为准。请仅在明确需要冻结用户入口时使用。发起交易需支付 gas。"
          />
          <ActionFormCard
            title="恢复合约（Unpause）"
            action="unpause"
            wallet={address}
            description="在链上调用 unpause()：解除暂停，使用户侧业务流程恢复正常。仅 owner 可执行；若当前未 pause，预检或链上可能失败。解除前请确认风险已排除、依赖服务已就绪。发起交易需支付 gas。"
          />
          <ActionFormCard
            title="放弃所有权（Renounce Ownership）"
            action="renounce_ownership"
            wallet={address}
            description="在链上调用 renounceOwnership()：主动放弃合约 owner 身份且通常不设新 owner（以 OpenZeppelin Ownable 语义为准）。执行后无人再拥有管理员权限，合约中依赖 onlyOwner 的功能将永久不可调用，可能造成资金或配置锁死。除非您完全理解后果并有替代治理方案，否则不要使用。发起交易需支付 gas。"
          />
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
