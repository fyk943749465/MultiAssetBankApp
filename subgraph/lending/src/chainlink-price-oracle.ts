import {
  FeedSet as FeedSetEvent,
  OwnershipTransferred as OwnershipTransferredEvent,
} from "../generated/ChainlinkPriceOracle/ChainlinkPriceOracle"
import { ChainlinkFeedSet, OwnershipTransferred } from "../generated/schema"

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

export function handleFeedSet(event: FeedSetEvent): void {
  let entity = new ChainlinkFeedSet(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.asset = event.params.asset
  entity.feed = event.params.feed
  entity.stalePeriod = event.params.stalePeriod
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()
}
