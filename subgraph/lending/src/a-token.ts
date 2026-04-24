import { Mint as MintEvent, Burn as BurnEvent } from "../generated/templates/AToken/AToken"
import { ATokenBurn, ATokenMint } from "../generated/schema"

export function handleATokenMint(event: MintEvent): void {
  let entity = new ATokenMint(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.token = event.address
  entity.to = event.params.to
  entity.scaledAmount = event.params.scaledAmount
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()
}

export function handleATokenBurn(event: BurnEvent): void {
  let entity = new ATokenBurn(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.token = event.address
  entity.from = event.params.from
  entity.scaledAmount = event.params.scaledAmount
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()
}
