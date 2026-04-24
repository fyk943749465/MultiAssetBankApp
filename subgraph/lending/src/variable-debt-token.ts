import {
  Mint as MintEvent,
  Burn as BurnEvent,
} from "../generated/templates/VariableDebtToken/VariableDebtToken"
import { VariableDebtTokenBurn, VariableDebtTokenMint } from "../generated/schema"

export function handleVariableDebtTokenMint(event: MintEvent): void {
  let entity = new VariableDebtTokenMint(
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

export function handleVariableDebtTokenBurn(event: BurnEvent): void {
  let entity = new VariableDebtTokenBurn(
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
