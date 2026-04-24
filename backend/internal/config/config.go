package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr            string
	DatabaseURL           string
	EthRPCURL             string
	CounterContract       string // optional: deployed Counter contract address (hex)
	EthPrivateKeyHex      string // optional: hex private key for contract writes (keep secret)
	BankContract          string // optional: MultiAssetBank for event indexer
	BankIndexerStartBlock uint64 // optional: first block to scan (0 = use head-2000 on init)
	SubgraphURL           string // optional: The Graph Studio query URL (bank)
	SubgraphAPIKey        string // optional: Studio API key (Bearer)
	// SubgraphQueryCacheTTLSeconds：子图 GraphQL 响应内存缓存秒数（相同 query+variables 命中则不发 HTTP）。0=关闭。默认 30，减轻 Studio 免费档每日约 3000 次查询压力。
	SubgraphQueryCacheTTLSeconds int
	// SubgraphQueryCacheMaxEntries：缓存条数上限（仅 TTL>0 时生效），默认 512。
	SubgraphQueryCacheMaxEntries int
	CodePulseAddress             string // optional: CodePulseAdvanced contract address (hex)
	SubgraphCodePulseURL         string // optional: Code Pulse subgraph query URL
	CodePulseSubgraphStartBlock  uint64 // optional: 子图同步起始块（减 1 后作为首次 blockNumber_gt 游标；0 表示从链起点拉取，可能很慢）
	CodePulseSubgraphPollSeconds int    // optional: 子图同步轮询间隔秒数，默认 90（约 960 次/天，为 API 留出 Studio 配额）
	CodePulseIndexerStartBlock   uint64 // optional: RPC 扫块索引起始块（0 表示与 Bank 相同：首次为 safe 头往前约 2000 块）
	// CodePulseSubgraphSync 为 true 时启用子图→PostgreSQL 增量同步；默认 false，数据库权威数据来自 RPC 扫块（与 Bank 一致）。
	CodePulseSubgraphSync bool
	// 以下为 Bank / Code Pulse RPC 索引器共用，减轻 Infura 等 429：轮询间隔、分块间隔、每段最大块数。
	// IndexerPollSeconds：扫块轮询秒数，默认 120（上界多为 finalized/safe，分钟级才前进，过频多为空转）；环境变量 INDEXER_POLL_SECONDS 可覆盖，≥5。
	IndexerPollSeconds        int
	IndexerFilterChunkPauseMs int
	IndexerMaxBlockSpan       uint64
	// CodePulseServerTx 为 true 时开放 POST /api/code-pulse/tx/submit（需 ETH_PRIVATE_KEY）。默认 false：仅钱包签名。
	CodePulseServerTx bool
	// CodePulseInitiatorReconcileSeconds 将 cp_wallet_roles 中全局 proposal_initiator 与链上白名单定时对齐（秒）；0 表示关闭。
	// 子图可用时按子图事件折叠结果写库；子图失败时用合约 isProposalInitiator 刷新库中已出现过的地址。
	CodePulseInitiatorReconcileSeconds int
	SubgraphNftURL                     string // optional: NFT platform subgraph (The Graph)
	NftSubgraphPollSeconds             int    // optional: indexer poll interval, default 35
	NftSubgraphStartBlock              uint64 // optional: first blockNumber_gt cursor = start-1
	SubgraphLendingURL                 string // optional: lending subgraph (The Graph); supplies list may prefer when non-empty
	LendingChainID                     int64  // optional: lending API / tables default chain_id (0 = handlers default 84532)
	// LendingSubgraphAPIKey：借贷子图专用 Studio Bearer；来自 SUBGRAPH_LENDING_API_KEY，否则 SUBGRAPH_API_SECOND_KEY。绝不使用 SUBGRAPH_API_KEY。
	LendingSubgraphAPIKey       string
	LendingSubgraphAPIKeySource string // which env won: SUBGRAPH_LENDING_API_KEY | SUBGRAPH_API_SECOND_KEY | "" (empty if no key)
	// LendingEthRPCURL：借贷专用 JSON-RPC；LENDING_ETH_RPC_URL 优先，否则 BASE_ETH_RPC_URL。与 EthRPCURL（银行/Code Pulse 等）隔离。
	LendingEthRPCURL string
	// LendingIndexerStartBlock：借贷 RPC 索引起始块（0=首次游标为已确认头往前约 2000 块，与 Bank 一致）。
	LendingIndexerStartBlock uint64
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	startBlk, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("BANK_INDEXER_START_BLOCK")), 10, 64)
	cpSubStart, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_START_BLOCK")), 10, 64)
	cpPoll := 90
	if v := strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_POLL_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cpPoll = n
		}
	}

	sgCacheTTL := 30
	if v := strings.TrimSpace(os.Getenv("SUBGRAPH_QUERY_CACHE_TTL_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			sgCacheTTL = n
		}
	}
	sgCacheMax := 512
	if v := strings.TrimSpace(os.Getenv("SUBGRAPH_QUERY_CACHE_MAX_ENTRIES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			sgCacheMax = n
		}
	}
	cpIdxStart, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("CODE_PULSE_INDEXER_START_BLOCK")), 10, 64)
	cpSubSync := strings.EqualFold(strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_SYNC")), "true") ||
		strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_SYNC")) == "1"

	idxPoll := 120
	if v := strings.TrimSpace(os.Getenv("INDEXER_POLL_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 5 {
			idxPoll = n
		}
	}
	idxPauseMs := 800
	if v := strings.TrimSpace(os.Getenv("INDEXER_FILTER_CHUNK_PAUSE_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			idxPauseMs = n
		}
	}
	idxSpan := uint64(1000)
	if v := strings.TrimSpace(os.Getenv("INDEXER_MAX_BLOCK_SPAN")); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil && n >= 50 && n <= 10000 {
			idxSpan = n
		}
	}

	cpServerTx := strings.EqualFold(strings.TrimSpace(os.Getenv("CODE_PULSE_SERVER_TX")), "true") ||
		strings.TrimSpace(os.Getenv("CODE_PULSE_SERVER_TX")) == "1"

	cpInitRec := 0
	if v := strings.TrimSpace(os.Getenv("CODE_PULSE_INITIATOR_RECONCILE_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cpInitRec = n
		}
	}

	nftSubPoll := 35
	if v := strings.TrimSpace(os.Getenv("NFT_SUBGRAPH_POLL_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 5 {
			nftSubPoll = n
		}
	}
	nftSubStart, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("NFT_SUBGRAPH_START_BLOCK")), 10, 64)

	lendingChainID := int64(0)
	if v := strings.TrimSpace(os.Getenv("LENDING_CHAIN_ID")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			lendingChainID = n
		}
	}

	lendingSubKey := strings.TrimSpace(os.Getenv("SUBGRAPH_LENDING_API_KEY"))
	lendingSubKeySource := ""
	if lendingSubKey != "" {
		lendingSubKeySource = "SUBGRAPH_LENDING_API_KEY"
	} else {
		lendingSubKey = strings.TrimSpace(os.Getenv("SUBGRAPH_API_SECOND_KEY"))
		if lendingSubKey != "" {
			lendingSubKeySource = "SUBGRAPH_API_SECOND_KEY"
		}
	}

	lendingRPC := strings.TrimSpace(os.Getenv("LENDING_ETH_RPC_URL"))
	if lendingRPC == "" {
		lendingRPC = strings.TrimSpace(os.Getenv("BASE_ETH_RPC_URL"))
	}

	lendingIdxStart, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("LENDING_INDEXER_START_BLOCK")), 10, 64)

	return &Config{
		ServerAddr:                         getEnv("SERVER_ADDR", ":8080"),
		DatabaseURL:                        os.Getenv("DATABASE_URL"),
		EthRPCURL:                          os.Getenv("ETH_RPC_URL"),
		CounterContract:                    os.Getenv("COUNTER_CONTRACT_ADDRESS"),
		EthPrivateKeyHex:                   os.Getenv("ETH_PRIVATE_KEY"),
		BankContract:                       os.Getenv("BANK_CONTRACT_ADDRESS"),
		BankIndexerStartBlock:              startBlk,
		SubgraphURL:                        os.Getenv("SUBGRAPH_URL"),
		SubgraphAPIKey:                     os.Getenv("SUBGRAPH_API_KEY"),
		SubgraphQueryCacheTTLSeconds:       sgCacheTTL,
		SubgraphQueryCacheMaxEntries:       sgCacheMax,
		CodePulseAddress:                   os.Getenv("CODE_PULSE_ADDRESS"),
		SubgraphCodePulseURL:               os.Getenv("SUBGRAPH_CODE_PULSE_URL"),
		CodePulseSubgraphStartBlock:        cpSubStart,
		CodePulseSubgraphPollSeconds:       cpPoll,
		CodePulseIndexerStartBlock:         cpIdxStart,
		CodePulseSubgraphSync:              cpSubSync,
		IndexerPollSeconds:                 idxPoll,
		IndexerFilterChunkPauseMs:          idxPauseMs,
		IndexerMaxBlockSpan:                idxSpan,
		CodePulseServerTx:                  cpServerTx,
		CodePulseInitiatorReconcileSeconds: cpInitRec,
		SubgraphNftURL:                     strings.TrimSpace(os.Getenv("SUBGRAPH_NFT_URL")),
		NftSubgraphPollSeconds:             nftSubPoll,
		NftSubgraphStartBlock:              nftSubStart,
		SubgraphLendingURL:                 strings.TrimSpace(os.Getenv("SUBGRAPH_LENDING_URL")),
		LendingChainID:                     lendingChainID,
		LendingSubgraphAPIKey:              lendingSubKey,
		LendingSubgraphAPIKeySource:        lendingSubKeySource,
		LendingEthRPCURL:                   lendingRPC,
		LendingIndexerStartBlock:         lendingIdxStart,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
