import {
  ItemListed as ItemListedEvent,
  ItemSold as ItemSoldEvent,
  ListingCanceled as ListingCanceledEvent,
  ListingPriceUpdated as ListingPriceUpdatedEvent,
  MaxRoyaltyBpsUpdated as MaxRoyaltyBpsUpdatedEvent,
  OwnershipTransferred as OwnershipTransferredEvent,
  Paused as PausedEvent,
  PlatformFeeUpdated as PlatformFeeUpdatedEvent,
  PlatformFeesWithdrawn as PlatformFeesWithdrawnEvent,
  Unpaused as UnpausedEvent,
  UntrackedEthWithdrawn as UntrackedEthWithdrawnEvent
} from "../generated/NFTMarketPlace/NFTMarketPlace"
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
} from "../generated/schema"

export function handleItemListed(event: ItemListedEvent): void {
  let entity = new ItemListed(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.collection = event.params.collection
  entity.tokenId = event.params.tokenId
  entity.seller = event.params.seller
  entity.price = event.params.price

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleItemSold(event: ItemSoldEvent): void {
  let entity = new ItemSold(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.collection = event.params.collection
  entity.tokenId = event.params.tokenId
  entity.seller = event.params.seller
  entity.buyer = event.params.buyer
  entity.price = event.params.price
  entity.platformFee = event.params.platformFee
  entity.royaltyAmount = event.params.royaltyAmount
  entity.feeBpsSnapshot = event.params.feeBpsSnapshot

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleListingCanceled(event: ListingCanceledEvent): void {
  let entity = new ListingCanceled(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.collection = event.params.collection
  entity.tokenId = event.params.tokenId
  entity.seller = event.params.seller

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleListingPriceUpdated(
  event: ListingPriceUpdatedEvent
): void {
  let entity = new ListingPriceUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.collection = event.params.collection
  entity.tokenId = event.params.tokenId
  entity.seller = event.params.seller
  entity.oldPrice = event.params.oldPrice
  entity.newPrice = event.params.newPrice

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleMaxRoyaltyBpsUpdated(
  event: MaxRoyaltyBpsUpdatedEvent
): void {
  let entity = new MaxRoyaltyBpsUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.newBps = event.params.newBps

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleOwnershipTransferred(
  event: OwnershipTransferredEvent
): void {
  let entity = new OwnershipTransferred(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.previousOwner = event.params.previousOwner
  entity.newOwner = event.params.newOwner

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

export function handlePlatformFeeUpdated(event: PlatformFeeUpdatedEvent): void {
  let entity = new PlatformFeeUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.oldBps = event.params.oldBps
  entity.newBps = event.params.newBps

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handlePlatformFeesWithdrawn(
  event: PlatformFeesWithdrawnEvent
): void {
  let entity = new PlatformFeesWithdrawn(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.to = event.params.to
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

export function handleUntrackedEthWithdrawn(
  event: UntrackedEthWithdrawnEvent
): void {
  let entity = new UntrackedEthWithdrawn(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.to = event.params.to
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
