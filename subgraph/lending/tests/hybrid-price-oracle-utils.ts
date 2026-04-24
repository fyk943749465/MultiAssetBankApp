import { newMockEvent } from "matchstick-as"
import { ethereum, Address, Bytes } from "@graphprotocol/graph-ts"
import {
  OwnershipTransferred,
  StreamConfigUpdated,
  StreamPriceFallbackToFeed
} from "../generated/HybridPriceOracle/HybridPriceOracle"

export function createOwnershipTransferredEvent(
  previousOwner: Address,
  newOwner: Address
): OwnershipTransferred {
  let ownershipTransferredEvent =
    changetype<OwnershipTransferred>(newMockEvent())

  ownershipTransferredEvent.parameters = new Array()

  ownershipTransferredEvent.parameters.push(
    new ethereum.EventParam(
      "previousOwner",
      ethereum.Value.fromAddress(previousOwner)
    )
  )
  ownershipTransferredEvent.parameters.push(
    new ethereum.EventParam("newOwner", ethereum.Value.fromAddress(newOwner))
  )

  return ownershipTransferredEvent
}

export function createStreamConfigUpdatedEvent(
  asset: Address,
  streamFeedId: Bytes,
  priceDecimals: i32
): StreamConfigUpdated {
  let streamConfigUpdatedEvent = changetype<StreamConfigUpdated>(newMockEvent())

  streamConfigUpdatedEvent.parameters = new Array()

  streamConfigUpdatedEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )
  streamConfigUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "streamFeedId",
      ethereum.Value.fromFixedBytes(streamFeedId)
    )
  )
  streamConfigUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "priceDecimals",
      ethereum.Value.fromUnsignedBigInt(BigInt.fromI32(priceDecimals))
    )
  )

  return streamConfigUpdatedEvent
}

export function createStreamPriceFallbackToFeedEvent(
  asset: Address
): StreamPriceFallbackToFeed {
  let streamPriceFallbackToFeedEvent =
    changetype<StreamPriceFallbackToFeed>(newMockEvent())

  streamPriceFallbackToFeedEvent.parameters = new Array()

  streamPriceFallbackToFeedEvent.parameters.push(
    new ethereum.EventParam("asset", ethereum.Value.fromAddress(asset))
  )

  return streamPriceFallbackToFeedEvent
}
