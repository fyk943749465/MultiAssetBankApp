import {
  AuthorizedOracleSet as AuthorizedOracleSetEvent,
  NativeSwept as NativeSweptEvent,
  OwnershipTransferred as OwnershipTransferredEvent,
  TokenSwept as TokenSweptEvent,
} from "../generated/ReportsVerifier/ReportsVerifier"
import {
  AuthorizedOracleSet,
  OwnershipTransferred,
  VerifierNativeSwept,
  VerifierTokenSwept,
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

export function handleAuthorizedOracleSet(event: AuthorizedOracleSetEvent): void {
  let entity = new AuthorizedOracleSet(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.oracle = event.params.oracle
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()
}

export function handleTokenSwept(event: TokenSweptEvent): void {
  let entity = new VerifierTokenSwept(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.token = event.params.token
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()
}

export function handleNativeSwept(event: NativeSweptEvent): void {
  let entity = new VerifierNativeSwept(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()
}
