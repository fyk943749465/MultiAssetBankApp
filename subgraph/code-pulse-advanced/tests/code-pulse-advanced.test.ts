import {
  assert,
  describe,
  test,
  clearStore,
  beforeAll,
  afterAll
} from "matchstick-as/assembly/index"
import { BigInt, Address, Bytes } from "@graphprotocol/graph-ts"
import { CampaignFinalized } from "../generated/schema"
import { CampaignFinalized as CampaignFinalizedEvent } from "../generated/CodePulseAdvanced/CodePulseAdvanced"
import { handleCampaignFinalized } from "../src/code-pulse-advanced"
import { createCampaignFinalizedEvent } from "./code-pulse-advanced-utils"

// Tests structure (matchstick-as >=0.5.0)
// https://thegraph.com/docs/en/subgraphs/developing/creating/unit-testing-framework/#tests-structure

describe("Describe entity assertions", () => {
  beforeAll(() => {
    let campaignId = BigInt.fromI32(234)
    let successful = "boolean Not implemented"
    let newCampaignFinalizedEvent = createCampaignFinalizedEvent(
      campaignId,
      successful
    )
    handleCampaignFinalized(newCampaignFinalizedEvent)
  })

  afterAll(() => {
    clearStore()
  })

  // For more test scenarios, see:
  // https://thegraph.com/docs/en/subgraphs/developing/creating/unit-testing-framework/#write-a-unit-test

  test("CampaignFinalized created and stored", () => {
    assert.entityCount("CampaignFinalized", 1)

    // 0xa16081f360e3847006db660bae1c6d1b2e17ec2a is the default address used in newMockEvent() function
    assert.fieldEquals(
      "CampaignFinalized",
      "0xa16081f360e3847006db660bae1c6d1b2e17ec2a-1",
      "campaignId",
      "234"
    )
    assert.fieldEquals(
      "CampaignFinalized",
      "0xa16081f360e3847006db660bae1c6d1b2e17ec2a-1",
      "successful",
      "boolean Not implemented"
    )

    // More assert options:
    // https://thegraph.com/docs/en/subgraphs/developing/creating/unit-testing-framework/#asserts
  })
})
