package lending

import (
	"context"
	"encoding/json"
	"fmt"

	"go-chain/backend/internal/subgraph"
)

const suppliesGraphQL = `query($first:Int!,$skip:Int!){
  supplies(first:$first,skip:$skip,orderBy:blockTimestamp,orderDirection:desc){
    id
    asset
    user
    amount
    blockNumber
    blockTimestamp
    transactionHash
  }
}`

// SubgraphSupplyRow 子图 Supply 实体 JSON 形态（字段名与 The Graph 生成一致）。
type SubgraphSupplyRow struct {
	ID               string `json:"id"`
	Asset            string `json:"asset"`
	User             string `json:"user"`
	Amount           string `json:"amount"`
	BlockNumber      string `json:"blockNumber"`
	BlockTimestamp   string `json:"blockTimestamp"`
	TransactionHash  string `json:"transactionHash"`
}

func fetchSubgraphSupplies(ctx context.Context, cl *subgraph.Client, first, skip int) ([]SubgraphSupplyRow, error) {
	raw, err := cl.Query(ctx, suppliesGraphQL, map[string]any{"first": first, "skip": skip})
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Supplies []SubgraphSupplyRow `json:"supplies"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("decode supplies: %w", err)
	}
	if wrap.Supplies == nil {
		return nil, nil
	}
	return wrap.Supplies, nil
}
