package models

import (
	"time"
)

// NFTAccount 对应 nft_accounts。
type NFTAccount struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID   int64     `gorm:"not null;index:ux_nft_accounts_chain_address,unique,priority:1" json:"chain_id"`
	Address   string    `gorm:"size:42;not null;index:ux_nft_accounts_chain_address,unique,priority:2" json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

func (NFTAccount) TableName() string { return "nft_accounts" }

// NFTContract 对应 nft_contracts。
type NFTContract struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID          int64     `gorm:"not null;index:ux_nft_contracts_chain_address,unique,priority:1" json:"chain_id"`
	Address          string    `gorm:"size:42;not null;index:ux_nft_contracts_chain_address,unique,priority:2" json:"address"`
	ContractKind     string    `gorm:"size:32;not null" json:"contract_kind"`
	DisplayLabel     *string   `gorm:"size:128" json:"display_label,omitempty"`
	DeployedBlock    *int64    `json:"deployed_block,omitempty"`
	DeployedTxHash   *string   `gorm:"size:66" json:"deployed_tx_hash,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

func (NFTContract) TableName() string { return "nft_contracts" }

// NFTCollection 对应 nft_collections。
type NFTCollection struct {
	ID                   uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID              int64     `gorm:"not null;index:ux_nft_collections_chain_contract,unique,priority:1;index:ux_nft_collections_chain_tx_log,unique,priority:1" json:"chain_id"`
	ContractID           uint64    `gorm:"not null;index:ux_nft_collections_chain_contract,unique,priority:2" json:"contract_id"`
	CreatorAccountID     uint64    `gorm:"not null" json:"creator_account_id"`
	CollectionName       *string   `gorm:"type:text" json:"collection_name,omitempty"`
	CollectionSymbol     *string   `gorm:"type:text" json:"collection_symbol,omitempty"`
	BaseURI              *string   `gorm:"type:text" json:"base_uri,omitempty"`
	DeploySaltHex        *string   `gorm:"size:66" json:"deploy_salt_hex,omitempty"`
	FeePaidWei           *string   `gorm:"column:fee_paid_wei;type:numeric(78,0)" json:"fee_paid_wei,omitempty"`
	CreatedBlockNumber   int64     `gorm:"not null" json:"created_block_number"`
	CreatedTxHash        string    `gorm:"size:66;not null;index:ux_nft_collections_chain_tx_log,unique,priority:2" json:"created_tx_hash"`
	CreatedLogIndex      int       `gorm:"not null;index:ux_nft_collections_chain_tx_log,unique,priority:3" json:"created_log_index"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (NFTCollection) TableName() string { return "nft_collections" }

// NFTToken 对应 nft_tokens。
type NFTToken struct {
	ID                   uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID              int64     `gorm:"not null" json:"chain_id"`
	CollectionID         uint64    `gorm:"not null;index:ux_nft_tokens_collection_token,unique,priority:1" json:"collection_id"`
	TokenID              string    `gorm:"column:token_id;type:numeric(78,0);not null;index:ux_nft_tokens_collection_token,unique,priority:2" json:"token_id"`
	OwnerAccountID       uint64    `gorm:"not null" json:"owner_account_id"`
	MintTxHash           *string   `gorm:"size:66" json:"mint_tx_hash,omitempty"`
	MintBlockNumber      *int64    `json:"mint_block_number,omitempty"`
	LastTransferTxHash   *string   `gorm:"size:66" json:"last_transfer_tx_hash,omitempty"`
	LastTransferBlock    *int64    `json:"last_transfer_block,omitempty"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (NFTToken) TableName() string { return "nft_tokens" }

// NFTTransfer 对应 nft_transfers。
type NFTTransfer struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID         int64     `gorm:"not null;index:ux_nft_transfers_chain_tx_log,unique,priority:1" json:"chain_id"`
	CollectionID    uint64    `gorm:"not null" json:"collection_id"`
	TokenID         string    `gorm:"column:token_id;type:numeric(78,0);not null" json:"token_id"`
	FromAccountID   *uint64   `json:"from_account_id,omitempty"`
	ToAccountID     uint64    `gorm:"not null" json:"to_account_id"`
	BlockNumber     int64     `gorm:"not null" json:"block_number"`
	BlockTime       time.Time `gorm:"not null" json:"block_time"`
	TxHash          string    `gorm:"size:66;not null;index:ux_nft_transfers_chain_tx_log,unique,priority:2" json:"tx_hash"`
	LogIndex        int       `gorm:"not null;index:ux_nft_transfers_chain_tx_log,unique,priority:3" json:"log_index"`
	CreatedAt       time.Time `json:"created_at"`
}

func (NFTTransfer) TableName() string { return "nft_transfers" }

// NFTFactoryEvent 对应 nft_factory_events。
type NFTFactoryEvent struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID             int64     `gorm:"not null;index:ux_nft_factory_events_chain_tx_log,unique,priority:1" json:"chain_id"`
	FactoryContractID   uint64    `gorm:"not null" json:"factory_contract_id"`
	EventType           string    `gorm:"size:48;not null" json:"event_type"`
	BlockNumber         int64     `gorm:"not null" json:"block_number"`
	BlockTime           time.Time `gorm:"not null" json:"block_time"`
	TxHash              string    `gorm:"size:66;not null;index:ux_nft_factory_events_chain_tx_log,unique,priority:2" json:"tx_hash"`
	LogIndex            int       `gorm:"not null;index:ux_nft_factory_events_chain_tx_log,unique,priority:3" json:"log_index"`
	PayloadJSON         []byte    `gorm:"type:jsonb;not null;default:'{}'" json:"payload_json"`
	CreatedAt           time.Time `json:"created_at"`
}

func (NFTFactoryEvent) TableName() string { return "nft_factory_events" }

// NFTMarketplaceAdminEvent 对应 nft_marketplace_admin_events。
type NFTMarketplaceAdminEvent struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID                 int64     `gorm:"not null;index:ux_nft_marketplace_admin_events_chain_tx_log,unique,priority:1" json:"chain_id"`
	MarketplaceContractID   uint64    `gorm:"not null" json:"marketplace_contract_id"`
	EventType               string    `gorm:"size:48;not null" json:"event_type"`
	BlockNumber             int64     `gorm:"not null" json:"block_number"`
	BlockTime               time.Time `gorm:"not null" json:"block_time"`
	TxHash                  string    `gorm:"size:66;not null;index:ux_nft_marketplace_admin_events_chain_tx_log,unique,priority:2" json:"tx_hash"`
	LogIndex                int       `gorm:"not null;index:ux_nft_marketplace_admin_events_chain_tx_log,unique,priority:3" json:"log_index"`
	PayloadJSON             []byte    `gorm:"type:jsonb;not null;default:'{}'" json:"payload_json"`
	CreatedAt               time.Time `json:"created_at"`
}

func (NFTMarketplaceAdminEvent) TableName() string { return "nft_marketplace_admin_events" }

// NFTMarketTradeEvent 对应 nft_market_trade_events。
type NFTMarketTradeEvent struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID                 int64     `gorm:"not null;index:ux_nft_market_trade_events_chain_tx_log,unique,priority:1" json:"chain_id"`
	MarketplaceContractID   uint64    `gorm:"not null" json:"marketplace_contract_id"`
	EventType               string    `gorm:"size:32;not null" json:"event_type"`
	CollectionAddress       string    `gorm:"size:42;not null" json:"collection_address"`
	TokenID                 string    `gorm:"column:token_id;type:numeric(78,0);not null" json:"token_id"`
	SellerAccountID         *uint64   `json:"seller_account_id,omitempty"`
	BuyerAccountID          *uint64   `json:"buyer_account_id,omitempty"`
	PriceWei                *string   `gorm:"column:price_wei;type:numeric(78,0)" json:"price_wei,omitempty"`
	OldPriceWei             *string   `gorm:"column:old_price_wei;type:numeric(78,0)" json:"old_price_wei,omitempty"`
	NewPriceWei             *string   `gorm:"column:new_price_wei;type:numeric(78,0)" json:"new_price_wei,omitempty"`
	PlatformFeeWei          *string   `gorm:"column:platform_fee_wei;type:numeric(78,0)" json:"platform_fee_wei,omitempty"`
	RoyaltyAmountWei        *string   `gorm:"column:royalty_amount_wei;type:numeric(78,0)" json:"royalty_amount_wei,omitempty"`
	FeeBpsSnapshot          *string   `gorm:"column:fee_bps_snapshot;type:numeric(78,0)" json:"fee_bps_snapshot,omitempty"`
	BlockNumber             int64     `gorm:"not null" json:"block_number"`
	BlockTime               time.Time `gorm:"not null" json:"block_time"`
	TxHash                  string    `gorm:"size:66;not null;index:ux_nft_market_trade_events_chain_tx_log,unique,priority:2" json:"tx_hash"`
	LogIndex                int       `gorm:"not null;index:ux_nft_market_trade_events_chain_tx_log,unique,priority:3" json:"log_index"`
	CreatedAt               time.Time `json:"created_at"`
}

func (NFTMarketTradeEvent) TableName() string { return "nft_market_trade_events" }

// NFTActiveListing 对应 nft_active_listings。
type NFTActiveListing struct {
	ID                      uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID                 int64      `gorm:"not null" json:"chain_id"`
	MarketplaceContractID   uint64     `gorm:"not null" json:"marketplace_contract_id"`
	CollectionAddress       string     `gorm:"size:42;not null" json:"collection_address"`
	TokenID                 string     `gorm:"column:token_id;type:numeric(78,0);not null" json:"token_id"`
	SellerAccountID         uint64     `gorm:"not null" json:"seller_account_id"`
	PriceWei                string     `gorm:"column:price_wei;type:numeric(78,0);not null" json:"price_wei"`
	ListedBlockNumber       int64      `gorm:"not null" json:"listed_block_number"`
	ListedTxHash            string     `gorm:"size:66;not null" json:"listed_tx_hash"`
	ListingStatus           string     `gorm:"size:16;not null;default:active" json:"listing_status"`
	ClosedAt                *time.Time `json:"closed_at,omitempty"`
	CloseTxHash             *string    `gorm:"size:66" json:"close_tx_hash,omitempty"`
	CloseLogIndex           *int       `json:"close_log_index,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

func (NFTActiveListing) TableName() string { return "nft_active_listings" }

// NFTCollectionEvent 对应 nft_collection_events。
type NFTCollectionEvent struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainID      int64     `gorm:"not null;index:ux_nft_collection_events_chain_tx_log,unique,priority:1" json:"chain_id"`
	CollectionID uint64    `gorm:"not null" json:"collection_id"`
	EventType    string    `gorm:"size:48;not null" json:"event_type"`
	BlockNumber  int64     `gorm:"not null" json:"block_number"`
	BlockTime    time.Time `gorm:"not null" json:"block_time"`
	TxHash       string    `gorm:"size:66;not null;index:ux_nft_collection_events_chain_tx_log,unique,priority:2" json:"tx_hash"`
	LogIndex     int       `gorm:"not null;index:ux_nft_collection_events_chain_tx_log,unique,priority:3" json:"log_index"`
	PayloadJSON  []byte    `gorm:"type:jsonb;not null;default:'{}'" json:"payload_json"`
	CreatedAt    time.Time `json:"created_at"`
}

func (NFTCollectionEvent) TableName() string { return "nft_collection_events" }
