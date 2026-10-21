package service

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hecode"

	authctx "life-core/auth"
	"life-core/model"
)

// requireOwnerOrAdmin 限制普通用户只能修改自己创建的数据，管理员可管理全部数据。
func requireOwnerOrAdmin(ctx context.Context, ownerID uint64) error {
	user, ok := authctx.UserFromContext(ctx)
	if !ok {
		return hecode.ErrUnauthorized
	}
	if user.Role == model.UserRoleAdmin || user.ID == ownerID {
		return nil
	}
	return hecode.ErrPermissionDenied
}
