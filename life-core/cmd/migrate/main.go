package main

import (
	"context"
	"fmt"
	"log"

	mysqldriver "github.com/go-sql-driver/mysql"

	"life-core/config"
	"life-core/infrastructure/database"
	"life-core/migration"
)

func run() (*migration.Report, error) {
	if err := config.Startup(); err != nil {
		return nil, err
	}
	if err := config.Get().Validate(); err != nil {
		return nil, err
	}
	if err := database.InitMySQL(); err != nil {
		return nil, err
	}
	defer func() {
		if err := database.Shutdown(context.Background()); err != nil {
			log.Printf("关闭数据库失败: %v", err)
		}
	}()
	ctx := context.Background()
	dsnConfig, err := mysqldriver.ParseDSN(config.Get().Database.MySQL.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse configured MySQL DSN: %w", err)
	}
	if dsnConfig.DBName == "" {
		return nil, fmt.Errorf("configured MySQL DSN must name a database")
	}
	var selectedDatabase string
	if err := database.MySQL().WithContext(ctx).Raw("SELECT DATABASE()").Scan(&selectedDatabase).Error; err != nil {
		return nil, fmt.Errorf("check selected database: %w", err)
	}
	if selectedDatabase != dsnConfig.DBName {
		return nil, fmt.Errorf("refusing migration: DSN selects %q, expected %q", selectedDatabase, dsnConfig.DBName)
	}
	if err := migration.Apply(ctx, database.MySQL()); err != nil {
		return nil, err
	}
	return migration.Verify(ctx, database.MySQL(), dsnConfig.DBName)
}

func main() {
	report, err := run()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("数据库迁移完成并通过校验: database=%s collation=%s tables=%d foreign_keys=%d",
		report.Database, report.Collation, report.TableCount, report.ForeignKeyNum)
}
