import type { CPCampaign, CPProposal } from "./types";

function addrEq(a?: string | null, b?: string | null): boolean {
  if (!a || !b) return false;
  return a.toLowerCase() === b.toLowerCase();
}

function weiBI(s: string | undefined | null): bigint {
  try {
    return BigInt((s ?? "").trim() || "0");
  } catch {
    return 0n;
  }
}

export function lastSortedCampaign(campaigns: CPCampaign[]): CPCampaign | null {
  if (!campaigns.length) return null;
  return [...campaigns].sort((a, b) => a.round_index - b.round_index).at(-1) ?? null;
}

/** 近似链上 `_isCampaignSettled`（依赖读模型字段） */
export function isCampaignSettledOnChain(c: CPCampaign): boolean {
  if (c.state === "fundraising") return false;
  if (c.state === "failed_refundable") return weiBI(c.unclaimed_refund_pool_wei) === 0n;
  if (c.state === "successful") return weiBI(c.total_withdrawn_wei) === weiBI(c.amount_raised_wei);
  return false;
}

export type OrganizerActionKey =
  | "submit_first_round_for_review"
  | "launch_approved_round"
  | "submit_follow_on_round_for_review";

export type AdminActionKey = "review_proposal_approve" | "review_proposal_reject" | "review_funding_round";

export type ProposalFlowComputation = {
  phaseHeadline: string;
  phaseDetail: string;
  organizerActions: OrganizerActionKey[];
  adminActions: AdminActionKey[];
};

/**
 * 根据提案与 campaign 读模型推导「当前处于哪一步」以及各角色应看到的动作（不含钱包过滤）。
 */
export function computeProposalFlow(proposal: CPProposal, campaigns: CPCampaign[]): ProposalFlowComputation {
  const st = proposal.status;
  const rnd = proposal.round_review_state ?? null;

  if (st === "rejected") {
    return {
      phaseHeadline: "提案未通过",
      phaseDetail: "该提案已被管理员拒绝，链上不再进入众筹流程。",
      organizerActions: [],
      adminActions: [],
    };
  }

  if (st === "pending_review") {
    return {
      phaseHeadline: "等待管理员审核提案",
      phaseDetail: "发起人已提交提案。管理员通过或拒绝后，才会进入「提交众筹轮次」流程。",
      organizerActions: [],
      adminActions: ["review_proposal_approve", "review_proposal_reject"],
    };
  }

  const hasLiveFundraising = campaigns.some((c) => c.state === "fundraising");
  const last = lastSortedCampaign(campaigns);
  const roundsEverLaunched = proposal.current_round_count > 0;

  if (rnd === "round_review_pending") {
    return {
      phaseHeadline: "等待管理员审核本轮众筹",
      phaseDetail: "发起人已将本轮目标、周期与里程碑提交审核。通过后需由发起人再发起 `launch` 才会开捐。",
      organizerActions: [],
      adminActions: ["review_funding_round"],
    };
  }

  if (rnd === "round_review_approved") {
    return {
      phaseHeadline: "本轮已通过审核，等待发起人上线众筹",
      phaseDetail: "请发起人发送「发起已批准轮次」交易；上线后捐款入口在对应 campaign 页面。",
      organizerActions: ["launch_approved_round"],
      adminActions: [],
    };
  }

  if (hasLiveFundraising) {
    return {
      phaseHeadline: "本轮众筹进行中",
      phaseDetail: "提案侧暂无需操作。捐款、追加开发者等请在下方「关联众筹轮次」进入具体 campaign。",
      organizerActions: [],
      adminActions: [],
    };
  }

  if (last && !isCampaignSettledOnChain(last)) {
    if (last.state === "failed_refundable" && weiBI(last.unclaimed_refund_pool_wei) > 0n) {
      return {
        phaseHeadline: "上一轮失败，仍有待退款",
        phaseDetail: "需等待捐款人领取退款、退款池归零后，才可提交下一轮众筹审核。",
        organizerActions: [],
        adminActions: [],
      };
    }
    if (last.state === "successful" && weiBI(last.total_withdrawn_wei) < weiBI(last.amount_raised_wei)) {
      return {
        phaseHeadline: "上一轮已成功，里程碑款未全部结清",
        phaseDetail: "请先在本轮 campaign 上完成里程碑批准与开发者领取；资金侧结清后才可发起下一轮众筹。",
        organizerActions: [],
        adminActions: [],
      };
    }
  }

  const roundBlocked =
    rnd === "round_review_rejected"
      ? "上一轮众筹参数未通过审核，请发起人重新提交（首轮可再次「提交首轮审核」；后续轮请重新填写表单）。"
      : null;

  if (!roundsEverLaunched) {
    return {
      phaseHeadline: "提案已通过，等待发起人提交首轮众筹审核",
      phaseDetail:
        roundBlocked ??
        "发起人需将提案中已填写的目标/周期/里程碑正式提交给管理员审本轮；通过后还需再「上线」才会开捐。",
      organizerActions: ["submit_first_round_for_review"],
      adminActions: [],
    };
  }

  if (last && isCampaignSettledOnChain(last)) {
    return {
      phaseHeadline: "上一轮已结清，可发起下一轮众筹",
      phaseDetail:
        roundBlocked ??
        "请发起人填写新一轮目标、周期与三条里程碑说明并提交审核；通过后同样需管理员审本轮、再上线。",
      organizerActions: ["submit_follow_on_round_for_review"],
      adminActions: [],
    };
  }

  return {
    phaseHeadline: "等待链上状态推进",
    phaseDetail: "当前读模型与预期不一致，或索引尚未同步。可稍后在管理台查看同步游标。",
    organizerActions: [],
    adminActions: [],
  };
}

export function isProposalOrganizer(wallet: string | undefined, proposal: CPProposal): boolean {
  return addrEq(wallet, proposal.organizer_address);
}

export function isContractOwnerWallet(wallet: string | undefined, ownerAddress?: string | null): boolean {
  return addrEq(wallet, ownerAddress);
}
