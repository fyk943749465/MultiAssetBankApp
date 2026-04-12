import {
  CampaignFinalized as CampaignFinalizedEvent,
  CrowdfundingLaunched as CrowdfundingLaunchedEvent,
  DeveloperAdded as DeveloperAddedEvent,
  DeveloperRemoved as DeveloperRemovedEvent,
  Donated as DonatedEvent,
  FundingRoundReviewed as FundingRoundReviewedEvent,
  FundingRoundSubmittedForReview as FundingRoundSubmittedForReviewEvent,
  MilestoneApproved as MilestoneApprovedEvent,
  MilestoneShareClaimed as MilestoneShareClaimedEvent,
  OwnershipTransferred as OwnershipTransferredEvent,
  Paused as PausedEvent,
  PlatformDonated as PlatformDonatedEvent,
  PlatformFundsWithdrawn as PlatformFundsWithdrawnEvent,
  ProposalInitiatorUpdated as ProposalInitiatorUpdatedEvent,
  ProposalReviewed as ProposalReviewedEvent,
  ProposalSubmitted as ProposalSubmittedEvent,
  RefundClaimed as RefundClaimedEvent,
  StaleFundsSwept as StaleFundsSweptEvent,
  Unpaused as UnpausedEvent
} from "../generated/CodePulseAdvanced/CodePulseAdvanced"
import {
  CampaignFinalized,
  CrowdfundingLaunched,
  DeveloperAdded,
  DeveloperRemoved,
  Donated,
  FundingRoundReviewed,
  FundingRoundSubmittedForReview,
  MilestoneApproved,
  MilestoneShareClaimed,
  OwnershipTransferred,
  Paused,
  PlatformDonated,
  PlatformFundsWithdrawn,
  ProposalInitiatorUpdated,
  ProposalReviewed,
  ProposalSubmitted,
  RefundClaimed,
  StaleFundsSwept,
  Unpaused
} from "../generated/schema"

export function handleCampaignFinalized(event: CampaignFinalizedEvent): void {
  let entity = new CampaignFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
  entity.successful = event.params.successful

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleCrowdfundingLaunched(
  event: CrowdfundingLaunchedEvent
): void {
  let entity = new CrowdfundingLaunched(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.proposalId = event.params.proposalId
  entity.campaignId = event.params.campaignId
  entity.organizer = event.params.organizer
  entity.githubUrlHash = event.params.githubUrlHash
  entity.githubUrl = event.params.githubUrl
  entity.target = event.params.target
  entity.deadline = event.params.deadline
  entity.roundIndex = event.params.roundIndex

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleDeveloperAdded(event: DeveloperAddedEvent): void {
  let entity = new DeveloperAdded(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
  entity.developer = event.params.developer

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleDeveloperRemoved(event: DeveloperRemovedEvent): void {
  let entity = new DeveloperRemoved(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
  entity.developer = event.params.developer

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleDonated(event: DonatedEvent): void {
  let entity = new Donated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
  entity.contributor = event.params.contributor
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleFundingRoundReviewed(
  event: FundingRoundReviewedEvent
): void {
  let entity = new FundingRoundReviewed(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.proposalId = event.params.proposalId
  entity.approved = event.params.approved

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleFundingRoundSubmittedForReview(
  event: FundingRoundSubmittedForReviewEvent
): void {
  let entity = new FundingRoundSubmittedForReview(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.proposalId = event.params.proposalId
  entity.roundOrdinal = event.params.roundOrdinal

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleMilestoneApproved(event: MilestoneApprovedEvent): void {
  let entity = new MilestoneApproved(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
  entity.milestoneIndex = event.params.milestoneIndex

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleMilestoneShareClaimed(
  event: MilestoneShareClaimedEvent
): void {
  let entity = new MilestoneShareClaimed(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
  entity.milestoneIndex = event.params.milestoneIndex
  entity.developer = event.params.developer
  entity.amount = event.params.amount

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

export function handlePlatformDonated(event: PlatformDonatedEvent): void {
  let entity = new PlatformDonated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.donor = event.params.donor
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handlePlatformFundsWithdrawn(
  event: PlatformFundsWithdrawnEvent
): void {
  let entity = new PlatformFundsWithdrawn(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.to = event.params.to
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleProposalInitiatorUpdated(
  event: ProposalInitiatorUpdatedEvent
): void {
  let entity = new ProposalInitiatorUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.account = event.params.account
  entity.allowed = event.params.allowed

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleProposalReviewed(event: ProposalReviewedEvent): void {
  let entity = new ProposalReviewed(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.proposalId = event.params.proposalId
  entity.approved = event.params.approved

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleProposalSubmitted(event: ProposalSubmittedEvent): void {
  let entity = new ProposalSubmitted(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.proposalId = event.params.proposalId
  entity.organizer = event.params.organizer
  entity.githubUrl = event.params.githubUrl
  entity.target = event.params.target
  entity.duration = event.params.duration

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleRefundClaimed(event: RefundClaimedEvent): void {
  let entity = new RefundClaimed(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
  entity.contributor = event.params.contributor
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleStaleFundsSwept(event: StaleFundsSweptEvent): void {
  let entity = new StaleFundsSwept(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.campaignId = event.params.campaignId
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
