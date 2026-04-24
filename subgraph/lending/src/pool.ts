import { Address } from "@graphprotocol/graph-ts"
import {
  Borrow as BorrowEvent,
  EModeCategoryConfigured as EModeCategoryConfiguredEvent,
  LiquidationCall as LiquidationCallEvent,
  Paused as PausedEvent,
  ProtocolFeeRecipientUpdated as ProtocolFeeRecipientUpdatedEvent,
  Repay as RepayEvent,
  ReserveCapsUpdated as ReserveCapsUpdatedEvent,
  ReserveInitialized as ReserveInitializedEvent,
  ReserveLiquidationProtocolFeeUpdated as ReserveLiquidationProtocolFeeUpdatedEvent,
  Supply as SupplyEvent,
  Unpaused as UnpausedEvent,
  UserEModeSet as UserEModeSetEvent,
  Withdraw as WithdrawEvent
} from "../generated/Pool/Pool"
import { AToken, VariableDebtToken } from "../generated/templates"
import {
  Borrow,
  EModeCategoryConfigured,
  LiquidationCall,
  Paused,
  ProtocolFeeRecipientUpdated,
  Repay,
  ReserveCapsUpdated,
  ReserveInitialized,
  ReserveLiquidationProtocolFeeUpdated,
  Supply,
  Unpaused,
  UserEModeSet,
  Withdraw
} from "../generated/schema"

export function handleBorrow(event: BorrowEvent): void {
  let entity = new Borrow(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.asset = event.params.asset
  entity.user = event.params.user
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleLiquidationCall(event: LiquidationCallEvent): void {
  let entity = new LiquidationCall(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.collateralAsset = event.params.collateralAsset
  entity.debtAsset = event.params.debtAsset
  entity.borrower = event.params.borrower
  entity.liquidator = event.params.liquidator
  entity.debtCovered = event.params.debtCovered
  entity.collateralToLiquidator = event.params.collateralToLiquidator
  entity.collateralProtocolFee = event.params.collateralProtocolFee

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handlePaused(event: PausedEvent): void {
  let entity = new Paused(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.account = event.params.account

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleProtocolFeeRecipientUpdated(
  event: ProtocolFeeRecipientUpdatedEvent
): void {
  let entity = new ProtocolFeeRecipientUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.newRecipient = event.params.newRecipient

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleRepay(event: RepayEvent): void {
  let entity = new Repay(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.asset = event.params.asset
  entity.user = event.params.user
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleReserveCapsUpdated(event: ReserveCapsUpdatedEvent): void {
  let entity = new ReserveCapsUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.asset = event.params.asset
  entity.supplyCap = event.params.supplyCap
  entity.borrowCap = event.params.borrowCap

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleReserveInitialized(event: ReserveInitializedEvent): void {
  let entity = new ReserveInitialized(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.asset = event.params.asset
  entity.aToken = event.params.aToken
  entity.debtToken = event.params.debtToken
  entity.interestRateStrategy = event.params.interestRateStrategy
  entity.ltv = event.params.ltv
  entity.liquidationThreshold = event.params.liquidationThreshold
  entity.liquidationBonus = event.params.liquidationBonus
  entity.supplyCap = event.params.supplyCap
  entity.borrowCap = event.params.borrowCap
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()

  if (!event.params.aToken.equals(Address.zero())) {
    AToken.create(event.params.aToken)
  }
  if (!event.params.debtToken.equals(Address.zero())) {
    VariableDebtToken.create(event.params.debtToken)
  }
}

export function handleEModeCategoryConfigured(
  event: EModeCategoryConfiguredEvent
): void {
  let entity = new EModeCategoryConfigured(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.categoryId = event.params.categoryId
  entity.ltv = event.params.ltv
  entity.liquidationThreshold = event.params.liquidationThreshold
  entity.liquidationBonus = event.params.liquidationBonus
  entity.label = event.params.label
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()
}

export function handleReserveLiquidationProtocolFeeUpdated(
  event: ReserveLiquidationProtocolFeeUpdatedEvent
): void {
  let entity = new ReserveLiquidationProtocolFeeUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.asset = event.params.asset
  entity.feeBps = event.params.feeBps

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleSupply(event: SupplyEvent): void {
  let entity = new Supply(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.asset = event.params.asset
  entity.user = event.params.user
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleUnpaused(event: UnpausedEvent): void {
  let entity = new Unpaused(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.account = event.params.account

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleUserEModeSet(event: UserEModeSetEvent): void {
  let entity = new UserEModeSet(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.user = event.params.user
  entity.categoryId = event.params.categoryId

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleWithdraw(event: WithdrawEvent): void {
  let entity = new Withdraw(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.asset = event.params.asset
  entity.user = event.params.user
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
