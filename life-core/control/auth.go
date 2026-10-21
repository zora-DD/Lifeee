package control

import (
	"context"
	"fmt"

	"github.com/vaynedu/hollow/pkg/hecode"

	"life-core/auth"
	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
	"life-core/third_party/wechat"
)

func (c *Control) WechatLogin(ctx context.Context, req *proto.WechatLoginRequest) (*proto.WechatLoginResponse, error) {
	svc, err := newAuthService()
	if err != nil {
		return nil, err
	}
	return svc.Login(ctx, req)
}

func (c *Control) GetCurrentUser(ctx context.Context, _ *proto.GetCurrentUserRequest) (*proto.UserItem, error) {
	return service.CurrentUser(ctx)
}

func newAuthService() (*service.AuthService, error) {
	cfg := config.Get()
	if cfg == nil || !cfg.Auth.Enabled {
		return nil, fmt.Errorf("%w: auth is not enabled", hecode.ErrConfig)
	}
	tokens, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", hecode.ErrConfig, err)
	}
	var exchanger service.WechatCodeExchanger
	if cfg.WeChat.MockEnabled {
		exchanger = &wechat.MockClient{OpenID: cfg.WeChat.MockOpenID}
	} else {
		exchanger = wechat.NewClient(cfg.WeChat.AppID, cfg.WeChat.AppSecret)
	}
	return service.NewAuthService(
		database.MySQL(), dao.NewUserDao(), exchanger, tokens, cfg.Auth.BootstrapFirstUser,
	), nil
}
