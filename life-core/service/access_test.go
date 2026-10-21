package service

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type accessUserRepositoryStub struct {
	users         []model.User
	query         dao.UserListQuery
	total         int64
	updatedUserID uint64
	updatedStatus string
	updatedNote   string
}

func (s *accessUserRepositoryStub) GetByID(_ context.Context, _ *gorm.DB, id uint64) (*model.User, error) {
	for i := range s.users {
		if s.users[i].ID == id {
			row := s.users[i]
			return &row, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *accessUserRepositoryStub) List(_ context.Context, _ *gorm.DB, query dao.UserListQuery) ([]model.User, int64, error) {
	s.query = query
	return s.users, s.total, nil
}

func (s *accessUserRepositoryStub) UpdateStatus(_ context.Context, _ *gorm.DB, id uint64, status string) error {
	s.updatedUserID = id
	s.updatedStatus = status
	return nil
}

func (s *accessUserRepositoryStub) UpdateApplicationNote(_ context.Context, _ *gorm.DB, id uint64, note string) error {
	s.updatedUserID = id
	s.updatedNote = note
	for i := range s.users {
		if s.users[i].ID == id {
			s.users[i].ApplicationNote = optionalString(note)
		}
	}
	return nil
}

func adminContext() context.Context {
	return authctx.WithUser(context.Background(), &model.User{
		ID: 1, Status: model.UserStatusApproved, Role: model.UserRoleAdmin,
	})
}

func TestAccessServiceApply(t *testing.T) {
	Convey("待审批用户可提交身份备注", t, func() {
		repository := &accessUserRepositoryStub{users: []model.User{{
			ID: 8, Nickname: "随机昵称", Status: model.UserStatusPending, Role: model.UserRoleUser,
		}}}
		svc := NewAccessService(new(gorm.DB), repository)
		ctx := authctx.WithUser(context.Background(), &repository.users[0])

		response, err := svc.Apply(ctx, &proto.ApplyAccessRequest{Note: "我是小帅，点点的家人"})

		So(err, ShouldBeNil)
		So(repository.updatedUserID, ShouldEqual, uint64(8))
		So(repository.updatedNote, ShouldEqual, "我是小帅，点点的家人")
		So(response.ApplicationNote, ShouldEqual, "我是小帅，点点的家人")
	})

	Convey("身份备注不能为空或超过长度限制", t, func() {
		user := &model.User{ID: 8, Status: model.UserStatusPending, Role: model.UserRoleUser}
		svc := NewAccessService(new(gorm.DB), &accessUserRepositoryStub{users: []model.User{*user}})
		ctx := authctx.WithUser(context.Background(), user)

		_, err := svc.Apply(ctx, &proto.ApplyAccessRequest{Note: " "})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("被拒绝用户重新提交备注后恢复待审批", t, func() {
		user := &model.User{ID: 9, Status: model.UserStatusRejected, Role: model.UserRoleUser}
		repository := &accessUserRepositoryStub{users: []model.User{*user}}
		svc := NewAccessService(new(gorm.DB), repository)
		ctx := authctx.WithUser(context.Background(), user)

		_, err := svc.Apply(ctx, &proto.ApplyAccessRequest{Note: "我是家人，重新申请"})

		So(err, ShouldBeNil)
		So(repository.updatedStatus, ShouldEqual, model.UserStatusPending)
	})
}

func TestAccessServiceList(t *testing.T) {
	Convey("管理员默认查看待审批普通用户", t, func() {
		repository := &accessUserRepositoryStub{
			users: []model.User{{ID: 8, Nickname: "申请用户", Status: model.UserStatusPending, Role: model.UserRoleUser}},
			total: 21,
		}
		svc := NewAccessService(new(gorm.DB), repository)

		response, err := svc.List(adminContext(), &proto.ListAccessRequestsRequest{})

		So(err, ShouldBeNil)
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].Id, ShouldEqual, "8")
		So(response.HasMore, ShouldBeTrue)
		So(repository.query.Status, ShouldEqual, model.UserStatusPending)
		So(repository.query.Role, ShouldEqual, model.UserRoleUser)
	})

	Convey("普通用户不能读取审批列表", t, func() {
		ctx := authctx.WithUser(context.Background(), &model.User{
			ID: 2, Status: model.UserStatusApproved, Role: model.UserRoleUser,
		})
		_, err := NewAccessService(new(gorm.DB), &accessUserRepositoryStub{}).
			List(ctx, &proto.ListAccessRequestsRequest{})

		So(errors.Is(err, hecode.ErrPermissionDenied), ShouldBeTrue)
	})

	Convey("拒绝非法状态和分页", t, func() {
		svc := NewAccessService(new(gorm.DB), &accessUserRepositoryStub{})
		_, err := svc.List(adminContext(), &proto.ListAccessRequestsRequest{Status: "unknown"})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		_, err = svc.List(adminContext(), &proto.ListAccessRequestsRequest{PageSize: 101})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})
}

func TestAccessServiceModify(t *testing.T) {
	Convey("管理员可以通过待审批用户", t, func() {
		repository := &accessUserRepositoryStub{users: []model.User{{
			ID: 8, Nickname: "申请用户", Status: model.UserStatusPending, Role: model.UserRoleUser,
		}}}
		svc := NewAccessService(new(gorm.DB), repository)

		response, err := svc.Modify(adminContext(), &proto.ModifyAccessRequestRequest{
			UserId: "8", Status: model.UserStatusApproved,
		})

		So(err, ShouldBeNil)
		So(repository.updatedUserID, ShouldEqual, uint64(8))
		So(repository.updatedStatus, ShouldEqual, model.UserStatusApproved)
		So(response.AccessStatus, ShouldEqual, model.UserStatusApproved)
	})

	Convey("不能审批管理员自己或提交非法状态", t, func() {
		repository := &accessUserRepositoryStub{users: []model.User{{
			ID: 1, Nickname: "管理员", Status: model.UserStatusApproved, Role: model.UserRoleAdmin,
		}}}
		svc := NewAccessService(new(gorm.DB), repository)

		_, err := svc.Modify(adminContext(), &proto.ModifyAccessRequestRequest{UserId: "1", Status: model.UserStatusRejected})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		_, err = svc.Modify(adminContext(), &proto.ModifyAccessRequestRequest{UserId: "8", Status: model.UserStatusDisabled})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})
}
