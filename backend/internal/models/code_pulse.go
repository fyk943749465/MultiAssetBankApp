package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ---------------------------------------------------------------------------
// JSON 辅助类型（用于 GORM JSONB 字段）
// ---------------------------------------------------------------------------

// JSONB 映射 PostgreSQL 的 jsonb 列；在 Go 侧表现为 json.RawMessage。
type JSONB json.RawMessage

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONB) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		cp := make([]byte, len(v))
		copy(cp, v)
		*j = cp
	case string:
		*j = []byte(v)
	default:
		return errors.New("models: unsupported JSONB source type")
	}
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	cp := make([]byte, len(data))
	copy(cp, data)
	*j = cp
	return nil
}

// ---------------------------------------------------------------------------
// 5. 主实体
// ---------------------------------------------------------------------------

// CPProposal 提案主表：保存每个众筹提案的聚合态。
type CPProposal struct {
	ProposalID               uint64     `gorm:"primaryKey;column:proposal_id" json:"proposal_id"`
	OrganizerAddress         string     `gorm:"size:42;not null;index:idx_cp_proposals_organizer_address" json:"organizer_address"`
	GithubURL                string     `gorm:"not null" json:"github_url"`
	GithubURLHash            *string    `json:"github_url_hash"`
	TargetWei                string     `gorm:"type:numeric(78,0);not null" json:"target_wei"`
	DurationSeconds          int64      `gorm:"not null" json:"duration_seconds"`
	Status                   string     `gorm:"not null;index:idx_cp_proposals_status" json:"status"`
	StatusCode               int        `gorm:"not null" json:"status_code"`
	LastCampaignID           *uint64    `gorm:"index:idx_cp_proposals_last_campaign_id" json:"last_campaign_id"`
	CurrentRoundCount        int        `gorm:"not null;default:0" json:"current_round_count"`
	PendingRoundTargetWei    *string    `gorm:"type:numeric(78,0)" json:"pending_round_target_wei"`
	PendingRoundDurationSecs *int64     `gorm:"column:pending_round_duration_seconds" json:"pending_round_duration_seconds"`
	RoundReviewState         *string    `json:"round_review_state"`
	RoundReviewStateCode     *int       `json:"round_review_state_code"`
	SubmittedTxHash          *string    `gorm:"size:66" json:"submitted_tx_hash"`
	SubmittedBlockNumber     *uint64    `json:"submitted_block_number"`
	SubmittedAt              *time.Time `json:"submitted_at"`
	ReviewedAt               *time.Time `json:"reviewed_at"`
	ApprovedAt               *time.Time `json:"approved_at"`
	RejectedAt               *time.Time `json:"rejected_at"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

func (CPProposal) TableName() string { return "cp_proposals" }

// CPCampaign 众筹轮次表：每次 launchApprovedRound 后创建一条记录。
type CPCampaign struct {
	CampaignID             uint64     `gorm:"primaryKey;column:campaign_id" json:"campaign_id"`
	ProposalID             uint64     `gorm:"not null;index:idx_cp_campaigns_proposal_id" json:"proposal_id"`
	RoundIndex             int        `gorm:"not null" json:"round_index"`
	OrganizerAddress       string     `gorm:"size:42;not null;index:idx_cp_campaigns_organizer_address" json:"organizer_address"`
	GithubURL              string     `gorm:"not null" json:"github_url"`
	TargetWei              string     `gorm:"type:numeric(78,0);not null" json:"target_wei"`
	DeadlineAt             time.Time  `gorm:"not null;index:idx_cp_campaigns_deadline_at" json:"deadline_at"`
	AmountRaisedWei        string     `gorm:"type:numeric(78,0);not null;default:0" json:"amount_raised_wei"`
	TotalWithdrawnWei      string     `gorm:"type:numeric(78,0);not null;default:0" json:"total_withdrawn_wei"`
	UnclaimedRefundPoolWei string     `gorm:"type:numeric(78,0);not null;default:0" json:"unclaimed_refund_pool_wei"`
	State                  string     `gorm:"not null;index:idx_cp_campaigns_state" json:"state"`
	StateCode              int        `gorm:"not null" json:"state_code"`
	DonorCount             int        `gorm:"not null;default:0" json:"donor_count"`
	DeveloperCount         int        `gorm:"not null;default:0" json:"developer_count"`
	FinalizedAt            *time.Time `json:"finalized_at"`
	SuccessAt              *time.Time `json:"success_at"`
	DormantFundsSwept      bool       `gorm:"not null;default:false" json:"dormant_funds_swept"`
	LaunchedTxHash         string     `gorm:"size:66;not null" json:"launched_tx_hash"`
	LaunchedBlockNumber    uint64     `gorm:"not null" json:"launched_block_number"`
	LaunchedAt             time.Time  `gorm:"not null" json:"launched_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (CPCampaign) TableName() string { return "cp_campaigns" }

// CPContribution 捐款聚合表：按 (campaign, contributor) 聚合累计捐款与退款。
type CPContribution struct {
	CampaignID          uint64     `gorm:"primaryKey;column:campaign_id" json:"campaign_id"`
	ContributorAddress  string     `gorm:"primaryKey;size:42;column:contributor_address;index:idx_cp_contributions_contributor" json:"contributor_address"`
	TotalContributedWei string     `gorm:"type:numeric(78,0);not null;default:0" json:"total_contributed_wei"`
	RefundClaimedWei    string     `gorm:"type:numeric(78,0);not null;default:0" json:"refund_claimed_wei"`
	LastDonatedAt       *time.Time `json:"last_donated_at"`
	LastRefundAt        *time.Time `json:"last_refund_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (CPContribution) TableName() string { return "cp_contributions" }

// ---------------------------------------------------------------------------
// 6. 里程碑与成员
// ---------------------------------------------------------------------------

// CPProposalMilestone 提案里程碑定义表。
type CPProposalMilestone struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProposalID     uint64    `gorm:"not null;uniqueIndex:uq_cp_proposal_milestones_natural;index:idx_cp_proposal_milestones_proposal" json:"proposal_id"`
	RoundOrdinal   int       `gorm:"not null;uniqueIndex:uq_cp_proposal_milestones_natural" json:"round_ordinal"`
	MilestoneIndex int       `gorm:"not null;uniqueIndex:uq_cp_proposal_milestones_natural" json:"milestone_index"`
	Description    string    `gorm:"not null" json:"description"`
	PercentageRaw  string    `gorm:"type:numeric(20,0);not null" json:"percentage_raw"`
	SourceType     string    `gorm:"not null;uniqueIndex:uq_cp_proposal_milestones_natural" json:"source_type"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (CPProposalMilestone) TableName() string { return "cp_proposal_milestones" }

// CPCampaignMilestone 众筹里程碑快照表：launch 后的里程碑审批与放款状态。
type CPCampaignMilestone struct {
	CampaignID     uint64     `gorm:"primaryKey;column:campaign_id" json:"campaign_id"`
	MilestoneIndex int        `gorm:"primaryKey;column:milestone_index" json:"milestone_index"`
	Description    string     `gorm:"not null" json:"description"`
	PercentageRaw  string     `gorm:"type:numeric(20,0);not null" json:"percentage_raw"`
	Approved       bool       `gorm:"not null;default:false" json:"approved"`
	Claimed        bool       `gorm:"not null;default:false" json:"claimed"`
	ApprovedAt     *time.Time `json:"approved_at"`
	UnlockAt       *time.Time `json:"unlock_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (CPCampaignMilestone) TableName() string { return "cp_campaign_milestones" }

// CPCampaignDeveloper 众筹开发者名单表：记录 add/remove 历史。
type CPCampaignDeveloper struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CampaignID       uint64     `gorm:"not null;index:idx_cp_campaign_developers_campaign_developer;index:idx_cp_campaign_developers_active" json:"campaign_id"`
	DeveloperAddress string     `gorm:"size:42;not null;index:idx_cp_campaign_developers_campaign_developer;index:idx_cp_campaign_developers_developer_address" json:"developer_address"`
	IsActive         bool       `gorm:"not null;default:true" json:"is_active"`
	AddedTxHash      *string    `gorm:"size:66" json:"added_tx_hash"`
	RemovedTxHash    *string    `gorm:"size:66" json:"removed_tx_hash"`
	AddedAt          *time.Time `json:"added_at"`
	RemovedAt        *time.Time `json:"removed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (CPCampaignDeveloper) TableName() string { return "cp_campaign_developers" }

// CPMilestoneClaim 里程碑领取记录表。
type CPMilestoneClaim struct {
	CampaignID       uint64    `gorm:"primaryKey;column:campaign_id;index:idx_cp_milestone_claims_campaign" json:"campaign_id"`
	MilestoneIndex   int       `gorm:"primaryKey;column:milestone_index" json:"milestone_index"`
	DeveloperAddress string    `gorm:"primaryKey;size:42;column:developer_address;index:idx_cp_milestone_claims_developer" json:"developer_address"`
	ClaimedAmountWei string    `gorm:"type:numeric(78,0);not null" json:"claimed_amount_wei"`
	ClaimedTxHash    string    `gorm:"size:66;not null" json:"claimed_tx_hash"`
	ClaimedAt        time.Time `gorm:"not null" json:"claimed_at"`
}

func (CPMilestoneClaim) TableName() string { return "cp_milestone_claims" }

// ---------------------------------------------------------------------------
// 7. 事件流水与同步
// ---------------------------------------------------------------------------

// CPEventLog 统一事件流水表：从子图同步的所有链上事件。
type CPEventLog struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID         uint64    `gorm:"not null" json:"chain_id"`
	ContractAddress string    `gorm:"size:42;not null" json:"contract_address"`
	EventName       string    `gorm:"not null;index:idx_cp_event_log_event_name" json:"event_name"`
	ProposalID      *uint64   `gorm:"index:idx_cp_event_log_proposal_id" json:"proposal_id"`
	CampaignID      *uint64   `gorm:"index:idx_cp_event_log_campaign_id" json:"campaign_id"`
	WalletAddress   *string   `gorm:"size:42;index:idx_cp_event_log_wallet_address" json:"wallet_address"`
	EntityKey       *string   `json:"entity_key"`
	TxHash          string    `gorm:"size:66;not null;uniqueIndex:uq_cp_event_log_tx_log" json:"tx_hash"`
	LogIndex        int       `gorm:"not null;uniqueIndex:uq_cp_event_log_tx_log" json:"log_index"`
	BlockNumber     uint64    `gorm:"not null;index:idx_cp_event_log_block_number_desc" json:"block_number"`
	BlockTimestamp  time.Time `gorm:"not null" json:"block_timestamp"`
	Payload         JSONB     `gorm:"type:jsonb;not null" json:"payload"`
	Source          string    `gorm:"not null;default:'subgraph'" json:"source"`
	CreatedAt       time.Time `json:"created_at"`
}

func (CPEventLog) TableName() string { return "cp_event_log" }

// CPSyncCursor 同步游标表：记录子图增量同步进度。
type CPSyncCursor struct {
	SyncName           string     `gorm:"primaryKey;column:sync_name" json:"sync_name"`
	LastBlockNumber    *uint64    `json:"last_block_number"`
	LastBlockTimestamp *time.Time `json:"last_block_timestamp"`
	LastEventID        *string    `json:"last_event_id"`
	// 子图可达性（与链上重组无关；重组需运维策略，见 README）
	LastSubgraphQueryOKAt     *time.Time `json:"last_subgraph_query_ok_at"`
	LastSubgraphError         *string    `json:"last_subgraph_error"`
	LastSubgraphErrorAt       *time.Time `json:"last_subgraph_error_at"`
	SubgraphConsecutiveErrors int        `gorm:"not null;default:0" json:"subgraph_consecutive_errors"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

func (CPSyncCursor) TableName() string { return "cp_sync_cursors" }

// CPSystemState 链上系统状态缓存：保存 owner / paused 等链上为准的数据。
type CPSystemState struct {
	ContractAddress   string     `gorm:"primaryKey;size:42;column:contract_address" json:"contract_address"`
	OwnerAddress      string     `gorm:"size:42;not null" json:"owner_address"`
	Paused            bool       `gorm:"not null;default:false" json:"paused"`
	Source            string     `gorm:"not null;default:'chain'" json:"source"`
	SourceBlockNumber *uint64    `json:"source_block_number"`
	SyncedAt          *time.Time `json:"synced_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (CPSystemState) TableName() string { return "cp_system_states" }

// ---------------------------------------------------------------------------
// 8. 角色与画像
// ---------------------------------------------------------------------------

// CPWalletProfile 钱包画像表：保存用户展示信息。
type CPWalletProfile struct {
	WalletAddress  string    `gorm:"primaryKey;size:42;column:wallet_address" json:"wallet_address"`
	DisplayName    *string   `json:"display_name"`
	GithubUsername *string   `json:"github_username"`
	AvatarURL      *string   `json:"avatar_url"`
	Bio            *string   `json:"bio"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (CPWalletProfile) TableName() string { return "cp_wallet_profiles" }

// CPWalletRole 钱包角色表：按 (钱包, 角色, 作用域) 记录身份。
type CPWalletRole struct {
	ID                uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	WalletAddress     string     `gorm:"size:42;not null;index:idx_cp_wallet_roles_wallet" json:"wallet_address"`
	Role              string     `gorm:"not null" json:"role"`
	ScopeType         string     `gorm:"not null" json:"scope_type"`
	ScopeID           *string    `json:"scope_id"`
	Active            bool       `gorm:"not null;default:true" json:"active"`
	DerivedFrom       string     `gorm:"not null" json:"derived_from"`
	Source            string     `gorm:"not null;default:'db'" json:"source"`
	SourceBlockNumber *uint64    `json:"source_block_number"`
	SyncedAt          *time.Time `json:"synced_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (CPWalletRole) TableName() string { return "cp_wallet_roles" }

// ---------------------------------------------------------------------------
// 9. 交易尝试与错误
// ---------------------------------------------------------------------------

// CPTxAttempt 交易尝试表：记录每次链上动作的完整生命周期与 custom error 解码。
type CPTxAttempt struct {
	ID                 uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	WalletAddress      string    `gorm:"size:42;not null" json:"wallet_address"`
	RoleSnapshot       JSONB     `gorm:"type:jsonb;not null" json:"role_snapshot"`
	Action             string    `gorm:"not null" json:"action"`
	ProposalID         *uint64   `json:"proposal_id"`
	CampaignID         *uint64   `json:"campaign_id"`
	MilestoneIndex     *int      `json:"milestone_index"`
	RequestPayload     JSONB     `gorm:"type:jsonb;not null" json:"request_payload"`
	SimulationOK       *bool     `json:"simulation_ok"`
	RevertErrorName    *string   `json:"revert_error_name"`
	RevertErrorArgs    JSONB     `gorm:"type:jsonb" json:"revert_error_args"`
	TxHash             *string   `gorm:"size:66;index:idx_cp_tx_attempts_tx_hash" json:"tx_hash"`
	TxStatus           string    `gorm:"not null" json:"tx_status"`
	ReceiptBlockNumber *uint64   `json:"receipt_block_number"`
	FailureStage       *string   `json:"failure_stage"`
	FailureMessage     *string   `json:"failure_message"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (CPTxAttempt) TableName() string { return "cp_tx_attempts" }

// ---------------------------------------------------------------------------
// 10. 平台资金流水
// ---------------------------------------------------------------------------

// CPPlatformFundMovement 平台资金流水表：记录平台捐赠与提现明细。
type CPPlatformFundMovement struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Direction      string    `gorm:"not null;index:idx_cp_platform_fund_movements_direction" json:"direction"`
	WalletAddress  string    `gorm:"size:42;not null;index:idx_cp_platform_fund_movements_wallet" json:"wallet_address"`
	AmountWei      string    `gorm:"type:numeric(78,0);not null" json:"amount_wei"`
	TxHash         string    `gorm:"size:66;not null;uniqueIndex:uq_cp_platform_fund_movements_tx_log" json:"tx_hash"`
	LogIndex       int       `gorm:"not null;uniqueIndex:uq_cp_platform_fund_movements_tx_log" json:"log_index"`
	BlockNumber    uint64    `gorm:"not null;index:idx_cp_platform_fund_movements_block_desc" json:"block_number"`
	BlockTimestamp time.Time `gorm:"not null" json:"block_timestamp"`
	CreatedAt      time.Time `json:"created_at"`
}

func (CPPlatformFundMovement) TableName() string { return "cp_platform_fund_movements" }

// ---------------------------------------------------------------------------
// 11. 每日统计快照
// ---------------------------------------------------------------------------

// CPSnapshotDaily 每日统计快照表：为首页 dashboard 提供历史趋势数据。
type CPSnapshotDaily struct {
	SnapshotDate            string    `gorm:"primaryKey;type:date;column:snapshot_date" json:"snapshot_date"`
	ProposalCount           int       `gorm:"not null" json:"proposal_count"`
	CampaignCount           int       `gorm:"not null" json:"campaign_count"`
	LiveCampaignCount       int       `gorm:"not null" json:"live_campaign_count"`
	SuccessfulCampaignCount int       `gorm:"not null" json:"successful_campaign_count"`
	FailedCampaignCount     int       `gorm:"not null" json:"failed_campaign_count"`
	TotalRaisedWei          string    `gorm:"type:numeric(78,0);not null" json:"total_raised_wei"`
	TotalRefundedWei        string    `gorm:"type:numeric(78,0);not null" json:"total_refunded_wei"`
	CreatedAt               time.Time `json:"created_at"`
}

func (CPSnapshotDaily) TableName() string { return "cp_snapshots_daily" }
