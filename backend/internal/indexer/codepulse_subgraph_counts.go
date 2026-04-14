package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"go-chain/backend/internal/subgraph"

	"golang.org/x/sync/errgroup"
)

// 与 codePulseSyncQuery / codePulseAdminFeedQuery 中的实体集合一致（每类一条链上事件对应子图一条记录）。
var codePulseSubgraphEntityCollections = []string{
	"proposalSubmitteds",
	"proposalRevieweds",
	"fundingRoundSubmittedForReviews",
	"fundingRoundRevieweds",
	"crowdfundingLauncheds",
	"donateds",
	"campaignFinalizeds",
	"refundClaimeds",
	"developerAddeds",
	"developerRemoveds",
	"milestoneApproveds",
	"milestoneShareClaimeds",
	"platformDonateds",
	"platformFundsWithdrawns",
	"ownershipTransferreds",
	"pauseds",
	"unpauseds",
	"proposalInitiatorUpdateds",
	"staleFundsSwepts",
}

const subgraphCountPage = 1000

// CountCodePulseSubgraphEventEntities 统计子图中各类实体条数之和，与链上已索引事件数量一致（子图自部署块起通常完整）。
func CountCodePulseSubgraphEventEntities(ctx context.Context, sg *subgraph.Client) (int, error) {
	if sg == nil || !sg.Configured() {
		return 0, fmt.Errorf("code-pulse subgraph not configured")
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	var total int64
	for _, c := range codePulseSubgraphEntityCollections {
		name := c
		g.Go(func() error {
			n, err := countSubgraphCollection(ctx, sg, name)
			if err != nil {
				return err
			}
			atomic.AddInt64(&total, int64(n))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return 0, err
	}
	return int(total), nil
}

func countSubgraphCollection(ctx context.Context, sg *subgraph.Client, collection string) (int, error) {
	// orderBy + skip 分页，避免大 skip 时部分节点不稳定时可再改为 id 游标。
	q := fmt.Sprintf(
		`query($first:Int!,$skip:Int!){ %s(first:$first,skip:$skip,orderBy:blockNumber,orderDirection:asc){ id } }`,
		collection,
	)
	n := 0
	skip := 0
	for {
		data, err := sg.Query(ctx, q, map[string]any{"first": subgraphCountPage, "skip": skip})
		if err != nil {
			return n, err
		}
		var wrap map[string][]struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &wrap); err != nil {
			return n, err
		}
		rows := wrap[collection]
		n += len(rows)
		if len(rows) < subgraphCountPage {
			break
		}
		skip += subgraphCountPage
		if skip > 500000 {
			break
		}
	}
	return n, nil
}
