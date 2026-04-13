package codepulse

import (
	"go-chain/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Register 注册 /api/code-pulse 下全部路由。
func Register(r *gin.Engine, h *handlers.Handlers) {
	cp := r.Group("/api/code-pulse")

	cp.GET("/summary", Summary(h))
	cp.GET("/config", Config(h))

	cp.GET("/proposals", Proposals(h))
	cp.GET("/proposals/:proposalId", ProposalDetail(h))
	cp.GET("/proposals/:proposalId/timeline", ProposalTimeline(h))

	cp.GET("/campaigns", Campaigns(h))
	cp.GET("/campaigns/:campaignId", CampaignDetail(h))
	cp.GET("/campaigns/:campaignId/timeline", CampaignTimeline(h))
	cp.GET("/campaigns/:campaignId/contributions", CampaignContributions(h))

	cp.GET("/wallets/:address/overview", WalletOverview(h))

	cp.POST("/actions/check", ActionCheck(h))

	cp.POST("/tx/build", TxBuild(h))
	cp.POST("/tx/submit", TxSubmit(h))
	cp.GET("/tx/:attemptId", TxDetail(h))

	cp.GET("/admin/dashboard", AdminDashboard(h))
	cp.GET("/initiators/:address/dashboard", InitiatorDashboard(h))
	cp.GET("/contributors/:address/dashboard", ContributorDashboard(h))
	cp.GET("/developers/:address/dashboard", DeveloperDashboard(h))

	cp.GET("/admin/proposal-initiators", ListInitiators(h))
	cp.POST("/admin/proposal-initiators", AddInitiator(h))
	cp.DELETE("/admin/proposal-initiators/:address", RemoveInitiator(h))
	cp.GET("/admin/platform-funds", PlatformFunds(h))
	cp.GET("/admin/sync-status", SyncStatus(h))
}
