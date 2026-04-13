import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { fetchCodePulseCampaigns, fetchCodePulseProposals } from "../../features/codepulse/api";
import {
  CampaignCard,
  EmptyState,
  ErrorState,
  LoadingState,
  PaginationControls,
  ProposalCard,
  SectionIntro,
} from "../../features/codepulse/components";
import type { CampaignListResponse, ProposalListResponse } from "../../features/codepulse/types";

type ExploreView = "proposals" | "campaigns";

const proposalStatuses = [
  "", "pending_review", "approved", "rejected",
  "round_review_pending", "round_review_approved", "round_review_rejected", "settled",
] as const;

const campaignStates = [
  "", "fundraising", "successful", "milestone_in_progress", "completed", "failed_refundable",
] as const;

const selectClass =
  "mt-1 w-full rounded-lg border bg-background px-3.5 py-2.5 font-mono text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20";

export function CrowdfundingExplorePage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [proposalData, setProposalData] = useState<ProposalListResponse | null>(null);
  const [campaignData, setCampaignData] = useState<CampaignListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const view: ExploreView = searchParams.get("view") === "campaigns" ? "campaigns" : "proposals";
  const page = Math.max(1, Number(searchParams.get("page") || 1));
  const keyword = searchParams.get("keyword") ?? "";
  const status = searchParams.get("status") ?? "";
  const reviewState = searchParams.get("review_state") ?? "";
  const proposalSort = (searchParams.get("sort") as "submitted_at_desc" | "submitted_at_asc" | null) ?? "submitted_at_desc";
  const campaignState = searchParams.get("state") ?? "";
  const campaignSort =
    (searchParams.get("sort") as "launched_at_desc" | "deadline_at_asc" | "amount_raised_desc" | null) ?? "launched_at_desc";

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    (async () => {
      try {
        if (view === "proposals") {
          const result = await fetchCodePulseProposals({
            status: status || undefined,
            review_state: reviewState || undefined,
            sort: proposalSort,
            page,
            page_size: 8,
          });
          if (cancelled) return;
          setProposalData(result);
        } else {
          const result = await fetchCodePulseCampaigns({
            state: campaignState || undefined,
            sort: campaignSort,
            page,
            page_size: 8,
          });
          if (cancelled) return;
          setCampaignData(result);
        }
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [campaignSort, campaignState, page, proposalSort, reviewState, status, view]);

  function updateParams(nextValues: Record<string, string | null>) {
    const next = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(nextValues)) {
      if (!value) next.delete(key);
      else next.set(key, value);
    }
    if (!("page" in nextValues)) next.set("page", "1");
    setSearchParams(next);
  }

  const filteredProposals = useMemo(() => {
    const list = proposalData?.proposals ?? [];
    const needle = keyword.trim().toLowerCase();
    if (!needle) return list;
    return list.filter(
      (item) =>
        item.github_url.toLowerCase().includes(needle) ||
        item.organizer_address.toLowerCase().includes(needle) ||
        String(item.proposal_id).includes(needle),
    );
  }, [keyword, proposalData]);

  const filteredCampaigns = useMemo(() => {
    const list = campaignData?.campaigns ?? [];
    const needle = keyword.trim().toLowerCase();
    if (!needle) return list;
    return list.filter(
      (item) =>
        item.github_url.toLowerCase().includes(needle) ||
        item.organizer_address.toLowerCase().includes(needle) ||
        String(item.campaign_id).includes(needle) ||
        String(item.proposal_id).includes(needle),
    );
  }, [campaignData, keyword]);

  const numericKeyword = /^\d+$/.test(keyword.trim()) ? keyword.trim() : "";

  return (
    <div className="space-y-6">
      <Card>
        <CardContent>
          <SectionIntro
            eyebrow="Explore"
            title="浏览提案与众筹活动"
            description="支持基础筛选、排序、分页，并提供按编号快速直达详情。关键词搜索当前页结果中的 GitHub URL、地址和实体编号。"
            action={
              <>
                <Button
                  variant={view === "proposals" ? "default" : "outline"}
                  onClick={() => updateParams({ view: "proposals", page: "1", state: null })}
                >
                  提案
                </Button>
                <Button
                  variant={view === "campaigns" ? "default" : "outline"}
                  onClick={() => updateParams({ view: "campaigns", page: "1", status: null, review_state: null })}
                >
                  活动
                </Button>
              </>
            }
          />

          <div className="grid gap-4 lg:grid-cols-4">
            <div className="space-y-1">
              <label className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">关键词</label>
              <Input
                value={keyword}
                placeholder={view === "proposals" ? "GitHub URL / 发起人 / Proposal ID" : "GitHub URL / 发起人 / Campaign ID"}
                onChange={(e) => updateParams({ keyword: e.target.value, page: "1" })}
              />
            </div>

            {view === "proposals" ? (
              <>
                <div className="space-y-1">
                  <label className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">提案状态</label>
                  <select className={selectClass} value={status} onChange={(e) => updateParams({ status: e.target.value, page: "1" })}>
                    {proposalStatuses.map((value) => (
                      <option key={value || "all"} value={value}>{value || "全部状态"}</option>
                    ))}
                  </select>
                </div>
                <div className="space-y-1">
                  <label className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">轮次审核状态</label>
                  <select className={selectClass} value={reviewState} onChange={(e) => updateParams({ review_state: e.target.value, page: "1" })}>
                    {["", "pending", "approved", "rejected"].map((value) => (
                      <option key={value || "all"} value={value}>{value || "全部审核态"}</option>
                    ))}
                  </select>
                </div>
                <div className="space-y-1">
                  <label className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">排序</label>
                  <select className={selectClass} value={proposalSort} onChange={(e) => updateParams({ sort: e.target.value, page: "1" })}>
                    <option value="submitted_at_desc">最新提交优先</option>
                    <option value="submitted_at_asc">最早提交优先</option>
                  </select>
                </div>
              </>
            ) : (
              <>
                <div className="space-y-1">
                  <label className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">活动状态</label>
                  <select className={selectClass} value={campaignState} onChange={(e) => updateParams({ state: e.target.value, page: "1" })}>
                    {campaignStates.map((value) => (
                      <option key={value || "all"} value={value}>{value || "全部状态"}</option>
                    ))}
                  </select>
                </div>
                <div className="space-y-1">
                  <label className="text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">排序</label>
                  <select className={selectClass} value={campaignSort} onChange={(e) => updateParams({ sort: e.target.value, page: "1" })}>
                    <option value="launched_at_desc">最新发起优先</option>
                    <option value="deadline_at_asc">最早截止优先</option>
                    <option value="amount_raised_desc">募集金额优先</option>
                  </select>
                </div>
                <Card className="flex flex-col justify-center">
                  <CardContent>
                    <p className="mb-2 text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">快速跳转</p>
                    {numericKeyword ? (
                      <Link
                        to={`/crowdfunding/campaigns/${numericKeyword}`}
                        className={cn(buttonVariants({ size: "sm" }))}
                      >
                        打开 Campaign #{numericKeyword}
                      </Link>
                    ) : (
                      <p className="text-sm text-muted-foreground">输入纯数字可快速直达活动详情。</p>
                    )}
                  </CardContent>
                </Card>
              </>
            )}
          </div>

          {view === "proposals" && numericKeyword ? (
            <div className="mt-4">
              <Link
                to={`/crowdfunding/proposals/${numericKeyword}`}
                className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
              >
                打开 Proposal #{numericKeyword}
              </Link>
            </div>
          ) : null}
        </CardContent>
      </Card>

      {loading ? <LoadingState message="正在加载列表…" /> : null}
      {error ? <ErrorState message={error} /> : null}

      {!loading && !error && view === "proposals" ? (
        <section className="space-y-4">
          {filteredProposals.length > 0 ? (
            <>
              <div className="grid gap-4 xl:grid-cols-2">
                {filteredProposals.map((proposal) => (
                  <ProposalCard key={proposal.proposal_id} proposal={proposal} />
                ))}
              </div>
              {proposalData?.pagination ? (
                <PaginationControls pagination={proposalData.pagination} onPageChange={(nextPage) => updateParams({ page: String(nextPage) })} />
              ) : null}
            </>
          ) : (
            <EmptyState title="没有匹配的提案" description="尝试清空筛选，或直接输入 Proposal 编号跳转到详情。" />
          )}
        </section>
      ) : null}

      {!loading && !error && view === "campaigns" ? (
        <section className="space-y-4">
          {filteredCampaigns.length > 0 ? (
            <>
              <div className="grid gap-4 xl:grid-cols-2">
                {filteredCampaigns.map((campaign) => (
                  <CampaignCard key={campaign.campaign_id} campaign={campaign} />
                ))}
              </div>
              {campaignData?.pagination ? (
                <PaginationControls pagination={campaignData.pagination} onPageChange={(nextPage) => updateParams({ page: String(nextPage) })} />
              ) : null}
            </>
          ) : (
            <EmptyState title="没有匹配的活动" description="尝试调整状态筛选，或输入 Campaign 编号直接跳转。" />
          )}
        </section>
      ) : null}
    </div>
  );
}
