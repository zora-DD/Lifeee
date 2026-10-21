//go:build integration

package dao

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestContentQueriesWithMySQL 验证跨表 UNION 查询可被真实 MySQL 执行。
func TestContentQueriesWithMySQL(t *testing.T) {
	dsn := os.Getenv("LIFE_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置 LIFE_TEST_DSN")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var userID uint64
	if err := db.Raw("SELECT id FROM users ORDER BY id LIMIT 1").Scan(&userID).Error; err != nil {
		t.Fatal(err)
	}
	if userID == 0 {
		t.Skip("数据库中暂无用户")
	}
	repository := NewContentDao()
	if _, _, err := repository.ListContributions(context.Background(), db, userID, "", 1, 20); err != nil {
		t.Fatalf("查询我的贡献失败: %v", err)
	}
	if _, _, err := repository.ListFavorites(context.Background(), db, userID, "", 1, 20); err != nil {
		t.Fatalf("查询收藏失败: %v", err)
	}
}
