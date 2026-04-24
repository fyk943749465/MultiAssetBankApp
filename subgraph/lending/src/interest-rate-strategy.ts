import { ethereum, dataSource } from "@graphprotocol/graph-ts"
import {
  InterestRateStrategy,
  InterestRateStrategyDeployed as InterestRateStrategyDeployedEvent,
} from "../generated/InterestRateStrategy/InterestRateStrategy"
import {
  InterestRateStrategyDeployed,
  InterestRateStrategyImmutableParams,
} from "../generated/schema"

export function handleInterestRateStrategyInit(block: ethereum.Block): void {
  let c = InterestRateStrategy.bind(dataSource.address())

  let ou = c.try_optimalUtilization()
  let bb = c.try_baseBorrowRate()
  let s1 = c.try_slope1()
  let s2 = c.try_slope2()
  let rf = c.try_reserveFactor()
  if (ou.reverted || bb.reverted || s1.reverted || s2.reverted || rf.reverted) {
    return
  }

  let entity = new InterestRateStrategyImmutableParams(dataSource.address())
  entity.optimalUtilization = ou.value
  entity.baseBorrowRate = bb.value
  entity.slope1 = s1.value
  entity.slope2 = s2.value
  entity.reserveFactor = rf.value
  entity.blockNumber = block.number
  entity.blockTimestamp = block.timestamp
  entity.save()
}

export function handleInterestRateStrategyDeployed(
  event: InterestRateStrategyDeployedEvent,
): void {
  let entity = new InterestRateStrategyDeployed(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.strategy = event.address
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
