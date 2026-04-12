import { newMockEvent } from "matchstick-as"
import { ethereum, BigInt, Address, Bytes } from "@graphprotocol/graph-ts"
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
} from "../generated/CodePulseAdvanced/CodePulseAdvanced"

export function createCampaignFinalizedEvent(
  campaignId: BigInt,
  successful: boolean
): CampaignFinalized {
  let campaignFinalizedEvent = changetype<CampaignFinalized>(newMockEvent())

  campaignFinalizedEvent.parameters = new Array()

  campaignFinalizedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  campaignFinalizedEvent.parameters.push(
    new ethereum.EventParam(
      "successful",
      ethereum.Value.fromBoolean(successful)
    )
  )

  return campaignFinalizedEvent
}

export function createCrowdfundingLaunchedEvent(
  proposalId: BigInt,
  campaignId: BigInt,
  organizer: Address,
  githubUrlHash: Bytes,
  githubUrl: string,
  target: BigInt,
  deadline: BigInt,
  roundIndex: BigInt
): CrowdfundingLaunched {
  let crowdfundingLaunchedEvent =
    changetype<CrowdfundingLaunched>(newMockEvent())

  crowdfundingLaunchedEvent.parameters = new Array()

  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam(
      "proposalId",
      ethereum.Value.fromUnsignedBigInt(proposalId)
    )
  )
  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam("organizer", ethereum.Value.fromAddress(organizer))
  )
  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam(
      "githubUrlHash",
      ethereum.Value.fromFixedBytes(githubUrlHash)
    )
  )
  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam("githubUrl", ethereum.Value.fromString(githubUrl))
  )
  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam("target", ethereum.Value.fromUnsignedBigInt(target))
  )
  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam(
      "deadline",
      ethereum.Value.fromUnsignedBigInt(deadline)
    )
  )
  crowdfundingLaunchedEvent.parameters.push(
    new ethereum.EventParam(
      "roundIndex",
      ethereum.Value.fromUnsignedBigInt(roundIndex)
    )
  )

  return crowdfundingLaunchedEvent
}

export function createDeveloperAddedEvent(
  campaignId: BigInt,
  developer: Address
): DeveloperAdded {
  let developerAddedEvent = changetype<DeveloperAdded>(newMockEvent())

  developerAddedEvent.parameters = new Array()

  developerAddedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  developerAddedEvent.parameters.push(
    new ethereum.EventParam("developer", ethereum.Value.fromAddress(developer))
  )

  return developerAddedEvent
}

export function createDeveloperRemovedEvent(
  campaignId: BigInt,
  developer: Address
): DeveloperRemoved {
  let developerRemovedEvent = changetype<DeveloperRemoved>(newMockEvent())

  developerRemovedEvent.parameters = new Array()

  developerRemovedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  developerRemovedEvent.parameters.push(
    new ethereum.EventParam("developer", ethereum.Value.fromAddress(developer))
  )

  return developerRemovedEvent
}

export function createDonatedEvent(
  campaignId: BigInt,
  contributor: Address,
  amount: BigInt
): Donated {
  let donatedEvent = changetype<Donated>(newMockEvent())

  donatedEvent.parameters = new Array()

  donatedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  donatedEvent.parameters.push(
    new ethereum.EventParam(
      "contributor",
      ethereum.Value.fromAddress(contributor)
    )
  )
  donatedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return donatedEvent
}

export function createFundingRoundReviewedEvent(
  proposalId: BigInt,
  approved: boolean
): FundingRoundReviewed {
  let fundingRoundReviewedEvent =
    changetype<FundingRoundReviewed>(newMockEvent())

  fundingRoundReviewedEvent.parameters = new Array()

  fundingRoundReviewedEvent.parameters.push(
    new ethereum.EventParam(
      "proposalId",
      ethereum.Value.fromUnsignedBigInt(proposalId)
    )
  )
  fundingRoundReviewedEvent.parameters.push(
    new ethereum.EventParam("approved", ethereum.Value.fromBoolean(approved))
  )

  return fundingRoundReviewedEvent
}

export function createFundingRoundSubmittedForReviewEvent(
  proposalId: BigInt,
  roundOrdinal: BigInt
): FundingRoundSubmittedForReview {
  let fundingRoundSubmittedForReviewEvent =
    changetype<FundingRoundSubmittedForReview>(newMockEvent())

  fundingRoundSubmittedForReviewEvent.parameters = new Array()

  fundingRoundSubmittedForReviewEvent.parameters.push(
    new ethereum.EventParam(
      "proposalId",
      ethereum.Value.fromUnsignedBigInt(proposalId)
    )
  )
  fundingRoundSubmittedForReviewEvent.parameters.push(
    new ethereum.EventParam(
      "roundOrdinal",
      ethereum.Value.fromUnsignedBigInt(roundOrdinal)
    )
  )

  return fundingRoundSubmittedForReviewEvent
}

export function createMilestoneApprovedEvent(
  campaignId: BigInt,
  milestoneIndex: BigInt
): MilestoneApproved {
  let milestoneApprovedEvent = changetype<MilestoneApproved>(newMockEvent())

  milestoneApprovedEvent.parameters = new Array()

  milestoneApprovedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  milestoneApprovedEvent.parameters.push(
    new ethereum.EventParam(
      "milestoneIndex",
      ethereum.Value.fromUnsignedBigInt(milestoneIndex)
    )
  )

  return milestoneApprovedEvent
}

export function createMilestoneShareClaimedEvent(
  campaignId: BigInt,
  milestoneIndex: BigInt,
  developer: Address,
  amount: BigInt
): MilestoneShareClaimed {
  let milestoneShareClaimedEvent =
    changetype<MilestoneShareClaimed>(newMockEvent())

  milestoneShareClaimedEvent.parameters = new Array()

  milestoneShareClaimedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  milestoneShareClaimedEvent.parameters.push(
    new ethereum.EventParam(
      "milestoneIndex",
      ethereum.Value.fromUnsignedBigInt(milestoneIndex)
    )
  )
  milestoneShareClaimedEvent.parameters.push(
    new ethereum.EventParam("developer", ethereum.Value.fromAddress(developer))
  )
  milestoneShareClaimedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return milestoneShareClaimedEvent
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

export function createPlatformDonatedEvent(
  donor: Address,
  amount: BigInt
): PlatformDonated {
  let platformDonatedEvent = changetype<PlatformDonated>(newMockEvent())

  platformDonatedEvent.parameters = new Array()

  platformDonatedEvent.parameters.push(
    new ethereum.EventParam("donor", ethereum.Value.fromAddress(donor))
  )
  platformDonatedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return platformDonatedEvent
}

export function createPlatformFundsWithdrawnEvent(
  to: Address,
  amount: BigInt
): PlatformFundsWithdrawn {
  let platformFundsWithdrawnEvent =
    changetype<PlatformFundsWithdrawn>(newMockEvent())

  platformFundsWithdrawnEvent.parameters = new Array()

  platformFundsWithdrawnEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  platformFundsWithdrawnEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return platformFundsWithdrawnEvent
}

export function createProposalInitiatorUpdatedEvent(
  account: Address,
  allowed: boolean
): ProposalInitiatorUpdated {
  let proposalInitiatorUpdatedEvent =
    changetype<ProposalInitiatorUpdated>(newMockEvent())

  proposalInitiatorUpdatedEvent.parameters = new Array()

  proposalInitiatorUpdatedEvent.parameters.push(
    new ethereum.EventParam("account", ethereum.Value.fromAddress(account))
  )
  proposalInitiatorUpdatedEvent.parameters.push(
    new ethereum.EventParam("allowed", ethereum.Value.fromBoolean(allowed))
  )

  return proposalInitiatorUpdatedEvent
}

export function createProposalReviewedEvent(
  proposalId: BigInt,
  approved: boolean
): ProposalReviewed {
  let proposalReviewedEvent = changetype<ProposalReviewed>(newMockEvent())

  proposalReviewedEvent.parameters = new Array()

  proposalReviewedEvent.parameters.push(
    new ethereum.EventParam(
      "proposalId",
      ethereum.Value.fromUnsignedBigInt(proposalId)
    )
  )
  proposalReviewedEvent.parameters.push(
    new ethereum.EventParam("approved", ethereum.Value.fromBoolean(approved))
  )

  return proposalReviewedEvent
}

export function createProposalSubmittedEvent(
  proposalId: BigInt,
  organizer: Address,
  githubUrl: string,
  target: BigInt,
  duration: BigInt
): ProposalSubmitted {
  let proposalSubmittedEvent = changetype<ProposalSubmitted>(newMockEvent())

  proposalSubmittedEvent.parameters = new Array()

  proposalSubmittedEvent.parameters.push(
    new ethereum.EventParam(
      "proposalId",
      ethereum.Value.fromUnsignedBigInt(proposalId)
    )
  )
  proposalSubmittedEvent.parameters.push(
    new ethereum.EventParam("organizer", ethereum.Value.fromAddress(organizer))
  )
  proposalSubmittedEvent.parameters.push(
    new ethereum.EventParam("githubUrl", ethereum.Value.fromString(githubUrl))
  )
  proposalSubmittedEvent.parameters.push(
    new ethereum.EventParam("target", ethereum.Value.fromUnsignedBigInt(target))
  )
  proposalSubmittedEvent.parameters.push(
    new ethereum.EventParam(
      "duration",
      ethereum.Value.fromUnsignedBigInt(duration)
    )
  )

  return proposalSubmittedEvent
}

export function createRefundClaimedEvent(
  campaignId: BigInt,
  contributor: Address,
  amount: BigInt
): RefundClaimed {
  let refundClaimedEvent = changetype<RefundClaimed>(newMockEvent())

  refundClaimedEvent.parameters = new Array()

  refundClaimedEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  refundClaimedEvent.parameters.push(
    new ethereum.EventParam(
      "contributor",
      ethereum.Value.fromAddress(contributor)
    )
  )
  refundClaimedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return refundClaimedEvent
}

export function createStaleFundsSweptEvent(
  campaignId: BigInt,
  amount: BigInt
): StaleFundsSwept {
  let staleFundsSweptEvent = changetype<StaleFundsSwept>(newMockEvent())

  staleFundsSweptEvent.parameters = new Array()

  staleFundsSweptEvent.parameters.push(
    new ethereum.EventParam(
      "campaignId",
      ethereum.Value.fromUnsignedBigInt(campaignId)
    )
  )
  staleFundsSweptEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )

  return staleFundsSweptEvent
}

export function createUnpausedEvent(account: Address): Unpaused {
  let unpausedEvent = changetype<Unpaused>(newMockEvent())

  unpausedEvent.parameters = new Array()

  unpausedEvent.parameters.push(
    new ethereum.EventParam("account", ethereum.Value.fromAddress(account))
  )

  return unpausedEvent
}
