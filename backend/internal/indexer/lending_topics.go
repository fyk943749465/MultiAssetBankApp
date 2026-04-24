package indexer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// 事件 topic0（Solidity canonical 签名），与 subgraph/lending 合约 ABI 一致。

var (
	topicSupply        = crypto.Keccak256Hash([]byte("Supply(address,address,uint256)"))
	topicWithdraw      = crypto.Keccak256Hash([]byte("Withdraw(address,address,uint256)"))
	topicBorrow        = crypto.Keccak256Hash([]byte("Borrow(address,address,uint256)"))
	topicRepay         = crypto.Keccak256Hash([]byte("Repay(address,address,uint256)"))
	topicLiquidation   = crypto.Keccak256Hash([]byte("LiquidationCall(address,address,address,address,uint256,uint256,uint256)"))
	topicUserEModeSet  = crypto.Keccak256Hash([]byte("UserEModeSet(address,uint8)"))
	topicReserveCaps   = crypto.Keccak256Hash([]byte("ReserveCapsUpdated(address,uint256,uint256)"))
	topicReserveLiqFee = crypto.Keccak256Hash([]byte("ReserveLiquidationProtocolFeeUpdated(address,uint256)"))
	topicProtocolFee   = crypto.Keccak256Hash([]byte("ProtocolFeeRecipientUpdated(address)"))
	topicReserveInit   = crypto.Keccak256Hash([]byte("ReserveInitialized(address,address,address,address,uint256,uint256,uint256,uint256,uint256)"))
	topicEModeCat = crypto.Keccak256Hash([]byte("EModeCategoryConfigured(uint8,uint256,uint256,uint256,string)"))
	// Paused / Unpaused 与 OpenZeppelin Pausable 一致，topic0 与 nft_platform_rpc 中 topicPaused、topicUnpaused 相同，此处复用。

	topicOwnership = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))
	topicPoolSet   = crypto.Keccak256Hash([]byte("PoolSet(address)"))
	topicStreamCfg = crypto.Keccak256Hash([]byte("StreamConfigUpdated(address,bytes32,uint8)"))
	topicStreamFB  = crypto.Keccak256Hash([]byte("StreamPriceFallbackToFeed(address)"))

	topicAuthOracle = crypto.Keccak256Hash([]byte("AuthorizedOracleSet(address)"))
	topicTokenSweep = crypto.Keccak256Hash([]byte("TokenSwept(address,address,uint256)"))
	topicNativeSweep = crypto.Keccak256Hash([]byte("NativeSwept(address,uint256)"))

	topicFeedSet = crypto.Keccak256Hash([]byte("FeedSet(address,address,uint256)"))

	topicStrategyCreated = crypto.Keccak256Hash([]byte("StrategyCreated(address,uint256,uint256,uint256,uint256,uint256,uint256)"))
	topicIRDeployed        = crypto.Keccak256Hash([]byte("InterestRateStrategyDeployed(uint256,uint256,uint256,uint256,uint256)"))

	topicMint = crypto.Keccak256Hash([]byte("Mint(address,uint256)"))
	topicBurn = crypto.Keccak256Hash([]byte("Burn(address,uint256)"))
)

func allLendingTopics() []common.Hash {
	return []common.Hash{
		topicSupply, topicWithdraw, topicBorrow, topicRepay, topicLiquidation,
		topicUserEModeSet, topicReserveCaps, topicReserveLiqFee, topicProtocolFee,
		topicReserveInit, topicEModeCat, topicPaused, topicUnpaused,
		topicOwnership, topicPoolSet, topicStreamCfg, topicStreamFB,
		topicAuthOracle, topicTokenSweep, topicNativeSweep,
		topicFeedSet, topicStrategyCreated, topicIRDeployed,
		topicMint, topicBurn,
	}
}
