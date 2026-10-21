package service

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	"life-core/auth"
	"life-core/model"
	"life-core/proto"
)

type fakeAuthUsers struct {
	user      *model.User
	count     int64
	created   *model.User
	updated   *model.User
	lastLogin time.Time
}

func (f *fakeAuthUsers) Create(_ context.Context, _ *gorm.DB, row *model.User) error {
	row.ID = 1
	row.CreatedAt = time.Now()
	f.created = row
	f.user = row
	return nil
}

func (f *fakeAuthUsers) GetByID(_ context.Context, _ *gorm.DB, id uint64) (*model.User, error) {
	if f.user == nil || f.user.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return f.user, nil
}

func (f *fakeAuthUsers) GetByOpenID(_ context.Context, _ *gorm.DB, openID string) (*model.User, error) {
	if f.user == nil || f.user.WxOpenID != openID {
		return nil, gorm.ErrRecordNotFound
	}
	return f.user, nil
}

func (f *fakeAuthUsers) Count(context.Context, *gorm.DB) (int64, error) {
	return f.count, nil
}

func (f *fakeAuthUsers) UpdateProfile(_ context.Context, _ *gorm.DB, row *model.User) error {
	copyRow := *row
	f.updated = &copyRow
	return nil
}

func (f *fakeAuthUsers) UpdateLastLoginAt(_ context.Context, _ *gorm.DB, _ uint64, at time.Time) error {
	f.lastLogin = at
	return nil
}

type fakeCodeExchanger struct {
	session *WechatSession
	err     error
}

func (f *fakeCodeExchanger) Exchange(context.Context, string) (*WechatSession, error) {
	return f.session, f.err
}

func newTestTokenManager() *auth.TokenManager {
	manager, _ := auth.NewTokenManager("unit-test-jwt-secret-with-at-least-32-characters", time.Hour)
	return manager
}

func TestAuthServiceLogin(t *testing.T) {
	Convey("首个登录用户自动成为已通过管理员", t, func() {
		users := new(fakeAuthUsers)
		exchanger := &fakeCodeExchanger{session: &WechatSession{OpenID: "owner-open-id"}}
		svc := NewAuthService(&gorm.DB{}, users, exchanger, newTestTokenManager(), true)

		resp, err := svc.Login(context.Background(), &proto.WechatLoginRequest{Code: "wx-code"})

		So(err, ShouldBeNil)
		So(resp.GetAccessToken(), ShouldNotBeEmpty)
		So(resp.GetUser().GetAccessStatus(), ShouldEqual, model.UserStatusApproved)
		So(resp.GetUser().GetRole(), ShouldEqual, model.UserRoleAdmin)
		So(users.created, ShouldNotBeNil)
		So(users.created.Nickname, ShouldNotEqual, legacyDefaultNickname)
		So(users.created.AvatarEmoji, ShouldNotBeNil)
		So(users.created.AvatarColor, ShouldNotBeNil)
		So(users.created.Signature, ShouldNotBeNil)
		So(users.lastLogin.IsZero(), ShouldBeFalse)
	})

	Convey("已有默认资料用户登录时自动升级并保持身份状态", t, func() {
		users := &fakeAuthUsers{user: &model.User{
			ID: 8, WxOpenID: "legacy-open-id", Nickname: legacyDefaultNickname,
			Status: model.UserStatusApproved, Role: model.UserRoleUser,
		}}
		exchanger := &fakeCodeExchanger{session: &WechatSession{OpenID: "legacy-open-id"}}
		svc := NewAuthService(&gorm.DB{}, users, exchanger, newTestTokenManager(), true)

		resp, err := svc.Login(context.Background(), &proto.WechatLoginRequest{Code: "wx-code"})

		So(err, ShouldBeNil)
		So(users.updated, ShouldNotBeNil)
		So(resp.GetUser().GetNickname(), ShouldNotEqual, legacyDefaultNickname)
		So(resp.GetUser().GetAccessStatus(), ShouldEqual, model.UserStatusApproved)
	})

	Convey("已有待审核用户仍可登录并获得查询自身状态的 Token", t, func() {
		users := &fakeAuthUsers{user: &model.User{
			ID: 7, WxOpenID: "pending-open-id", Nickname: "体验官",
			Status: model.UserStatusPending, Role: model.UserRoleUser,
		}}
		exchanger := &fakeCodeExchanger{session: &WechatSession{OpenID: "pending-open-id"}}
		svc := NewAuthService(&gorm.DB{}, users, exchanger, newTestTokenManager(), true)

		resp, err := svc.Login(context.Background(), &proto.WechatLoginRequest{Code: "wx-code"})

		So(err, ShouldBeNil)
		So(resp.GetUser().GetAccessStatus(), ShouldEqual, model.UserStatusPending)
		userID, parseErr := newTestTokenManager().Parse(resp.GetAccessToken())
		So(parseErr, ShouldBeNil)
		So(userID, ShouldEqual, uint64(7))
		So(users.created, ShouldBeNil)
	})

	Convey("禁用用户无法登录", t, func() {
		users := &fakeAuthUsers{user: &model.User{
			ID: 9, WxOpenID: "disabled-open-id", Nickname: "体验官",
			Status: model.UserStatusDisabled, Role: model.UserRoleUser,
		}}
		exchanger := &fakeCodeExchanger{session: &WechatSession{OpenID: "disabled-open-id"}}
		svc := NewAuthService(&gorm.DB{}, users, exchanger, newTestTokenManager(), true)

		_, err := svc.Login(context.Background(), &proto.WechatLoginRequest{Code: "wx-code"})

		So(hecode.IsError(err, hecode.ErrForbidden), ShouldBeTrue)
	})

	Convey("微信 code 换取失败返回未授权", t, func() {
		svc := NewAuthService(
			&gorm.DB{}, new(fakeAuthUsers), &fakeCodeExchanger{err: errors.New("bad code")},
			newTestTokenManager(), true,
		)
		_, err := svc.Login(context.Background(), &proto.WechatLoginRequest{Code: "wx-code"})
		So(hecode.IsError(err, hecode.ErrUnauthorized), ShouldBeTrue)
	})
}
