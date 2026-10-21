package service

import (
	"context"
	"errors"
	"testing"

	"github.com/vaynedu/hollow/pkg/hecode"

	authctx "life-core/auth"
	"life-core/model"
)

func TestRequireOwnerOrAdmin(t *testing.T) {
	owner := authctx.WithUser(context.Background(), &model.User{ID: 7, Role: model.UserRoleUser})
	if err := requireOwnerOrAdmin(owner, 7); err != nil {
		t.Fatalf("创建人应可修改: %v", err)
	}
	if err := requireOwnerOrAdmin(owner, 8); !errors.Is(err, hecode.ErrPermissionDenied) {
		t.Fatalf("普通用户不应修改他人数据: %v", err)
	}
	if err := requireOwnerOrAdmin(adminContext(), 8); err != nil {
		t.Fatalf("管理员应可修改全部数据: %v", err)
	}
}
