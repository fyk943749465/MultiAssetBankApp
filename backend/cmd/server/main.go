// @title           go-chain API
// @version         0.1.0
// @description     REST API: health, chain status, counter contract calls, bank ledger (PostgreSQL indexer + optional The Graph subgraph).
// @BasePath        /
//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -d .,../../internal/handlers,../../internal/handlers/system,../../internal/handlers/chain,../../internal/handlers/bank,../../internal/handlers/contract,../../internal/handlers/codepulse -g main.go -o ../../docs --parseDependency --parseInternal

package main

import (
	"context"
	"crypto/ecdsa"
	"log"
	"strings"

	_ "go-chain/backend/docs"

	"go-chain/backend/internal/chain"
	"go-chain/backend/internal/config"
	"go-chain/backend/internal/contracts"
	"go-chain/backend/internal/database"
	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/indexer"
	"go-chain/backend/internal/router"
	"go-chain/backend/internal/subgraph"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	ethClient, err := chain.Dial(cfg.EthRPCURL)
	if err != nil {
		log.Printf("chain: ETH_RPC_URL 已配置但连接失败，链上接口与 bank 索引将不可用: %v", err)
		ethClient = nil
	}
	if ethClient != nil {
		defer ethClient.Close()
	}

	var counter *contracts.Counter
	if ethClient != nil && cfg.CounterContract != "" {
		if !common.IsHexAddress(cfg.CounterContract) {
			log.Printf("counter: invalid COUNTER_CONTRACT_ADDRESS")
		} else {
			addr := common.HexToAddress(cfg.CounterContract)
			var errCtr error
			counter, errCtr = contracts.NewCounter(ethClient.Eth(), addr)
			if errCtr != nil {
				log.Printf("counter: bind contract: %v", errCtr)
				counter = nil
			}
		}
	}

	var txKey *ecdsa.PrivateKey
	if k := strings.TrimSpace(cfg.EthPrivateKeyHex); k != "" {
		k = strings.TrimPrefix(strings.TrimPrefix(k, "0x"), "0X")
		pk, errPK := crypto.HexToECDSA(k)
		if errPK != nil {
			//log.Printf("ETH_PRIVATE_KEY invalid: %v", errPK)
			log.Printf("ETH_PRIVATE_KEY invalid")
		} else {
			txKey = pk
		}
	}

	var subClient *subgraph.Client
	if u := strings.TrimSpace(cfg.SubgraphURL); u != "" {
		subClient = subgraph.New(u, cfg.SubgraphAPIKey)
		log.Printf("subgraph: 已配置 SUBGRAPH_URL（充值/提现子图 API 可用）")
	}

	var cpSubClient *subgraph.Client
	if u := strings.TrimSpace(cfg.SubgraphCodePulseURL); u != "" {
		cpSubClient = subgraph.New(u, cfg.SubgraphAPIKey)
		log.Printf("subgraph: 已配置 SUBGRAPH_CODE_PULSE_URL（众筹子图 API 可用）")
	}

	var codePulse *contracts.CodePulse
	if a := strings.TrimSpace(cfg.CodePulseAddress); a != "" {
		if !common.IsHexAddress(a) {
			log.Printf("code-pulse: CODE_PULSE_ADDRESS 不是合法 0x 地址")
		} else if ethClient != nil {
			addr := common.HexToAddress(a)
			cp, errCP := contracts.NewCodePulse(ethClient.Eth(), addr)
			if errCP != nil {
				log.Printf("code-pulse: bind contract: %v", errCP)
			} else {
				codePulse = cp
				log.Printf("code-pulse: 合约绑定就绪 %s", addr.Hex())
			}
		} else {
			log.Printf("code-pulse: CODE_PULSE_ADDRESS 已配置但 ETH_RPC_URL 不可用，合约绑定跳过")
		}
	}

	h := &handlers.Handlers{
		DB:                db,
		Chain:             ethClient,
		Counter:           counter,
		TxKey:             txKey,
		Subgraph:          subClient,
		CodePulse:         codePulse,
		SubgraphCodePulse: cpSubClient,
	}
	r := router.New(h)

	switch {
	case ethClient == nil:
		log.Printf("bank indexer: 未启动（ETH_RPC_URL 未配置或无法连接节点，请检查 .env 与网络）")
	case strings.TrimSpace(cfg.BankContract) == "":
		log.Printf("bank indexer: 未启动（未设置 BANK_CONTRACT_ADDRESS）")
	default:
		if !common.IsHexAddress(cfg.BankContract) {
			log.Printf("bank indexer: 未启动（BANK_CONTRACT_ADDRESS 不是合法 0x 地址）")
			break
		}
		cid, errID := ethClient.ChainID(context.Background())
		if errID != nil {
			log.Printf("bank indexer: 未启动（读取 chainId 失败: %v）", errID)
			break
		}
		bankAddr := common.HexToAddress(cfg.BankContract)
		ix := indexer.NewBank(db, ethClient.Eth(), bankAddr, *cid, cfg.BankIndexerStartBlock)
		log.Printf("bank indexer: 已启动，合约 %s chain_id=%d（游标在首次成功同步链头后写入 chain_indexer_cursors）", bankAddr.Hex(), *cid)
		go ix.Run(context.Background())
	}

	log.Printf("listening on %s", cfg.ServerAddr)
	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
