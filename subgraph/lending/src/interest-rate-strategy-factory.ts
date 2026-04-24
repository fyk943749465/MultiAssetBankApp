import {
  OwnershipTransferred as OwnershipTransferredEvent,
  StrategyCreated as StrategyCreatedEvent,
} from "../generated/InterestRateStrategyFactory/InterestRateStrategyFactory"
import {
  InterestRateStrategyCreated,
  OwnershipTransferred,
} from "../generated/schema"

export function handleOwnershipTransferred(
  event: OwnershipTransferredEvent,
): void {
  let entity = new OwnershipTransferred(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.previousOwner = event.params.previousOwner
  entity.newOwner = event.params.newOwner
  entity.emitter = event.address

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleStrategyCreated(event: StrategyCreatedEvent): void {
  let entity = new InterestRateStrategyCreated(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.factory = event.address
  entity.strategy = event.params.strategy
  entity.strategyIndex = event.params.id
  entity.optimalUtilization = event.params.optimalUtilization
  entity.baseBorrowRate = event.params.baseBorrowRate
  entity.slope1 = event.params.slope1
  entity.slope2 = event.params.slope2
  entity.reserveFactor = event.params.reserveFactor

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
