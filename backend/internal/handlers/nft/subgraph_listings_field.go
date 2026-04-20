package nft

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-chain/backend/internal/subgraph"
)

// 与 subgraph/nft-platform/schema.graphql 中实体 NFTMarketItemListed 对应；根 Query 上的集合字段名由运行时生成，不同 graph-node / 托管 API 可能不同。
const qSubgraphIntrospectQueryRootFields = `query NftSubgraphIntrospectQueryRoot { __schema { queryType { fields { name } } } }`

var (
	listingsCollectionFieldMu     sync.Mutex
	listingsCollectionFieldByHost = map[string]string{} // cache key -> GraphQL collection field name
)

// cacheKeyVersion 变更可丢弃历史上误缓存的单条查询字段名（如 nftMarketItemListed，需 id 而非 first/skip）。
const listingsFieldCacheKeyVersion = "v3-plural-only"

func resolveNFTMarketListingsCollectionField(ctx context.Context, sg *subgraph.Client) (string, error) {
	if sg == nil || !sg.Configured() {
		return "", fmt.Errorf("subgraph not configured")
	}
	key := sg.EndpointHostPath() + "\x00" + listingsFieldCacheKeyVersion
	listingsCollectionFieldMu.Lock()
	if v, ok := listingsCollectionFieldByHost[key]; ok {
		listingsCollectionFieldMu.Unlock()
		return v, nil
	}
	listingsCollectionFieldMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 18*time.Second)
	defer cancel()

	picked, err := detectNFTMarketListingsCollectionField(ctx, sg)
	if err != nil {
		return "", err
	}

	listingsCollectionFieldMu.Lock()
	listingsCollectionFieldByHost[key] = picked
	listingsCollectionFieldMu.Unlock()
	return picked, nil
}

func detectNFTMarketListingsCollectionField(ctx context.Context, sg *subgraph.Client) (string, error) {
	var introErr error
	raw, err := sg.Query(ctx, qSubgraphIntrospectQueryRootFields, nil)
	if err == nil {
		names, perr := parseSubgraphQueryRootFieldNames(raw)
		if perr != nil {
			introErr = perr
		} else if name := pickNFTMarketItemListedCollectionField(names); name != "" {
			return name, nil
		}
	} else {
		introErr = err
	}

	// 部分托管方关闭 introspection，或命名不在启发式内：用轻量查询探测合法字段名。
	for _, field := range []string{"nftMarketItemListeds", "nFTMarketItemListeds", "NFTMarketItemListeds"} {
		q := fmt.Sprintf(`query NftListingsFieldProbe($first: Int!, $skip: Int!) {
  %s(first: $first, skip: $skip, orderBy: blockNumber, orderDirection: desc) { id }
}`, field)
		if _, err := sg.Query(ctx, q, map[string]any{"first": 1, "skip": 0}); err == nil {
			return field, nil
		}
	}
	if introErr != nil {
		return "", fmt.Errorf("listings field: introspection failed and probe failed: %w", introErr)
	}
	return "", fmt.Errorf("listings field: no Query field matched NFTMarketItemListed collection (introspection returned no candidate); check SUBGRAPH_NFT_URL points to subgraph/nft-platform deployment")
}

func parseSubgraphQueryRootFieldNames(data json.RawMessage) ([]string, error) {
	var wrap struct {
		Schema struct {
			QueryType struct {
				Fields []struct {
					Name string `json:"name"`
				} `json:"fields"`
			} `json:"queryType"`
		} `json:"__schema"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(wrap.Schema.QueryType.Fields))
	for _, f := range wrap.Schema.QueryType.Fields {
		if strings.TrimSpace(f.Name) != "" {
			out = append(out, f.Name)
		}
	}
	return out, nil
}

// looksLikeItemListedPluralCollection 为 true 时表示可用 first/skip 的集合字段。
// 单条查询 nftMarketItemListed(id: …) 也包含子串 "itemlisted"，误选会导致 “No value provided for required argument: id”。
func looksLikeItemListedPluralCollection(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "itemlisteds") {
		return true
	}
	if strings.HasSuffix(lower, "_collection") && strings.Contains(lower, "itemlisted") {
		return true
	}
	return false
}

func pickNFTMarketItemListedCollectionField(names []string) string {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	for _, cand := range []string{"nftMarketItemListeds", "nFTMarketItemListeds", "NFTMarketItemListeds"} {
		if _, ok := set[cand]; ok {
			return cand
		}
	}
	var plural []string
	for _, n := range names {
		if looksLikeItemListedPluralCollection(n) {
			plural = append(plural, n)
		}
	}
	for _, n := range plural {
		if strings.Contains(strings.ToLower(n), "nftmarket") {
			return n
		}
	}
	if len(plural) > 0 {
		return plural[0]
	}
	return ""
}
