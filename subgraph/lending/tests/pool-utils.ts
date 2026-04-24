import { newMockEvent } from "matchstick-as"
import { ethereum, Address, BigInt } from "@graphprotocol/graph-ts"
import {
  Borrow,
  LiquidationCall,
  Paused,
  ProtocolFeeRecipientUpdated,
  Repay,
  ReserveCapsUpdated,
  ReserveLiquidationProtocolFeeUpdated,
  Supply,
  Unpaused,
  UserEModeSet,
  Withdraw
} from "../generated/Pool/Pool"

export function createBorrowEvent(
  asset: Address,
  user: Address,
  amount: BigInt
): Borrow {
  let borrowEvent = changetype<Borrow>(newMockEvent())

  borrowEvent.parameters = new Array()

  borrowEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )
  borrowEvent.parameters.push(
    new ethereum.EventParam("user", ethereum.Value.fromAddress(user))
  )
  borrowEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return borrowEvent
}

export function createLiquidationCallEvent(
  collateralAsset: Address,
  debtAsset: Address,
  borrower: Address,
  liquidator: Address,
  debtCovered: BigInt,
  collateralToLiquidator: BigInt,
  collateralProtocolFee: BigInt
): LiquidationCall {
  let liquidationCallEvent = changetype<LiquidationCall>(newMockEvent())

  liquidationCallEvent.parameters = new Array()

  liquidationCallEvent.parameters.push(
    new ethereum.EventParam(
      "collateralAsset",
      ethereum.Value.fromAddress(collateralAsset)
    )
  )
  liquidationCallEvent.parameters.push(
    new ethereum.EventParam("debtAsset", ethereum.Value.fromAddress(debtAsset))
  )
  liquidationCallEvent.parameters.push(
    new ethereum.EventParam("borrower", ethereum.Value.fromAddress(borrower))
  )
  liquidationCallEvent.parameters.push(
    new ethereum.EventParam(
      "liquidator",
      ethereum.Value.fromAddress(liquidator)
    )
  )
  liquidationCallEvent.parameters.push(
    new ethereum.EventParam(
      "debtCovered",
      ethereum.Value.fromUnsignedBigInt(debtCovered)
    )
  )
  liquidationCallEvent.parameters.push(
    new ethereum.EventParam(
      "collateralToLiquidator",
      ethereum.Value.fromUnsignedBigInt(collateralToLiquidator)
    )
  )
  liquidationCallEvent.parameters.push(
    new ethereum.EventParam(
      "collateralProtocolFee",
      ethereum.Value.fromUnsignedBigInt(collateralProtocolFee)
    )
  )

  return liquidationCallEvent
}

export function createPausedEvent(account: Address): Paused {
  let pausedEvent = changetype<Paused>(newMockEvent())

  pausedEvent.parameters = new Array()

  pausedEvent.parameters.push(
    new ethereum.EventParam("account", ethereum.Value.fromAddress(account))
  )

  return pausedEvent
}

export function createProtocolFeeRecipientUpdatedEvent(
  newRecipient: Address
): ProtocolFeeRecipientUpdated {
  let protocolFeeRecipientUpdatedEvent =
    changetype<ProtocolFeeRecipientUpdated>(newMockEvent())

  protocolFeeRecipientUpdatedEvent.parameters = new Array()

  protocolFeeRecipientUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "newRecipient",
      ethereum.Value.fromAddress(newRecipient)
    )
  )

  return protocolFeeRecipientUpdatedEvent
}

export function createRepayEvent(
  asset: Address,
  user: Address,
  amount: BigInt
): Repay {
  let repayEvent = changetype<Repay>(newMockEvent())

  repayEvent.parameters = new Array()

  repayEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )
  repayEvent.parameters.push(
    new ethereum.EventParam("user", ethereum.Value.fromAddress(user))
  )
  repayEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return repayEvent
}

export function createReserveCapsUpdatedEvent(
  asset: Address,
  supplyCap: BigInt,
  borrowCap: BigInt
): ReserveCapsUpdated {
  let reserveCapsUpdatedEvent = changetype<ReserveCapsUpdated>(newMockEvent())

  reserveCapsUpdatedEvent.parameters = new Array()

  reserveCapsUpdatedEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )
  reserveCapsUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "supplyCap",
      ethereum.Value.fromUnsignedBigInt(supplyCap)
    )
  )
  reserveCapsUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "borrowCap",
      ethereum.Value.fromUnsignedBigInt(borrowCap)
    )
  )

  return reserveCapsUpdatedEvent
}

export function createReserveLiquidationProtocolFeeUpdatedEvent(
  asset: Address,
  feeBps: BigInt
): ReserveLiquidationProtocolFeeUpdated {
  let reserveLiquidationProtocolFeeUpdatedEvent =
    changetype<ReserveLiquidationProtocolFeeUpdated>(newMockEvent())

  reserveLiquidationProtocolFeeUpdatedEvent.parameters = new Array()

  reserveLiquidationProtocolFeeUpdatedEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )
  reserveLiquidationProtocolFeeUpdatedEvent.parameters.push(
    new ethereum.EventParam("feeBps", ethereum.Value.fromUnsignedBigInt(feeBps))
  )

  return reserveLiquidationProtocolFeeUpdatedEvent
}

export function createSupplyEvent(
  asset: Address,
  user: Address,
  amount: BigInt
): Supply {
  let supplyEvent = changetype<Supply>(newMockEvent())

  supplyEvent.parameters = new Array()

  supplyEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )
  supplyEvent.parameters.push(
    new ethereum.EventParam("user", ethereum.Value.fromAddress(user))
  )
  supplyEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return supplyEvent
}

export function createUnpausedEvent(account: Address): Unpaused {
  let unpausedEvent = changetype<Unpaused>(newMockEvent())

  unpausedEvent.parameters = new Array()

  unpausedEvent.parameters.push(
    new ethereum.EventParam("account", ethereum.Value.fromAddress(account))
  )

  return unpausedEvent
}

export function createUserEModeSetEvent(
  user: Address,
  categoryId: i32
): UserEModeSet {
  let userEModeSetEvent = changetype<UserEModeSet>(newMockEvent())

  userEModeSetEvent.parameters = new Array()

  userEModeSetEvent.parameters.push(
    new ethereum.EventParam("user", ethereum.Value.fromAddress(user))
  )
  userEModeSetEvent.parameters.push(
    new ethereum.EventParam(
      "categoryId",
      ethereum.Value.fromUnsignedBigInt(BigInt.fromI32(categoryId))
    )
  )

  return userEModeSetEvent
}

export function createWithdrawEvent(
  asset: Address,
  user: Address,
  amount: BigInt
): Withdraw {
  let withdrawEvent = changetype<Withdraw>(newMockEvent())

  withdrawEvent.parameters = new Array()

  withdrawEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )
  withdrawEvent.parameters.push(
    new ethereum.EventParam("user", ethereum.Value.fromAddress(user))
  )
  withdrawEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return withdrawEvent
}
