package database

import (
	"fmt"
	"strings"

	"go-chain/backend/migrations"

	"gorm.io/gorm"
)

// ApplyLending006Migration 执行 006_lending.sql（CREATE / INDEX / COMMENT / INSERT 种子）。
// 依赖 GORM AutoMigrate 已创建 chain_indexer_cursors（见 models.ChainIndexerCursor）。
func ApplyLending006Migration(db *gorm.DB) error {
	sql := strings.TrimSpace(migrations.SQL006Lending)
	if sql == "" {
		return fmt.Errorf("embedded 006_lending.sql is empty")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	stmts := splitSQLStatements(sql)
	for i, stmt := range stmts {
		if _, err := sqlDB.Exec(stmt); err != nil {
			head := stmt
			if len(head) > 120 {
				head = head[:120] + "…"
			}
			return fmt.Errorf("006_lending statement #%d: %w\n-- %s", i+1, err, head)
		}
	}
	return nil
}
