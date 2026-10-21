//go:build integration

package service

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

// TestFavoriteFlowWithMySQL 在事务中验证收藏、状态、列表和取消收藏全链路。
func TestFavoriteFlowWithMySQL(t *testing.T) {
	dsn := os.Getenv("LIFE_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置 LIFE_TEST_DSN")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()

	const userID = uint64(900000001)
	task := &model.Task{
		Title: "收藏集成测试", Status: model.TaskStatusTodo, Priority: model.TaskPriorityMedium,
		Category: "测试", CreatedBy: userID,
	}
	if err := tx.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	ctx := auth.WithUser(context.Background(), &model.User{ID: userID})
	svc := NewFavoriteService(tx, dao.NewFavoriteDao(), dao.NewContentDao())
	targetID := strconvFormatUint(task.ID)

	modified, err := svc.Modify(ctx, &proto.ModifyFavoriteRequest{
		TargetType: model.FavoriteTargetTask, TargetId: targetID, Favorited: true,
	})
	if err != nil || !modified.Favorited {
		t.Fatalf("收藏失败: response=%v err=%v", modified, err)
	}
	status, err := svc.Status(ctx, &proto.GetFavoriteStatusRequest{
		TargetType: model.FavoriteTargetTask, TargetId: targetID,
	})
	if err != nil || !status.Favorited {
		t.Fatalf("收藏状态错误: response=%v err=%v", status, err)
	}
	list, err := svc.List(ctx, &proto.ListFavoritesRequest{TargetType: model.FavoriteTargetTask, Page: 1, PageSize: 20})
	if err != nil || len(list.Items) != 1 || list.Items[0].Target.TargetId != targetID {
		t.Fatalf("收藏列表错误: response=%v err=%v", list, err)
	}
	modified, err = svc.Modify(ctx, &proto.ModifyFavoriteRequest{
		TargetType: model.FavoriteTargetTask, TargetId: targetID, Favorited: false,
	})
	if err != nil || modified.Favorited {
		t.Fatalf("取消收藏失败: response=%v err=%v", modified, err)
	}
}
