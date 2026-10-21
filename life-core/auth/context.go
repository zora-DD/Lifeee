package auth

import (
	"context"

	"life-core/model"
)

type userContextKey struct{}

// WithUser 把已通过 Token 校验的用户写入请求上下文。
func WithUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// UserFromContext 获取当前登录用户。
func UserFromContext(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(*model.User)
	return user, ok && user != nil
}

// UserIDFromContext 获取当前登录用户 ID。
func UserIDFromContext(ctx context.Context) (uint64, bool) {
	user, ok := UserFromContext(ctx)
	if !ok || user.ID == 0 {
		return 0, false
	}
	return user.ID, true
}

// UserIDOrDefault 兼容不经过 HTTP 中间件的单元测试和内部调用。
func UserIDOrDefault(ctx context.Context, fallback uint64) uint64 {
	if id, ok := UserIDFromContext(ctx); ok {
		return id
	}
	return fallback
}
