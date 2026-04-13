package codepulse

import (
	"net/http"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// WalletOverview 钱包总览：角色、统计、可用工作台。
// @Summary      Wallet overview
// @Tags         code-pulse
// @Produce      json
// @Param        address path string true "Wallet address (0x...)"
// @Success      200 {object} map[string]any
// @Router       /api/code-pulse/wallets/{address}/overview [get]
func WalletOverview(h *handlers.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireDB(h, c) {
			return
		}

		addr := normalizeAddress(c.Param("address"))

		isAdmin, _ := resolveGlobalRole(h, addr, "admin", false)
		isProposalInitiator, _ := resolveGlobalRole(h, addr, "proposal_initiator", false)

		var roles []models.CPWalletRole
		h.DB.Where("LOWER(wallet_address) = ? AND active = true", addr).Find(&roles)

		roleSet := make(map[string]bool)
		for _, r := range roles {
			roleSet[r.Role] = true
		}
		if isAdmin {
			roleSet["admin"] = true
		}
		if isProposalInitiator {
			roleSet["proposal_initiator"] = true
		}

		var proposalCount int64
		h.DB.Model(&models.CPProposal{}).
			Where("LOWER(organizer_address) = ?", addr).Count(&proposalCount)

		var campaignAsOrganizerCount int64
		h.DB.Model(&models.CPCampaign{}).
			Where("LOWER(organizer_address) = ?", addr).Count(&campaignAsOrganizerCount)

		var donationCount int64
		h.DB.Model(&models.CPContribution{}).
			Where("LOWER(contributor_address) = ?", addr).Count(&donationCount)

		var developerCampaignCount int64
		h.DB.Model(&models.CPCampaignDeveloper{}).
			Where("LOWER(developer_address) = ? AND is_active = true", addr).Count(&developerCampaignCount)

		dashboards := []string{}
		if roleSet["admin"] {
			dashboards = append(dashboards, "admin")
		}
		if roleSet["organizer"] || roleSet["proposal_initiator"] || proposalCount > 0 {
			dashboards = append(dashboards, "initiator")
		}
		if donationCount > 0 {
			dashboards = append(dashboards, "contributor")
		}
		if developerCampaignCount > 0 {
			dashboards = append(dashboards, "developer")
		}

		c.JSON(http.StatusOK, gin.H{
			"wallet_address":              addr,
			"roles":                       roles,
			"is_admin":                    isAdmin,
			"is_proposal_initiator":       isProposalInitiator,
			"proposal_count":              proposalCount,
			"campaign_as_organizer_count": campaignAsOrganizerCount,
			"donation_count":              donationCount,
			"developer_campaign_count":    developerCampaignCount,
			"available_dashboards":        dashboards,
		})
	}
}
