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
		&models.Example{},
		&models.BankDeposit{},
		&models.BankWithdrawal{},
		&models.ChainIndexerCursor{},
	); err != nil {
		return nil, err
	}
	return db, nil
}
