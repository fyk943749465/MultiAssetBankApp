import { ethereum, dataSource } from "@graphprotocol/graph-ts"
import { InterestRateStrategy } from "../generated/InterestRateStrategy/InterestRateStrategy"
import { InterestRateStrategyImmutableParams } from "../generated/schema"

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
