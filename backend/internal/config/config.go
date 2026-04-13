package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr              string
	DatabaseURL             string
	EthRPCURL               string
	CounterContract         string // optional: deployed Counter contract address (hex)
	EthPrivateKeyHex        string // optional: hex private key for contract writes (keep secret)
	BankContract            string // optional: MultiAssetBank for event indexer
	BankIndexerStartBlock   uint64 // optional: first block to scan (0 = use head-2000 on init)
	SubgraphURL             string // optional: The Graph Studio query URL (bank)
	SubgraphAPIKey          string // optional: Studio API key (Bearer)
	CodePulseAddress        string // optional: CodePulseAdvanced contract address (hex)
	SubgraphCodePulseURL    string // optional: Code Pulse subgraph query URL
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	startBlk, _ := strconv.ParseUint(strings.TrimSpace(os.Getenv("BANK_INDEXER_START_BLOCK")), 10, 64)

	return &Config{
		ServerAddr:            getEnv("SERVER_ADDR", ":8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		EthRPCURL:             os.Getenv("ETH_RPC_URL"),
		CounterContract:       os.Getenv("COUNTER_CONTRACT_ADDRESS"),
		EthPrivateKeyHex:      os.Getenv("ETH_PRIVATE_KEY"),
		BankContract:          os.Getenv("BANK_CONTRACT_ADDRESS"),
		BankIndexerStartBlock: startBlk,
		SubgraphURL:           os.Getenv("SUBGRAPH_URL"),
		SubgraphAPIKey:        os.Getenv("SUBGRAPH_API_KEY"),
		CodePulseAddress:      os.Getenv("CODE_PULSE_ADDRESS"),
		SubgraphCodePulseURL:  os.Getenv("SUBGRAPH_CODE_PULSE_URL"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
