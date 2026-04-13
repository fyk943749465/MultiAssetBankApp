import type { ReactNode } from "react";
import { RoutePressable } from "@/components/nav-spa";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  computeProgressPercent,
  formatDateTime,
  formatDuration,
  formatMilestonePercent,
  formatWei,
  payloadToText,
  shortHash,
  titleCaseStatus,
} from "./format";
import type {
  CPCampaign,
  CPCampaignMilestone,
  CPContribution,
  CPEventLog,
  CPProposal,
  CPProposalMilestone,
  Pagination,
} from "./types";

function inferStatusVariant(
  status?: string | null
): "default" | "secondary" | "destructive" | "outline" {
  if (!status) return "outline";
  if (/(success|approved|completed|claimed|active)/.test(status))
    return "default";
  if (/(pending|review|fundraising|progress)/.test(status))
    return "secondary";
  if (/(failed|rejected|refundable)/.test(status)) return "destructive";
  return "outline";
}

/* ── Section Intro ────────────────────────────────────── */
export function SectionIntro({
  eyebrow,
  title,
  description,
  action,
}: {
  readonly eyebrow?: string;
  readonly title: string;
  readonly description?: string;
  readonly action?: ReactNode;
}) {
  return (
    <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        {eyebrow ? (
          <p className="mb-1 text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            {eyebrow}
          </p>
        ) : null}
        <h2 className="text-xl font-semibold text-foreground">{title}</h2>
        {description ? (
          <p className="mt-2 max-w-3xl text-sm leading-relaxed text-muted-foreground">
            {description}
          </p>
        ) : null}
      </div>
      {action ? <div className="flex flex-wrap gap-2">{action}</div> : null}
    </div>
  );
}

/* ── Status Pill ──────────────────────────────────────── */
export function StatusPill({
  status,
}: {
  readonly status?: string | null;
}) {
  return (
    <Badge variant={inferStatusVariant(status)}>
      {titleCaseStatus(status)}
    </Badge>
  );
}

/* ── Small Meta Pill ──────────────────────────────────── */
export function SmallMetaPill({
  label,
  value,
}: {
  readonly label: string;
  readonly value: ReactNode;
}) {
  return (
    <div className="rounded-xl border bg-muted/40 px-3.5 py-2.5 dark:bg-muted/30">
      <p className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
        {label}
      </p>
      <div className="mt-1 text-sm font-medium text-foreground">{value}</div>
    </div>
  );
}

/* ── Stat Card ────────────────────────────────────────── */
export function StatCard({
  label,
  value,
  hint,
}: {
  readonly label: string;
  readonly value: ReactNode;
  readonly hint?: ReactNode;
}) {
  return (
    <Card>
      <CardHeader>
        <CardDescription className="text-[11px] font-semibold uppercase tracking-[0.2em]">
          {label}
        </CardDescription>
        <CardTitle className="text-2xl">{value}</CardTitle>
      </CardHeader>
      {hint ? (
        <CardContent>
          <p className="text-xs leading-relaxed text-muted-foreground">
            {hint}
          </p>
        </CardContent>
      ) : null}
    </Card>
  );
}

/* ── Loading / Error / Empty ──────────────────────────── */
export function LoadingState({
  message = "加载中…",
}: {
  readonly message?: string;
}) {
  return (
    <Card>
      <CardContent>
        <p className="text-sm text-muted-foreground">{message}</p>
      </CardContent>
    </Card>
  );
}

export function ErrorState({ message }: { readonly message: string }) {
  return (
    <Alert variant="destructive">
      <AlertTitle>请求失败</AlertTitle>
      <AlertDescription>
        <pre className="whitespace-pre-wrap break-words font-mono text-sm">
          {message}
        </pre>
      </AlertDescription>
    </Alert>
  );
}

export function EmptyState({
  title,
  description,
}: {
  readonly title: string;
  readonly description?: string;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {description ? (
          <CardDescription>{description}</CardDescription>
        ) : null}
      </CardHeader>
    </Card>
  );
}

/* ── Info Grid ────────────────────────────────────────── */
export function InfoGrid({
  items,
}: {
  readonly items: ReadonlyArray<{ label: string; value: ReactNode }>;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {items.map((item) => (
        <SmallMetaPill key={item.label} label={item.label} value={item.value} />
      ))}
    </div>
  );
}

/* ── Proposal Card ────────────────────────────────────── */
export function ProposalCard({
  proposal,
  compact = false,
}: {
  readonly proposal: CPProposal;
  readonly compact?: boolean;
}) {
  return (
    <RoutePressable
      to={`/crowdfunding/proposals/${proposal.proposal_id}`}
      className="block"
    >
      <Card className="transition-all duration-200 hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <CardDescription>
                Proposal #{proposal.proposal_id}
              </CardDescription>
              <CardTitle className="truncate">
                {proposal.github_url}
              </CardTitle>
            </div>
            <StatusPill status={proposal.status} />
          </div>
        </CardHeader>

        <CardContent>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <SmallMetaPill
              label="目标金额"
              value={formatWei(proposal.target_wei)}
            />
            <SmallMetaPill
              label="周期"
              value={formatDuration(proposal.duration_seconds)}
            />
            <SmallMetaPill
              label="发起人"
              value={
                <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">
                  {shortHash(proposal.organizer_address)}
                </code>
              }
            />
            <SmallMetaPill
              label="提交时间"
              value={formatDateTime(
                proposal.submitted_at ?? proposal.created_at
              )}
            />
          </div>

          {!compact && proposal.round_review_state ? (
            <p className="mt-4 text-sm text-muted-foreground">
              当前轮次审核:{" "}
              <span className="text-foreground">
                {titleCaseStatus(proposal.round_review_state)}
              </span>
            </p>
          ) : null}
        </CardContent>
      </Card>
    </RoutePressable>
  );
}

/* ── Campaign Card ────────────────────────────────────── */
export function CampaignCard({
  campaign,
  compact = false,
}: {
  readonly campaign: CPCampaign;
  readonly compact?: boolean;
}) {
  const progress = computeProgressPercent(
    campaign.amount_raised_wei,
    campaign.target_wei
  );

  return (
    <RoutePressable
      to={`/crowdfunding/campaigns/${campaign.campaign_id}`}
      className="block"
    >
      <Card className="transition-all duration-200 hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <CardDescription>
                Campaign #{campaign.campaign_id} · Proposal #
                {campaign.proposal_id}
              </CardDescription>
              <CardTitle className="truncate">
                {campaign.github_url}
              </CardTitle>
            </div>
            <StatusPill status={campaign.state} />
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          <div>
            <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground">
              <span>{formatWei(campaign.amount_raised_wei)}</span>
              <span>{progress.toFixed(2)}%</span>
              <span>{formatWei(campaign.target_wei)}</span>
            </div>
            <Progress value={progress} />
          </div>

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <SmallMetaPill
              label="截止时间"
              value={formatDateTime(campaign.deadline_at)}
            />
            <SmallMetaPill label="捐助者" value={campaign.donor_count} />
            <SmallMetaPill label="开发者" value={campaign.developer_count} />
            <SmallMetaPill
              label="发起时间"
              value={formatDateTime(campaign.launched_at)}
            />
          </div>

          {!compact && campaign.unclaimed_refund_pool_wei !== "0" ? (
            <p className="text-sm text-muted-foreground">
              待退款资金池:{" "}
              <span className="text-foreground">
                {formatWei(campaign.unclaimed_refund_pool_wei)}
              </span>
            </p>
          ) : null}
        </CardContent>
      </Card>
    </RoutePressable>
  );
}

/* ── Milestone List ───────────────────────────────────── */
export function MilestoneList({
  milestones,
}: {
  readonly milestones: ReadonlyArray<
    CPProposalMilestone | CPCampaignMilestone
  >;
}) {
  if (milestones.length === 0) {
    return (
      <EmptyState
        title="暂无里程碑"
        description="后端尚未同步到该对象的阶段信息。"
      />
    );
  }

  return (
    <div className="space-y-3">
      {milestones.map((milestone) => {
        const isCampaignMilestone = "campaign_id" in milestone;
        return (
          <Card
            key={`${isCampaignMilestone ? milestone.campaign_id : milestone.id}-${milestone.milestone_index}`}
          >
            <CardHeader>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <CardDescription>
                    Milestone {milestone.milestone_index + 1}
                    {"round_ordinal" in milestone
                      ? ` · Round ${milestone.round_ordinal}`
                      : ""}
                  </CardDescription>
                  <CardTitle>{milestone.description}</CardTitle>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="outline">
                    {formatMilestonePercent(milestone.percentage_raw)}
                  </Badge>
                  {"approved" in milestone ? (
                    <StatusPill
                      status={
                        milestone.approved ? "approved" : "pending_review"
                      }
                    />
                  ) : null}
                  {"claimed" in milestone && milestone.claimed ? (
                    <StatusPill status="claimed" />
                  ) : null}
                </div>
              </div>
            </CardHeader>
            {"unlock_at" in milestone && milestone.unlock_at ? (
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  解锁时间: {formatDateTime(milestone.unlock_at)}
                </p>
              </CardContent>
            ) : null}
          </Card>
        );
      })}
    </div>
  );
}

/* ── Timeline List ────────────────────────────────────── */
export function TimelineList({
  events,
}: {
  readonly events: ReadonlyArray<CPEventLog>;
}) {
  if (events.length === 0) {
    return (
      <EmptyState
        title="暂无时间线"
        description="事件同步完成后会在这里看到链上流水。"
      />
    );
  }

  return (
    <div className="space-y-3">
      {events.map((event) => (
        <Card key={`${event.tx_hash}-${event.log_index}`}>
          <CardHeader>
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <CardDescription>{event.event_name}</CardDescription>
                <CardTitle className="text-sm">
                  {formatDateTime(event.block_timestamp)}
                </CardTitle>
              </div>
              <div className="flex flex-wrap gap-2">
                {event.proposal_id ? (
                  <Badge variant="outline">
                    Proposal #{event.proposal_id}
                  </Badge>
                ) : null}
                {event.campaign_id ? (
                  <Badge variant="outline">
                    Campaign #{event.campaign_id}
                  </Badge>
                ) : null}
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <SmallMetaPill label="区块" value={event.block_number} />
              <SmallMetaPill label="Log Index" value={event.log_index} />
              <SmallMetaPill label="来源" value={event.source} />
              <SmallMetaPill
                label="交易"
                value={
                  <code className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[11px]">
                    {shortHash(event.tx_hash)}
                  </code>
                }
              />
            </div>
            <details>
              <summary className="cursor-pointer text-sm text-muted-foreground hover:text-foreground">
                查看 payload
              </summary>
              <pre className="mt-3 whitespace-pre-wrap rounded-lg border bg-muted/50 p-3 font-mono text-xs leading-relaxed text-muted-foreground">
                {payloadToText(event.payload)}
              </pre>
            </details>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

/* ── Contribution Table ───────────────────────────────── */
export function ContributionTable({
  contributions,
}: {
  readonly contributions: ReadonlyArray<CPContribution>;
}) {
  if (contributions.length === 0) {
    return (
      <EmptyState
        title="暂无贡献记录"
        description="该活动暂时还没有已聚合的捐助数据。"
      />
    );
  }

  return (
    <Card>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>贡献者</TableHead>
              <TableHead>累计捐助</TableHead>
              <TableHead>已退款</TableHead>
              <TableHead>最后捐助</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {contributions.map((item) => (
              <TableRow
                key={`${item.campaign_id}-${item.contributor_address}`}
              >
                <TableCell className="font-mono">
                  {shortHash(item.contributor_address)}
                </TableCell>
                <TableCell>
                  {formatWei(item.total_contributed_wei)}
                </TableCell>
                <TableCell>
                  {formatWei(item.refund_claimed_wei)}
                </TableCell>
                <TableCell>
                  {formatDateTime(item.last_donated_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

/* ── Pagination Controls ──────────────────────────────── */
export function PaginationControls({
  pagination,
  onPageChange,
}: {
  readonly pagination: Pagination;
  readonly onPageChange: (page: number) => void;
}) {
  const totalPages = Math.max(
    1,
    Math.ceil(pagination.total / pagination.page_size)
  );
  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-muted-foreground">
        第 {pagination.page} / {totalPages} 页，共 {pagination.total} 条
      </p>
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={pagination.page <= 1}
          onClick={() => onPageChange(pagination.page - 1)}
        >
          上一页
        </Button>
        <Button
          size="sm"
          disabled={pagination.page >= totalPages}
          onClick={() => onPageChange(pagination.page + 1)}
        >
          下一页
        </Button>
      </div>
    </div>
  );
}

/* ── Callout ──────────────────────────────────────────── */
export function Callout({
  tone = "neutral",
  title,
  description,
}: {
  readonly tone?: "neutral" | "warn";
  readonly title: string;
  readonly description: string;
}) {
  return (
    <Alert variant={tone === "warn" ? "destructive" : "default"}>
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>{description}</AlertDescription>
    </Alert>
  );
}
