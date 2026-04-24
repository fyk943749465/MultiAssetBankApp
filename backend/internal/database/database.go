package database

import (
	"fmt"

	"go-chain/backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is empty")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		// bank
		&models.Example{},
		&models.BankDeposit{},
		&models.BankWithdrawal{},
		&models.ChainIndexerCursor{},
		// code pulse
		&models.CPProposal{},
		&models.CPCampaign{},
		&models.CPContribution{},
		&models.CPProposalMilestone{},
		&models.CPCampaignMilestone{},
		&models.CPCampaignDeveloper{},
		&models.CPMilestoneClaim{},
		&models.CPEventLog{},
		&models.CPSyncCursor{},
		&models.CPSystemState{},
		&models.CPWalletProfile{},
		&models.CPWalletRole{},
		&models.CPTxAttempt{},
		&models.CPPlatformFundMovement{},
		&models.CPSnapshotDaily{},
	); err != nil {
		return nil, err
	}
	if err := ApplyLending006Migration(db); err != nil {
		return nil, err
	}
	if err := ApplyLending007Migration(db); err != nil {
		return nil, err
	}
	return db, nil
}
