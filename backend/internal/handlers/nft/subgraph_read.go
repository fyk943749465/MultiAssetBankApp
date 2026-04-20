package nft

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-chain/backend/internal/subgraph"
)

// 子图仅用于只读兜底展示，不向 PostgreSQL 写入；链重组后子图可能短暂与链不一致。
const qNFTFallbackCollections = `
query NftFallbackCollections($first: Int!, $skip: Int!) {
  collectionCreateds(first: $first, skip: $skip, orderBy: blockNumber, orderDirection: desc) {
    id collection creator feePaid salt blockNumber blockTimestamp transactionHash
  }
}
`

const qNFTCollectionCreatedByContract = `
query NftCollectionCreatedByContract($addr: Bytes!) {
  collectionCreateds(where: { collection: $addr }, first: 10, orderBy: blockNumber, orderDirection: desc) {
    id collection creator feePaid salt blockNumber blockTimestamp transactionHash
  }
}
`

// SubgraphCollectionView 子图 CollectionCreated 映射（无 PG 主键）。
type SubgraphCollectionView struct {
	SubgraphEntityID  string `json:"subgraph_entity_id"`
	CollectionAddress string `json:"collection_address"`
	CreatorAddress    string `json:"creator_address"`
	FeePaidWei        string `json:"fee_paid_wei"`
	SaltHex           string `json:"salt_hex,omitempty"`
	BlockNumber       string `json:"block_number"`
	BlockTimestamp    string `json:"block_timestamp"`
	TransactionHash   string `json:"transaction_hash"`
}

// SubgraphListingView 子图 ItemListed 映射（非 nft_active_listings 表结构）。
type SubgraphListingView struct {
	SubgraphEntityID  string `json:"subgraph_entity_id"`
	CollectionAddress string `json:"collection_address"`
	TokenID           string `json:"token_id"`
	SellerAddress     string `json:"seller_address"`
	PriceWei          string `json:"price_wei"`
	BlockNumber       string `json:"block_number"`
	BlockTimestamp    string `json:"block_timestamp"`
	TransactionHash   string `json:"transaction_hash"`
}

func fetchSubgraphCollectionsFallback(ctx context.Context, sg *subgraph.Client, page, pageSize int) ([]SubgraphCollectionView, bool, error) {
	if sg == nil || !sg.Configured() {
		return nil, false, fmt.Errorf("subgraph not configured")
	}
	first := pageSize + 1
	skip := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	raw, err := sg.Query(ctx, qNFTFallbackCollections, map[string]any{"first": first, "skip": skip})
	if err != nil {
		return nil, false, err
	}
	var wrap struct {
		CollectionCreateds []struct {
			ID              string `json:"id"`
			Collection      string `json:"collection"`
			Creator         string `json:"creator"`
			FeePaid         string `json:"feePaid"`
			Salt            string `json:"salt"`
			BlockNumber     string `json:"blockNumber"`
			BlockTimestamp  string `json:"blockTimestamp"`
			TransactionHash string `json:"transactionHash"`
		} `json:"collectionCreateds"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, false, fmt.Errorf("decode subgraph: %w", err)
	}
	// subgraph.Client.Query 返回的是响应中的 data 对象（与 Code Pulse 子图解析一致），不是再包一层 "data"。
	hasMore := len(wrap.CollectionCreateds) > pageSize
	n := pageSize
	if len(wrap.CollectionCreateds) < n {
		n = len(wrap.CollectionCreateds)
	}
	out := make([]SubgraphCollectionView, 0, n)
	for i := 0; i < n; i++ {
		r := wrap.CollectionCreateds[i]
		out = append(out, SubgraphCollectionView{
			SubgraphEntityID:  r.ID,
			CollectionAddress: normHexAddr(r.Collection),
			CreatorAddress:    normHexAddr(r.Creator),
			FeePaidWei:        strings.TrimSpace(r.FeePaid),
			SaltHex:           normHexBytes(r.Salt),
			BlockNumber:       strings.TrimSpace(r.BlockNumber),
			BlockTimestamp:    strings.TrimSpace(r.BlockTimestamp),
			TransactionHash:   normHexBytes(r.TransactionHash),
		})
	}
	return out, hasMore, nil
}

func fetchSubgraphCollectionsByContractAddress(ctx context.Context, sg *subgraph.Client, collectionAddr string) ([]SubgraphCollectionView, error) {
	if sg == nil || !sg.Configured() {
		return nil, fmt.Errorf("subgraph not configured")
	}
	addr := normHexAddr(collectionAddr)
	if addr == "" {
		return nil, fmt.Errorf("empty collection address")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	raw, err := sg.Query(ctx, qNFTCollectionCreatedByContract, map[string]any{"addr": addr})
	if err != nil {
		return nil, err
	}
	var wrap struct {
		CollectionCreateds []struct {
			ID              string `json:"id"`
			Collection      string `json:"collection"`
			Creator         string `json:"creator"`
			FeePaid         string `json:"feePaid"`
			Salt            string `json:"salt"`
			BlockNumber     string `json:"blockNumber"`
			BlockTimestamp  string `json:"blockTimestamp"`
			TransactionHash string `json:"transactionHash"`
		} `json:"collectionCreateds"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("decode subgraph: %w", err)
	}
	out := make([]SubgraphCollectionView, 0, len(wrap.CollectionCreateds))
	for _, r := range wrap.CollectionCreateds {
		out = append(out, SubgraphCollectionView{
			SubgraphEntityID:  r.ID,
			CollectionAddress: normHexAddr(r.Collection),
			CreatorAddress:    normHexAddr(r.Creator),
			FeePaidWei:        strings.TrimSpace(r.FeePaid),
			SaltHex:           normHexBytes(r.Salt),
			BlockNumber:       strings.TrimSpace(r.BlockNumber),
			BlockTimestamp:    strings.TrimSpace(r.BlockTimestamp),
			TransactionHash:   normHexBytes(r.TransactionHash),
		})
	}
	return out, nil
}

func fetchSubgraphListingsFallback(ctx context.Context, sg *subgraph.Client, page, pageSize int) ([]SubgraphListingView, bool, error) {
	if sg == nil || !sg.Configured() {
		return nil, false, fmt.Errorf("subgraph not configured")
	}
	first := pageSize + 1
	skip := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	field, err := resolveNFTMarketListingsCollectionField(ctx, sg)
	if err != nil {
		return nil, false, err
	}
	q := fmt.Sprintf(`query NftFallbackListings($first: Int!, $skip: Int!) {
  %s(first: $first, skip: $skip, orderBy: blockNumber, orderDirection: desc) {
    id collection tokenId seller price blockNumber blockTimestamp transactionHash
  }
}`, field)

	raw, err := sg.Query(ctx, q, map[string]any{"first": first, "skip": skip})
	if err != nil {
		return nil, false, err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, false, fmt.Errorf("decode subgraph: %w", err)
	}
	arrBytes, ok := top[field]
	if !ok {
		return nil, false, fmt.Errorf("decode subgraph: missing field %q in response", field)
	}
	var rows []struct {
		ID              string `json:"id"`
		Collection      string `json:"collection"`
		TokenID         string `json:"tokenId"`
		Seller          string `json:"seller"`
		Price           string `json:"price"`
		BlockNumber     string `json:"blockNumber"`
		BlockTimestamp  string `json:"blockTimestamp"`
		TransactionHash string `json:"transactionHash"`
	}
	if err := json.Unmarshal(arrBytes, &rows); err != nil {
		return nil, false, fmt.Errorf("decode subgraph listings: %w", err)
	}
	hasMore := len(rows) > pageSize
	n := pageSize
	if len(rows) < n {
		n = len(rows)
	}
	out := make([]SubgraphListingView, 0, n)
	for i := 0; i < n; i++ {
		r := rows[i]
		out = append(out, SubgraphListingView{
			SubgraphEntityID:  r.ID,
			CollectionAddress: normHexAddr(r.Collection),
			TokenID:           strings.TrimSpace(r.TokenID),
			SellerAddress:     normHexAddr(r.Seller),
			PriceWei:          strings.TrimSpace(r.Price),
			BlockNumber:       strings.TrimSpace(r.BlockNumber),
			BlockTimestamp:    strings.TrimSpace(r.BlockTimestamp),
			TransactionHash:   normHexBytes(r.TransactionHash),
		})
	}
	return out, hasMore, nil
}

func normHexAddr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	return strings.ToLower(s)
}

func normHexBytes(s string) string {
	s = strings.Trim(strings.TrimSpace(s), `"`)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	return strings.ToLower(s)
}
