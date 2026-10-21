package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	"life-core/auth"
	"life-core/model"
	"life-core/proto"
)

// WechatSession 是微信登录凭证换取后的用户标识。
type WechatSession struct {
	OpenID  string
	UnionID string
}

// WechatCodeExchanger 定义微信登录凭证交换能力，由 third_party/wechat 实现。
type WechatCodeExchanger interface {
	Exchange(context.Context, string) (*WechatSession, error)
}

// UserRepository 定义登录流程依赖的用户数据访问能力。
type UserRepository interface {
	Create(context.Context, *gorm.DB, *model.User) error
	GetByID(context.Context, *gorm.DB, uint64) (*model.User, error)
	GetByOpenID(context.Context, *gorm.DB, string) (*model.User, error)
	Count(context.Context, *gorm.DB) (int64, error)
	UpdateProfile(context.Context, *gorm.DB, *model.User) error
	UpdateLastLoginAt(context.Context, *gorm.DB, uint64, time.Time) error
}

// AuthService 负责微信登录、用户初始化和 Token 签发。
type AuthService struct {
	db                 *gorm.DB
	users              UserRepository
	exchanger          WechatCodeExchanger
	tokens             *auth.TokenManager
	bootstrapFirstUser bool
}

func NewAuthService(
	db *gorm.DB,
	users UserRepository,
	exchanger WechatCodeExchanger,
	tokens *auth.TokenManager,
	bootstrapFirstUser bool,
) *AuthService {
	return &AuthService{
		db: db, users: users, exchanger: exchanger, tokens: tokens,
		bootstrapFirstUser: bootstrapFirstUser,
	}
}

func (s *AuthService) Login(ctx context.Context, req *proto.WechatLoginRequest) (*proto.WechatLoginResponse, error) {
	if req == nil || strings.TrimSpace(req.GetCode()) == "" {
		return nil, fmt.Errorf("%w: code is required", hecode.ErrInvalidParam)
	}
	if s == nil || s.db == nil || s.users == nil || s.exchanger == nil || s.tokens == nil {
		return nil, fmt.Errorf("%w: auth service dependencies are not initialized", hecode.ErrService)
	}
	session, err := s.exchanger.Exchange(ctx, req.GetCode())
	if err != nil {
		return nil, fmt.Errorf("%w: exchange wechat code: %v", hecode.ErrUnauthorized, err)
	}
	user, err := s.users.GetByOpenID(ctx, s.db, session.OpenID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user, err = s.createUser(ctx, session)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get or create user: %v", hecode.ErrDBQuery, err)
	}
	if user.Status == model.UserStatusDisabled {
		return nil, fmt.Errorf("%w: user is disabled", hecode.ErrForbidden)
	}
	if user.Nickname == legacyDefaultNickname {
		applyGeneratedProfile(user, session.OpenID)
		if err := s.users.UpdateProfile(ctx, s.db, user); err != nil {
			return nil, fmt.Errorf("%w: upgrade generated user profile: %v", hecode.ErrDBUpdate, err)
		}
	}
	now := time.Now()
	if err := s.users.UpdateLastLoginAt(ctx, s.db, user.ID, now); err != nil {
		return nil, fmt.Errorf("%w: update last login: %v", hecode.ErrDBUpdate, err)
	}
	user.LastLoginAt = &now
	token, expiresAt, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: issue token: %v", hecode.ErrService, err)
	}
	return &proto.WechatLoginResponse{
		AccessToken: token, ExpiresAt: expiresAt.Format(time.RFC3339), User: userToProto(user),
	}, nil
}

func (s *AuthService) createUser(ctx context.Context, session *WechatSession) (*model.User, error) {
	count, err := s.users.Count(ctx, s.db)
	if err != nil {
		return nil, err
	}
	status := model.UserStatusPending
	role := model.UserRoleUser
	if s.bootstrapFirstUser && count == 0 {
		status = model.UserStatusApproved
		role = model.UserRoleAdmin
	}
	user := &model.User{
		WxOpenID: session.OpenID,
		Status:   status,
		Role:     role,
	}
	applyGeneratedProfile(user, session.OpenID)
	if session.UnionID != "" {
		user.WxUnionID = optionalString(session.UnionID)
	}
	if err := s.users.Create(ctx, s.db, user); err != nil {
		// 并发登录撞唯一键时返回已经创建完成的用户。
		existing, getErr := s.users.GetByOpenID(ctx, s.db, session.OpenID)
		if getErr == nil {
			return existing, nil
		}
		return nil, err
	}
	return user, nil
}

// CurrentUser 返回鉴权中间件写入上下文的当前用户。
func CurrentUser(ctx context.Context) (*proto.UserItem, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	return userToProto(user), nil
}

func userToProto(user *model.User) *proto.UserItem {
	if user == nil {
		return nil
	}
	item := &proto.UserItem{
		Id: strconvFormatUint(user.ID), Nickname: user.Nickname,
		AccessStatus: user.Status, Role: user.Role, CreatedAt: formatTime(user.CreatedAt),
	}
	if user.AvatarURL != nil {
		item.AvatarUrl = *user.AvatarURL
	}
	if user.AvatarEmoji != nil {
		item.AvatarEmoji = *user.AvatarEmoji
	}
	if user.AvatarColor != nil {
		item.AvatarColor = *user.AvatarColor
	}
	if user.Signature != nil {
		item.Signature = *user.Signature
	}
	if user.ApplicationNote != nil {
		item.ApplicationNote = *user.ApplicationNote
	}
	return item
}

func strconvFormatUint(value uint64) string {
	return fmt.Sprintf("%d", value)
}
