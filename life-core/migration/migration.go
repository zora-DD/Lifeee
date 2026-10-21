// Package migration 提供项目数据库初始化能力。
package migration

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

//go:embed 001_init.sql
var initSQL string

var expectedColumns = map[string]int64{
	"users":                   14,
	"tasks":                   11,
	"activity_logs":           10,
	"bills":                   13,
	"restaurants":             22,
	"restaurant_consumptions": 10,
	"home_dishes":             17,
	"home_cooking_records":    12,
	"snacks":                  16,
	"snack_tasting_records":   8,
	"places":                  22,
	"favorites":               5,
	"media_assets":            12,
	"memories":                11,
	"memory_photos":           10,
	"task_ai_summaries":       8,
	"user_module_permissions": 7,
	"daily_routines":          11,
	"daily_checkins":          8,
}

const addRestaurantFeaturesSQL = `
	ALTER TABLE restaurants
	ADD COLUMN features VARCHAR(500) NULL COMMENT '店铺特点' AFTER business_hours
`

const addRestaurantLatitudeSQL = `
	ALTER TABLE restaurants
	ADD COLUMN latitude DECIMAL(10,7) NULL COMMENT '纬度' AFTER address
`

const addRestaurantLongitudeSQL = `
	ALTER TABLE restaurants
	ADD COLUMN longitude DECIMAL(10,7) NULL COMMENT '经度' AFTER latitude
`

const addPlaceGuideURLSQL = `
	ALTER TABLE places
	ADD COLUMN guide_url VARCHAR(512) NULL COMMENT '微信公众号攻略文章地址' AFTER best_time
`

const addPlaceDistanceSQL = `
	ALTER TABLE places
	ADD COLUMN distance_km INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '以上海为起点的参考距离（公里）' AFTER best_time
`

const addPlaceDriveDurationSQL = `
	ALTER TABLE places
	ADD COLUMN drive_duration_minutes INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '参考自驾时长（分钟）' AFTER distance_km
`

var dailyRoutineScheduleColumns = []struct {
	name      string
	statement string
}{
	{name: "repeat_unit", statement: "ALTER TABLE daily_routines ADD COLUMN repeat_unit VARCHAR(16) NOT NULL DEFAULT 'day' COMMENT '重复单位：day/week/month' AFTER time_of_day"},
	{name: "repeat_interval", statement: "ALTER TABLE daily_routines ADD COLUMN repeat_interval INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '重复间隔' AFTER repeat_unit"},
	{name: "start_date", statement: "ALTER TABLE daily_routines ADD COLUMN start_date DATE NULL COMMENT '重复周期起始日期' AFTER repeat_interval"},
}

const addHomeDishStepsSQL = `ALTER TABLE home_dishes ADD COLUMN steps TEXT NOT NULL COMMENT '制作步骤' AFTER ingredients`
const addUserApplicationNoteSQL = `ALTER TABLE users ADD COLUMN application_note VARCHAR(200) NULL COMMENT '访问申请身份备注' AFTER signature`

// Apply 执行可重复运行的初始化 SQL，不包含 DROP 或数据删除操作。
func Apply(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migration.Apply: nil db")
	}
	for index, statement := range strings.Split(initSQL, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("migration.Apply: statement %d: %w", index+1, err)
		}
	}
	for _, column := range dailyRoutineScheduleColumns {
		var columnCount int64
		if err := db.WithContext(ctx).Raw(`
			SELECT COUNT(*)
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
			  AND table_name = 'daily_routines'
			  AND column_name = ?
		`, column.name).Scan(&columnCount).Error; err != nil {
			return fmt.Errorf("migration.Apply: check daily_routines.%s: %w", column.name, err)
		}
		if columnCount == 0 {
			if err := db.WithContext(ctx).Exec(column.statement).Error; err != nil {
				return fmt.Errorf("migration.Apply: add daily_routines.%s: %w", column.name, err)
			}
		}
	}
	var applicationNoteCount int64
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='users' AND column_name='application_note'`).Scan(&applicationNoteCount).Error; err != nil {
		return fmt.Errorf("migration.Apply: check users.application_note: %w", err)
	}
	if applicationNoteCount == 0 {
		if err := db.WithContext(ctx).Exec(addUserApplicationNoteSQL).Error; err != nil {
			return fmt.Errorf("migration.Apply: add users.application_note: %w", err)
		}
	}
	var homeDishStepsCount int64
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='home_dishes' AND column_name='steps'`).Scan(&homeDishStepsCount).Error; err != nil {
		return fmt.Errorf("migration.Apply: check home_dishes.steps: %w", err)
	}
	if homeDishStepsCount == 0 {
		if err := db.WithContext(ctx).Exec(addHomeDishStepsSQL).Error; err != nil {
			return fmt.Errorf("migration.Apply: add home_dishes.steps: %w", err)
		}
	}
	var featureColumnCount int64
	if err := db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'restaurants'
		  AND column_name = 'features'
	`).Scan(&featureColumnCount).Error; err != nil {
		return fmt.Errorf("migration.Apply: check restaurants.features: %w", err)
	}
	if featureColumnCount == 0 {
		if err := db.WithContext(ctx).Exec(addRestaurantFeaturesSQL).Error; err != nil {
			return fmt.Errorf("migration.Apply: add restaurants.features: %w", err)
		}
	}
	restaurantLocationColumns := []struct {
		name      string
		statement string
	}{
		{name: "latitude", statement: addRestaurantLatitudeSQL},
		{name: "longitude", statement: addRestaurantLongitudeSQL},
	}
	for _, column := range restaurantLocationColumns {
		var columnCount int64
		if err := db.WithContext(ctx).Raw(`
			SELECT COUNT(*)
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
			  AND table_name = 'restaurants'
			  AND column_name = ?
		`, column.name).Scan(&columnCount).Error; err != nil {
			return fmt.Errorf("migration.Apply: check restaurants.%s: %w", column.name, err)
		}
		if columnCount == 0 {
			if err := db.WithContext(ctx).Exec(column.statement).Error; err != nil {
				return fmt.Errorf("migration.Apply: add restaurants.%s: %w", column.name, err)
			}
		}
	}
	var guideURLColumnCount int64
	if err := db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'places'
		  AND column_name = 'guide_url'
	`).Scan(&guideURLColumnCount).Error; err != nil {
		return fmt.Errorf("migration.Apply: check places.guide_url: %w", err)
	}
	if guideURLColumnCount == 0 {
		if err := db.WithContext(ctx).Exec(addPlaceGuideURLSQL).Error; err != nil {
			return fmt.Errorf("migration.Apply: add places.guide_url: %w", err)
		}
	}
	placeTravelColumns := []struct {
		name      string
		statement string
	}{
		{name: "distance_km", statement: addPlaceDistanceSQL},
		{name: "drive_duration_minutes", statement: addPlaceDriveDurationSQL},
	}
	for _, column := range placeTravelColumns {
		var columnCount int64
		if err := db.WithContext(ctx).Raw(`
			SELECT COUNT(*)
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
			  AND table_name = 'places'
			  AND column_name = ?
		`, column.name).Scan(&columnCount).Error; err != nil {
			return fmt.Errorf("migration.Apply: check places.%s: %w", column.name, err)
		}
		if columnCount == 0 {
			if err := db.WithContext(ctx).Exec(column.statement).Error; err != nil {
				return fmt.Errorf("migration.Apply: add places.%s: %w", column.name, err)
			}
		}
	}
	return nil
}

// Report 是建表后的数据库结构检查结果。
type Report struct {
	Database      string
	Collation     string
	TableCount    int64
	ForeignKeyNum int64
}

// Verify 检查目标库、业务表字段数、排序规则及外键数量。
func Verify(ctx context.Context, db *gorm.DB, expectedDatabase string) (*Report, error) {
	if db == nil {
		return nil, fmt.Errorf("migration.Verify: nil db")
	}
	if strings.TrimSpace(expectedDatabase) == "" {
		return nil, fmt.Errorf("migration.Verify: expected database is required")
	}
	report := new(Report)
	if err := db.WithContext(ctx).Raw("SELECT DATABASE()").Scan(&report.Database).Error; err != nil {
		return nil, fmt.Errorf("migration.Verify: query database: %w", err)
	}
	if report.Database != expectedDatabase {
		return nil, fmt.Errorf("migration.Verify: expected database %q, got %q", expectedDatabase, report.Database)
	}
	if err := db.WithContext(ctx).Raw(`
		SELECT default_collation_name
		FROM information_schema.schemata
		WHERE schema_name = DATABASE()
	`).Scan(&report.Collation).Error; err != nil {
		return nil, fmt.Errorf("migration.Verify: query collation: %w", err)
	}
	if report.Collation != "utf8mb4_general_ci" && report.Collation != "utf8mb4_0900_ai_ci" {
		return nil, fmt.Errorf("migration.Verify: unexpected database collation %q", report.Collation)
	}

	tableNames := make([]string, 0, len(expectedColumns))
	for tableName := range expectedColumns {
		tableNames = append(tableNames, tableName)
	}
	type tableInfo struct {
		TableName      string
		TableCollation string
		ColumnCount    int64
	}
	var tables []tableInfo
	rows, err := db.WithContext(ctx).Raw(`
		SELECT t.table_name, t.table_collation, COUNT(c.column_name) AS column_count
		FROM information_schema.tables t
		JOIN information_schema.columns c
		  ON c.table_schema = t.table_schema AND c.table_name = t.table_name
		WHERE t.table_schema = DATABASE() AND t.table_name IN ?
		GROUP BY t.table_name, t.table_collation
	`, tableNames).Rows()
	if err != nil {
		return nil, fmt.Errorf("migration.Verify: query tables: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var table tableInfo
		if err := rows.Scan(&table.TableName, &table.TableCollation, &table.ColumnCount); err != nil {
			return nil, fmt.Errorf("migration.Verify: scan table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migration.Verify: iterate tables: %w", err)
	}
	report.TableCount = int64(len(tables))
	if report.TableCount != int64(len(expectedColumns)) {
		return nil, fmt.Errorf("migration.Verify: expected %d tables, got %d", len(expectedColumns), report.TableCount)
	}
	for _, table := range tables {
		if table.TableCollation != "utf8mb4_general_ci" {
			return nil, fmt.Errorf("migration.Verify: table %s has collation %s", table.TableName, table.TableCollation)
		}
		if table.ColumnCount != expectedColumns[table.TableName] {
			return nil, fmt.Errorf("migration.Verify: table %s expected %d columns, got %d", table.TableName, expectedColumns[table.TableName], table.ColumnCount)
		}
	}
	if err := db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM information_schema.referential_constraints
		WHERE constraint_schema = DATABASE() AND table_name IN ?
	`, tableNames).Scan(&report.ForeignKeyNum).Error; err != nil {
		return nil, fmt.Errorf("migration.Verify: query foreign keys: %w", err)
	}
	if report.ForeignKeyNum != 0 {
		return nil, fmt.Errorf("migration.Verify: expected no foreign keys, got %d", report.ForeignKeyNum)
	}
	return report, nil
}
