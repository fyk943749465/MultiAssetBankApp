import {
  OwnershipTransferred as OwnershipTransferredEvent,
  StreamConfigUpdated as StreamConfigUpdatedEvent,
  StreamPriceFallbackToFeed as StreamPriceFallbackToFeedEvent,
} from "../generated/HybridPriceOracle/HybridPriceOracle"
import {
  OwnershipTransferred,
  StreamConfigUpdated,
  StreamPriceFallbackToFeed,
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

export function handleStreamConfigUpdated(
  event: StreamConfigUpdatedEvent,
): void {
  let entity = new StreamConfigUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.asset = event.params.asset
  entity.streamFeedId = event.params.streamFeedId
  entity.priceDecimals = event.params.priceDecimals

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleStreamPriceFallbackToFeed(
  event: StreamPriceFallbackToFeedEvent,
): void {
  let entity = new StreamPriceFallbackToFeed(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.asset = event.params.asset

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
