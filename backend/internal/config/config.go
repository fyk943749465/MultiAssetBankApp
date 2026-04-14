package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr                   string
	DatabaseURL                  string
	EthRPCURL                    string
	CounterContract              string // optional: deployed Counter contract address (hex)
	EthPrivateKeyHex             string // optional: hex private key for contract writes (keep secret)
	BankContract                 string // optional: MultiAssetBank for event indexer
	BankIndexerStartBlock        uint64 // optional: first block to scan (0 = use head-2000 on init)
	SubgraphURL                  string // optional: The Graph Studio query URL (bank)
	SubgraphAPIKey               string // optional: Studio API key (Bearer)
	CodePulseAddress             string // optional: CodePulseAdvanced contract address (hex)
	SubgraphCodePulseURL         string // optional: Code Pulse subgraph query URL
	CodePulseSubgraphStartBlock  uint64 // optional: 子图同步起始块（减 1 后作为首次 blockNumber_gt 游标；0 表示从链起点拉取，可能很慢）
	CodePulseSubgraphPollSeconds int    // optional: 子图同步轮询间隔秒数，默认 25
	CodePulseIndexerStartBlock   uint64 // optional: RPC 扫块索引起始块（0 表示与 Bank 相同：首次为 safe 头往前约 2000 块）
	// CodePulseSubgraphSync 为 true 时启用子图→PostgreSQL 增量同步；默认 false，数据库权威数据来自 RPC 扫块（与 Bank 一致）。
	CodePulseSubgraphSync bool
	// 以下为 Bank / Code Pulse RPC 索引器共用，减轻 Infura 等 429：轮询间隔、分块间隔、每段最大块数。
	IndexerPollSeconds        int
	IndexerFilterChunkPauseMs int
	IndexerMaxBlockSpan       uint64
	// CodePulseServerTx 为 true 时开放 POST /api/code-pulse/tx/submit（需 ETH_PRIVATE_KEY）。默认 false：仅钱包签名。
	CodePulseServerTx bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	startBlk, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("BANK_INDEXER_START_BLOCK")), 10, 64)
	cpSubStart, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_START_BLOCK")), 10, 64)
	cpPoll := 25
	if v := strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_POLL_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cpPoll = n
		}
	}
	cpIdxStart, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("CODE_PULSE_INDEXER_START_BLOCK")), 10, 64)
	cpSubSync := strings.EqualFold(strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_SYNC")), "true") ||
		strings.TrimSpace(os.Getenv("CODE_PULSE_SUBGRAPH_SYNC")) == "1"

	idxPoll := 35
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

	return &Config{
		ServerAddr:                   getEnv("SERVER_ADDR", ":8080"),
		DatabaseURL:                  os.Getenv("DATABASE_URL"),
		EthRPCURL:                    os.Getenv("ETH_RPC_URL"),
		CounterContract:              os.Getenv("COUNTER_CONTRACT_ADDRESS"),
		EthPrivateKeyHex:             os.Getenv("ETH_PRIVATE_KEY"),
		BankContract:                 os.Getenv("BANK_CONTRACT_ADDRESS"),
		BankIndexerStartBlock:        startBlk,
		SubgraphURL:                  os.Getenv("SUBGRAPH_URL"),
		SubgraphAPIKey:               os.Getenv("SUBGRAPH_API_KEY"),
		CodePulseAddress:             os.Getenv("CODE_PULSE_ADDRESS"),
		SubgraphCodePulseURL:         os.Getenv("SUBGRAPH_CODE_PULSE_URL"),
		CodePulseSubgraphStartBlock:  cpSubStart,
		CodePulseSubgraphPollSeconds: cpPoll,
		CodePulseIndexerStartBlock:   cpIdxStart,
		CodePulseSubgraphSync:        cpSubSync,
		IndexerPollSeconds:           idxPoll,
		IndexerFilterChunkPauseMs:    idxPauseMs,
		IndexerMaxBlockSpan:          idxSpan,
		CodePulseServerTx:            cpServerTx,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
