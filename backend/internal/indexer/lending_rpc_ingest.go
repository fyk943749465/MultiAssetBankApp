package indexer

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func txLogConflict() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "tx_hash"},
			{Name: "log_index"},
		},
		DoNothing: true,
	}
}

func topicAddr(t common.Hash) common.Address {
	return common.BytesToAddress(t.Bytes()[12:])
}

func mustUint256Type() abi.Type {
	t, err := abi.NewType("uint256", "", nil)
	if err != nil {
		panic(err)
	}
	return t
}

func mustStringType() abi.Type {
	t, err := abi.NewType("string", "", nil)
	if err != nil {
		panic(err)
	}
	return t
}

var emodeTailArgs = abi.Arguments{
	{Type: mustUint256Type(), Name: "ltv"},
	{Type: mustUint256Type(), Name: "liquidationThreshold"},
	{Type: mustUint256Type(), Name: "liquidationBonus"},
	{Type: mustStringType(), Name: "label"},
}

func unpackEmodeCategoryData(data []byte) (ltv, liqT, liqB *big.Int, label string, err error) {
	if len(data) == 0 {
		return nil, nil, nil, "", fmt.Errorf("empty data")
	}
	out, err := emodeTailArgs.Unpack(data)
	if err != nil {
		return nil, nil, nil, "", err
	}
	ltv = out[0].(*big.Int)
	liqT = out[1].(*big.Int)
	liqB = out[2].(*big.Int)
	label = out[3].(string)
	return ltv, liqT, liqB, label, nil
}

func registerLendingContract(tx *gorm.DB, chainID int64, addr common.Address, kind, label string, kindByAddr map[common.Address]string) error {
	if addr == (common.Address{}) {
		return nil
	}
	lbl := label
	row := models.LendingContract{
		ChainID:      chainID,
		Address:      addrHex(addr),
		ContractKind: kind,
		DisplayLabel: &lbl,
	}
	conf := clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "address"},
		},
		DoNothing: true,
	}
	if err := tx.Clauses(conf).Create(&row).Error; err != nil {
		return err
	}
	kindByAddr[addr] = kind
	return nil
}

func upsertIRImmutableSnapshot(tx *gorm.DB, chainID int64, strategy common.Address, ou, bbr, s1, s2, rf *big.Int, bn int64, ts time.Time) error {
	if strategy == (common.Address{}) {
		return nil
	}
	row := models.LendingInterestRateStrategyImmutableParams{
		ChainID:               chainID,
		StrategyAddress:       addrHex(strategy),
		OptimalUtilizationRaw: ou.String(),
		BaseBorrowRateRaw:     bbr.String(),
		Slope1Raw:             s1.String(),
		Slope2Raw:             s2.String(),
		ReserveFactorRaw:      rf.String(),
		SourceBlockNumber:     bn,
		SourceBlockTime:       ts,
		CreatedAt:             time.Now().UTC(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "strategy_address"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"optimal_utilization_raw", "base_borrow_rate_raw", "slope1_raw", "slope2_raw", "reserve_factor_raw",
			"source_block_number", "source_block_time",
		}),
	}).Create(&row).Error
}

func ingestLendingLog(tx *gorm.DB, chainID int64, poolAddr common.Address, kindByAddr map[common.Address]string, lg types.Log, ts time.Time) error {
	if len(lg.Topics) == 0 {
		return nil
	}
	t0 := lg.Topics[0]
	txh := strings.ToLower(lg.TxHash.Hex())
	li := int(lg.Index)
	bn := int64(lg.BlockNumber)
	poolHex := addrHex(lg.Address)

	switch t0 {
	case topicSupply:
		if len(lg.Topics) < 3 || len(lg.Data) < 32 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		user := topicAddr(lg.Topics[2])
		amt := new(big.Int).SetBytes(lg.Data[:32])
		row := models.LendingSupply{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), UserAddress: addrHex(user), AmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicWithdraw:
		if len(lg.Topics) < 3 || len(lg.Data) < 32 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		user := topicAddr(lg.Topics[2])
		amt := new(big.Int).SetBytes(lg.Data[:32])
		row := models.LendingWithdrawal{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), UserAddress: addrHex(user), AmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicBorrow, topicRepay:
		if len(lg.Topics) < 3 || len(lg.Data) < 32 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		user := topicAddr(lg.Topics[2])
		amt := new(big.Int).SetBytes(lg.Data[:32])
		if t0 == topicBorrow {
			row := models.LendingBorrow{
				ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
				AssetAddress: addrHex(asset), UserAddress: addrHex(user), AmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
			}
			return tx.Clauses(txLogConflict()).Create(&row).Error
		}
		row := models.LendingRepay{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), UserAddress: addrHex(user), AmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicLiquidation:
		if len(lg.Topics) < 4 || len(lg.Data) < 32*4 {
			return nil
		}
		coll := topicAddr(lg.Topics[1])
		debt := topicAddr(lg.Topics[2])
		borrower := topicAddr(lg.Topics[3])
		liq := common.BytesToAddress(lg.Data[0:32])
		dc := new(big.Int).SetBytes(lg.Data[32:64])
		ctL := new(big.Int).SetBytes(lg.Data[64:96])
		ctP := new(big.Int).SetBytes(lg.Data[96:128])
		row := models.LendingLiquidation{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			CollateralAssetAddress: addrHex(coll), DebtAssetAddress: addrHex(debt), BorrowerAddress: addrHex(borrower),
			LiquidatorAddress: addrHex(liq), DebtCoveredRaw: dc.String(), CollateralToLiquidatorRaw: ctL.String(),
			CollateralProtocolFeeRaw: ctP.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicUserEModeSet:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		user := topicAddr(lg.Topics[1])
		cat := int(new(big.Int).SetBytes(lg.Data[:32]).Uint64())
		row := models.LendingUserEModeSet{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			UserAddress: addrHex(user), CategoryID: cat, CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicReserveCaps:
		if len(lg.Topics) < 2 || len(lg.Data) < 64 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		sCap := new(big.Int).SetBytes(lg.Data[:32])
		bCap := new(big.Int).SetBytes(lg.Data[32:64])
		row := models.LendingReserveCapsUpdated{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), SupplyCapRaw: sCap.String(), BorrowCapRaw: bCap.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicReserveLiqFee:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		fee := new(big.Int).SetBytes(lg.Data[:32])
		row := models.LendingReserveLiquidationProtocolFeeUpdated{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), FeeBpsRaw: fee.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicProtocolFee:
		if len(lg.Topics) < 2 {
			return nil
		}
		recipient := topicAddr(lg.Topics[1])
		row := models.LendingProtocolFeeRecipientUpdated{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			NewRecipientAddress: addrHex(recipient), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicReserveInit:
		if len(lg.Topics) < 4 || len(lg.Data) < 32*6 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		aTok := topicAddr(lg.Topics[2])
		dTok := topicAddr(lg.Topics[3])
		irStr := common.BytesToAddress(lg.Data[0:32])
		ltv := new(big.Int).SetBytes(lg.Data[32:64])
		liqT := new(big.Int).SetBytes(lg.Data[64:96])
		liqB := new(big.Int).SetBytes(lg.Data[96:128])
		sCap := new(big.Int).SetBytes(lg.Data[128:160])
		bCap := new(big.Int).SetBytes(lg.Data[160:192])
		row := models.LendingReserveInitialized{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), ATokenAddress: addrHex(aTok), DebtTokenAddress: addrHex(dTok),
			InterestRateStrategyAddress: addrHex(irStr), LtvRaw: ltv.String(), LiquidationThresholdRaw: liqT.String(),
			LiquidationBonusRaw: liqB.String(), SupplyCapRaw: sCap.String(), BorrowCapRaw: bCap.String(), CreatedAt: time.Now().UTC(),
		}
		if err := tx.Clauses(txLogConflict()).Create(&row).Error; err != nil {
			return err
		}
		_ = registerLendingContract(tx, chainID, aTok, "a_token", "AToken "+addrHex(asset), kindByAddr)
		_ = registerLendingContract(tx, chainID, dTok, "variable_debt_token", "VariableDebtToken "+addrHex(asset), kindByAddr)
		_ = registerLendingContract(tx, chainID, irStr, "interest_rate_strategy", "InterestRateStrategy "+addrHex(asset), kindByAddr)
		return nil

	case topicEModeCat:
		if len(lg.Topics) < 2 {
			return nil
		}
		catID := int(new(big.Int).SetBytes(lg.Topics[1].Bytes()).Uint64())
		ltv, liqT, liqB, label, err := unpackEmodeCategoryData(lg.Data)
		if err != nil {
			label = ""
			if len(lg.Data) >= 96 {
				ltv = new(big.Int).SetBytes(lg.Data[0:32])
				liqT = new(big.Int).SetBytes(lg.Data[32:64])
				liqB = new(big.Int).SetBytes(lg.Data[64:96])
			} else {
				ltv, liqT, liqB = big.NewInt(0), big.NewInt(0), big.NewInt(0)
			}
		}
		row := models.LendingEmodeCategoryConfigured{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			CategoryID: catID, LtvRaw: ltv.String(), LiquidationThresholdRaw: liqT.String(), LiquidationBonusRaw: liqB.String(),
			Label: label, CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicPaused, topicUnpaused:
		if len(lg.Topics) < 2 {
			return nil
		}
		acct := topicAddr(lg.Topics[1])
		if t0 == topicPaused {
			row := models.LendingPaused{
				ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
				AccountAddress: addrHex(acct), CreatedAt: time.Now().UTC(),
			}
			return tx.Clauses(txLogConflict()).Create(&row).Error
		}
		row := models.LendingUnpaused{
			ChainID: chainID, PoolAddress: poolHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AccountAddress: addrHex(acct), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicOwnership:
		if len(lg.Topics) < 3 {
			return nil
		}
		prev := topicAddr(lg.Topics[1])
		next := topicAddr(lg.Topics[2])
		row := models.LendingOwnershipTransferred{
			ChainID: chainID, EmitterAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			PreviousOwnerAddress: addrHex(prev), NewOwnerAddress: addrHex(next), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicPoolSet:
		if len(lg.Topics) < 2 {
			return nil
		}
		p := topicAddr(lg.Topics[1])
		row := models.LendingHybridPoolSet{
			ChainID: chainID, OracleAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			PoolAddress: addrHex(p), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicStreamCfg:
		if len(lg.Topics) < 3 || len(lg.Data) < 32 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		feedID := lg.Topics[2]
		dec := int(new(big.Int).SetBytes(lg.Data[:32]).Uint64())
		row := models.LendingHybridStreamConfigUpdated{
			ChainID: chainID, OracleAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), StreamFeedIDHex: strings.ToLower(feedID.Hex()), PriceDecimals: dec, CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicStreamFB:
		if len(lg.Topics) < 2 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		row := models.LendingHybridStreamPriceFallbackToFeed{
			ChainID: chainID, OracleAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicAuthOracle:
		if len(lg.Topics) < 2 {
			return nil
		}
		oracle := topicAddr(lg.Topics[1])
		row := models.LendingReportsAuthorizedOracleSet{
			ChainID: chainID, VerifierAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			OracleAddress: addrHex(oracle), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicTokenSweep:
		if len(lg.Topics) < 3 || len(lg.Data) < 32 {
			return nil
		}
		token := topicAddr(lg.Topics[1])
		to := topicAddr(lg.Topics[2])
		amt := new(big.Int).SetBytes(lg.Data[:32])
		row := models.LendingReportsTokenSwept{
			ChainID: chainID, VerifierAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			TokenAddress: addrHex(token), ToAddress: addrHex(to), AmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicNativeSweep:
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		to := topicAddr(lg.Topics[1])
		amt := new(big.Int).SetBytes(lg.Data[:32])
		row := models.LendingReportsNativeSwept{
			ChainID: chainID, VerifierAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			ToAddress: addrHex(to), AmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicFeedSet:
		if len(lg.Topics) < 3 || len(lg.Data) < 32 {
			return nil
		}
		asset := topicAddr(lg.Topics[1])
		feed := topicAddr(lg.Topics[2])
		stale := new(big.Int).SetBytes(lg.Data[:32])
		row := models.LendingChainlinkFeedSet{
			ChainID: chainID, OracleAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			AssetAddress: addrHex(asset), FeedAddress: addrHex(feed), StalePeriodRaw: stale.String(), CreatedAt: time.Now().UTC(),
		}
		return tx.Clauses(txLogConflict()).Create(&row).Error

	case topicStrategyCreated:
		if len(lg.Topics) < 3 || len(lg.Data) < 32*5 {
			return nil
		}
		strategy := topicAddr(lg.Topics[1])
		idx := new(big.Int).SetBytes(lg.Topics[2].Bytes())
		ou := new(big.Int).SetBytes(lg.Data[0:32])
		bbr := new(big.Int).SetBytes(lg.Data[32:64])
		s1 := new(big.Int).SetBytes(lg.Data[64:96])
		s2 := new(big.Int).SetBytes(lg.Data[96:128])
		rf := new(big.Int).SetBytes(lg.Data[128:160])
		row := models.LendingInterestRateStrategyCreated{
			ChainID: chainID, FactoryAddress: addrHex(lg.Address), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			StrategyAddress: addrHex(strategy), StrategyIndexRaw: idx.String(), OptimalUtilizationRaw: ou.String(),
			BaseBorrowRateRaw: bbr.String(), Slope1Raw: s1.String(), Slope2Raw: s2.String(), ReserveFactorRaw: rf.String(), CreatedAt: time.Now().UTC(),
		}
		if err := tx.Clauses(txLogConflict()).Create(&row).Error; err != nil {
			return err
		}
		return upsertIRImmutableSnapshot(tx, chainID, strategy, ou, bbr, s1, s2, rf, bn, ts)

	case topicIRDeployed:
		if len(lg.Data) < 32*5 {
			return nil
		}
		strategy := lg.Address
		ou := new(big.Int).SetBytes(lg.Data[0:32])
		bbr := new(big.Int).SetBytes(lg.Data[32:64])
		s1 := new(big.Int).SetBytes(lg.Data[64:96])
		s2 := new(big.Int).SetBytes(lg.Data[96:128])
		rf := new(big.Int).SetBytes(lg.Data[128:160])
		row := models.LendingInterestRateStrategyDeployed{
			ChainID: chainID, StrategyAddress: addrHex(strategy), TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
			OptimalUtilizationRaw: ou.String(), BaseBorrowRateRaw: bbr.String(), Slope1Raw: s1.String(), Slope2Raw: s2.String(),
			ReserveFactorRaw: rf.String(), CreatedAt: time.Now().UTC(),
		}
		if err := tx.Clauses(txLogConflict()).Create(&row).Error; err != nil {
			return err
		}
		return upsertIRImmutableSnapshot(tx, chainID, strategy, ou, bbr, s1, s2, rf, bn, ts)

	case topicMint, topicBurn:
		k := kindByAddr[lg.Address]
		tokHex := addrHex(lg.Address)
		if t0 == topicMint {
			if len(lg.Topics) < 2 || len(lg.Data) < 32 {
				return nil
			}
			to := topicAddr(lg.Topics[1])
			amt := new(big.Int).SetBytes(lg.Data[:32])
			switch k {
			case "a_token":
				return tx.Clauses(txLogConflict()).Create(&models.LendingATokenMint{
					ChainID: chainID, TokenAddress: tokHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
					ToAddress: addrHex(to), ScaledAmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
				}).Error
			case "variable_debt_token":
				return tx.Clauses(txLogConflict()).Create(&models.LendingVariableDebtTokenMint{
					ChainID: chainID, TokenAddress: tokHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
					ToAddress: addrHex(to), ScaledAmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
				}).Error
			}
			return nil
		}
		if len(lg.Topics) < 2 || len(lg.Data) < 32 {
			return nil
		}
		from := topicAddr(lg.Topics[1])
		amt := new(big.Int).SetBytes(lg.Data[:32])
		switch k {
		case "a_token":
			return tx.Clauses(txLogConflict()).Create(&models.LendingATokenBurn{
				ChainID: chainID, TokenAddress: tokHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
				FromAddress: addrHex(from), ScaledAmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
			}).Error
		case "variable_debt_token":
			return tx.Clauses(txLogConflict()).Create(&models.LendingVariableDebtTokenBurn{
				ChainID: chainID, TokenAddress: tokHex, TxHash: txh, LogIndex: li, BlockNumber: bn, BlockTime: ts,
				FromAddress: addrHex(from), ScaledAmountRaw: amt.String(), CreatedAt: time.Now().UTC(),
			}).Error
		}
		return nil

	default:
		return nil
	}
}
