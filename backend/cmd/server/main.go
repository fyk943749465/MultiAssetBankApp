package main

import (
	"context"
	"crypto/ecdsa"
	"log"
	"strings"

	"go-chain/backend/internal/chain"
	"go-chain/backend/internal/config"
	"go-chain/backend/internal/contracts"
	"go-chain/backend/internal/database"
	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/indexer"
	"go-chain/backend/internal/router"

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
		log.Printf("chain: optional RPC not available: %v", err)
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
			log.Printf("ETH_PRIVATE_KEY invalid: %v", errPK)
		} else {
			txKey = pk
		}
	}

	h := &handlers.Handlers{DB: db, Chain: ethClient, Counter: counter, TxKey: txKey}
	r := router.New(h)

	switch {
	case ethClient == nil:
		log.Print("bank indexer: 未启动（ETH_RPC_URL 未配置或无法连接节点，请检查 .env 与网络）")
	case strings.TrimSpace(cfg.BankContract) == "":
		log.Print("bank indexer: 未启动（未设置 BANK_CONTRACT_ADDRESS）")
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
