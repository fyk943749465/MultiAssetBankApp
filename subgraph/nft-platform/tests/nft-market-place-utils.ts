import { newMockEvent } from "matchstick-as"
import { ethereum, Address, BigInt } from "@graphprotocol/graph-ts"
import {
  ItemListed,
  ItemSold,
  ListingCanceled,
  ListingPriceUpdated,
  MaxRoyaltyBpsUpdated,
  OwnershipTransferred,
  Paused,
  PlatformFeeUpdated,
  PlatformFeesWithdrawn,
  Unpaused,
  UntrackedEthWithdrawn
} from "../generated/NFTMarketPlace/NFTMarketPlace"

export function createItemListedEvent(
  collection: Address,
  tokenId: BigInt,
  seller: Address,
  price: BigInt
): ItemListed {
  let itemListedEvent = changetype<ItemListed>(newMockEvent())

  itemListedEvent.parameters = new Array()

  itemListedEvent.parameters.push(
    new ethereum.EventParam(
      "collection",
      ethereum.Value.fromAddress(collection)
    )
  )
  itemListedEvent.parameters.push(
    new ethereum.EventParam(
      "tokenId",
      ethereum.Value.fromUnsignedBigInt(tokenId)
    )
  )
  itemListedEvent.parameters.push(
    new ethereum.EventParam("seller", ethereum.Value.fromAddress(seller))
  )
  itemListedEvent.parameters.push(
    new ethereum.EventParam("price", ethereum.Value.fromUnsignedBigInt(price))
  )

  return itemListedEvent
}

export function createItemSoldEvent(
  collection: Address,
  tokenId: BigInt,
  seller: Address,
  buyer: Address,
  price: BigInt,
  platformFee: BigInt,
  royaltyAmount: BigInt,
  feeBpsSnapshot: BigInt
): ItemSold {
  let itemSoldEvent = changetype<ItemSold>(newMockEvent())

  itemSoldEvent.parameters = new Array()

  itemSoldEvent.parameters.push(
    new ethereum.EventParam(
      "collection",
      ethereum.Value.fromAddress(collection)
    )
  )
  itemSoldEvent.parameters.push(
    new ethereum.EventParam(
      "tokenId",
      ethereum.Value.fromUnsignedBigInt(tokenId)
    )
  )
  itemSoldEvent.parameters.push(
    new ethereum.EventParam("seller", ethereum.Value.fromAddress(seller))
  )
  itemSoldEvent.parameters.push(
    new ethereum.EventParam("buyer", ethereum.Value.fromAddress(buyer))
  )
  itemSoldEvent.parameters.push(
    new ethereum.EventParam("price", ethereum.Value.fromUnsignedBigInt(price))
  )
  itemSoldEvent.parameters.push(
    new ethereum.EventParam(
      "platformFee",
      ethereum.Value.fromUnsignedBigInt(platformFee)
    )
  )
  itemSoldEvent.parameters.push(
    new ethereum.EventParam(
      "royaltyAmount",
      ethereum.Value.fromUnsignedBigInt(royaltyAmount)
    )
  )
  itemSoldEvent.parameters.push(
    new ethereum.EventParam(
      "feeBpsSnapshot",
      ethereum.Value.fromUnsignedBigInt(feeBpsSnapshot)
    )
  )

  return itemSoldEvent
}

export function createListingCanceledEvent(
  collection: Address,
  tokenId: BigInt,
  seller: Address
): ListingCanceled {
  let listingCanceledEvent = changetype<ListingCanceled>(newMockEvent())

  listingCanceledEvent.parameters = new Array()

  listingCanceledEvent.parameters.push(
    new ethereum.EventParam(
      "collection",
      ethereum.Value.fromAddress(collection)
    )
  )
  listingCanceledEvent.parameters.push(
    new ethereum.EventParam(
      "tokenId",
      ethereum.Value.fromUnsignedBigInt(tokenId)
    )
  )
  listingCanceledEvent.parameters.push(
    new ethereum.EventParam("seller", ethereum.Value.fromAddress(seller))
  )

  return listingCanceledEvent
}

export function createListingPriceUpdatedEvent(
  collection: Address,
  tokenId: BigInt,
  seller: Address,
  oldPrice: BigInt,
  newPrice: BigInt
): ListingPriceUpdated {
  let listingPriceUpdatedEvent = changetype<ListingPriceUpdated>(newMockEvent())

  listingPriceUpdatedEvent.parameters = new Array()

  listingPriceUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "collection",
      ethereum.Value.fromAddress(collection)
    )
  )
  listingPriceUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "tokenId",
      ethereum.Value.fromUnsignedBigInt(tokenId)
    )
  )
  listingPriceUpdatedEvent.parameters.push(
    new ethereum.EventParam("seller", ethereum.Value.fromAddress(seller))
  )
  listingPriceUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "oldPrice",
      ethereum.Value.fromUnsignedBigInt(oldPrice)
    )
  )
  listingPriceUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "newPrice",
      ethereum.Value.fromUnsignedBigInt(newPrice)
    )
  )

  return listingPriceUpdatedEvent
}

export function createMaxRoyaltyBpsUpdatedEvent(
  newBps: BigInt
): MaxRoyaltyBpsUpdated {
  let maxRoyaltyBpsUpdatedEvent =
    changetype<MaxRoyaltyBpsUpdated>(newMockEvent())

  maxRoyaltyBpsUpdatedEvent.parameters = new Array()

  maxRoyaltyBpsUpdatedEvent.parameters.push(
    new ethereum.EventParam("newBps", ethereum.Value.fromUnsignedBigInt(newBps))
  )

  return maxRoyaltyBpsUpdatedEvent
}

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

export function createPausedEvent(account: Address): Paused {
  let pausedEvent = changetype<Paused>(newMockEvent())

  pausedEvent.parameters = new Array()

  pausedEvent.parameters.push(
    new ethereum.EventParam("account", ethereum.Value.fromAddress(account))
  )

  return pausedEvent
}

export function createPlatformFeeUpdatedEvent(
  oldBps: BigInt,
  newBps: BigInt
): PlatformFeeUpdated {
  let platformFeeUpdatedEvent = changetype<PlatformFeeUpdated>(newMockEvent())

  platformFeeUpdatedEvent.parameters = new Array()

  platformFeeUpdatedEvent.parameters.push(
    new ethereum.EventParam("oldBps", ethereum.Value.fromUnsignedBigInt(oldBps))
  )
  platformFeeUpdatedEvent.parameters.push(
    new ethereum.EventParam("newBps", ethereum.Value.fromUnsignedBigInt(newBps))
  )

  return platformFeeUpdatedEvent
}

export function createPlatformFeesWithdrawnEvent(
  to: Address,
  amount: BigInt
): PlatformFeesWithdrawn {
  let platformFeesWithdrawnEvent =
    changetype<PlatformFeesWithdrawn>(newMockEvent())

  platformFeesWithdrawnEvent.parameters = new Array()

  platformFeesWithdrawnEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  platformFeesWithdrawnEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return platformFeesWithdrawnEvent
}

export function createUnpausedEvent(account: Address): Unpaused {
  let unpausedEvent = changetype<Unpaused>(newMockEvent())

  unpausedEvent.parameters = new Array()

  unpausedEvent.parameters.push(
    new ethereum.EventParam("account", ethereum.Value.fromAddress(account))
  )

  return unpausedEvent
}

export function createUntrackedEthWithdrawnEvent(
  to: Address,
  amount: BigInt
): UntrackedEthWithdrawn {
  let untrackedEthWithdrawnEvent =
    changetype<UntrackedEthWithdrawn>(newMockEvent())

  untrackedEthWithdrawnEvent.parameters = new Array()

  untrackedEthWithdrawnEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  untrackedEthWithdrawnEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return untrackedEthWithdrawnEvent
}
