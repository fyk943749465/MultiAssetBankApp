import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { fetchCodePulseEventLog } from "./api";
import { ErrorState, PaginationControls, SectionIntro } from "./components";
import { explorerTxUrl, formatDateTime, shortHash } from "./format";
import type { AdminEventsResponse } from "./types";

type Props = {
  chainId: number;
  eyebrow?: string;
  title?: string;
  description?: string;
};

export function CodePulseEventLogSection({
  chainId,
  eyebrow = "On-chain",
  title = "链上事件流水",
  description = "公开只读：数据优先来自子图，不可用时回退索引库。按区块倒序分页，可筛选事件名或提案/活动编号；与详情页时间线同源。",
}: Props) {
  const [eventsPage, setEventsPage] = useState(1);
  const [draftEventName, setDraftEventName] = useState("");
  const [draftProposalId, setDraftProposalId] = useState("");
  const [draftCampaignId, setDraftCampaignId] = useState("");
  const [appliedEventName, setAppliedEventName] = useState("");
  const [appliedProposalId, setAppliedProposalId] = useState("");
  const [appliedCampaignId, setAppliedCampaignId] = useState("");
  const [data, setData] = useState<AdminEventsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const proposalIdNum =
        /^\d+$/.test(appliedProposalId.trim()) ? Number(appliedProposalId.trim()) : undefined;
      const campaignIdNum =
        /^\d+$/.test(appliedCampaignId.trim()) ? Number(appliedCampaignId.trim()) : undefined;
      const res = await fetchCodePulseEventLog({
        page: eventsPage,
        page_size: 25,
        event_name: appliedEventName.trim() || undefined,
        proposal_id: proposalIdNum,
        campaign_id: campaignIdNum,
      });
      setData(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [eventsPage, appliedEventName, appliedProposalId, appliedCampaignId]);

  useEffect(() => {
    void load();
  }, [load]);

  function applyFilters() {
    setAppliedEventName(draftEventName);
    setAppliedProposalId(draftProposalId);
    setAppliedCampaignId(draftCampaignId);
    setEventsPage(1);
  }

  return (
    <section className="space-y-4">
      <SectionIntro eyebrow={eyebrow} title={title} description={description} />
      <Card>
        <CardContent className="space-y-4">
          {error ? <ErrorState message={error} /> : null}

          <div className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end">
            <div className="min-w-[10rem] flex-1 space-y-1">
              <p className="text-xs font-medium text-muted-foreground">事件名（精确匹配）</p>
              <Input
                value={draftEventName}
                onChange={(e) => setDraftEventName(e.target.value)}
                placeholder="如 ProposalReviewed"
                aria-label="按事件名筛选"
              />
            </div>
            <div className="w-full min-w-[6rem] max-w-[10rem] space-y-1">
              <p className="text-xs font-medium text-muted-foreground">Proposal ID</p>
              <Input
                value={draftProposalId}
                onChange={(e) => setDraftProposalId(e.target.value)}
                placeholder="数字"
                aria-label="按提案 ID 筛选"
              />
            </div>
            <div className="w-full min-w-[6rem] max-w-[10rem] space-y-1">
              <p className="text-xs font-medium text-muted-foreground">Campaign ID</p>
              <Input
                value={draftCampaignId}
                onChange={(e) => setDraftCampaignId(e.target.value)}
                placeholder="数字"
                aria-label="按活动 ID 筛选"
              />
            </div>
            <Button type="button" variant="secondary" onClick={() => applyFilters()}>
              应用筛选
            </Button>
          </div>

          {loading && !data ? (
            <p className="text-sm text-muted-foreground">正在加载事件流水…</p>
          ) : null}

          {data ? (
            <>
              <div className="overflow-x-auto rounded-lg border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="whitespace-nowrap">区块</TableHead>
                      <TableHead className="whitespace-nowrap">时间</TableHead>
                      <TableHead>事件</TableHead>
                      <TableHead className="whitespace-nowrap">P#</TableHead>
                      <TableHead className="whitespace-nowrap">C#</TableHead>
                      <TableHead>Tx</TableHead>
                      <TableHead>来源</TableHead>
                      <TableHead className="min-w-[8rem]">Payload</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.events.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={8} className="text-center text-muted-foreground">
                          当前条件下无事件
                        </TableCell>
                      </TableRow>
                    ) : (
                      data.events.map((ev) => (
                        <TableRow key={`${ev.tx_hash}-${ev.log_index}`}>
                          <TableCell className="font-mono text-xs">{ev.block_number}</TableCell>
                          <TableCell className="whitespace-nowrap text-xs">
                            {formatDateTime(ev.block_timestamp)}
                          </TableCell>
                          <TableCell className="max-w-[14rem] text-xs">{ev.event_name}</TableCell>
                          <TableCell className="font-mono text-xs">{ev.proposal_id ?? "—"}</TableCell>
                          <TableCell className="font-mono text-xs">{ev.campaign_id ?? "—"}</TableCell>
                          <TableCell>
                            <a
                              href={explorerTxUrl(chainId, ev.tx_hash)}
                              target="_blank"
                              rel="noreferrer"
                              className="font-mono text-xs text-primary underline-offset-2 hover:underline"
                            >
                              {shortHash(ev.tx_hash, 8, 6)}
                            </a>
                          </TableCell>
                          <TableCell className="text-xs text-muted-foreground">{ev.source}</TableCell>
                          <TableCell className="max-w-[12rem] align-top text-xs">
                            <details className="cursor-pointer">
                              <summary className="text-muted-foreground">查看</summary>
                              <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-all rounded-md border bg-muted/40 p-2 font-mono text-[10px] leading-relaxed">
                                {JSON.stringify(ev.payload, null, 2)}
                              </pre>
                            </details>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
              <PaginationControls pagination={data.pagination} onPageChange={setEventsPage} />
              {data.data_source ? (
                <p className="text-xs text-muted-foreground">
                  当前数据来源：<span className="font-medium text-foreground">{data.data_source}</span>
                  {data.data_source === "subgraph" ? "（子图模式下 total 为当前查询窗口内条数）" : null}
                </p>
              ) : null}
            </>
          ) : null}
        </CardContent>
      </Card>
    </section>
  );
}
