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
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type contributionRepositoryStub struct {
	rows       []dao.ContentSummary
	total      int64
	userID     uint64
	targetType string
	page       int
	pageSize   int
	err        error
}

func (s *contributionRepositoryStub) ListContributions(
	_ context.Context, _ *gorm.DB, userID uint64, targetType string, page, pageSize int,
) ([]dao.ContentSummary, int64, error) {
	s.userID, s.targetType, s.page, s.pageSize = userID, targetType, page, pageSize
	return s.rows, s.total, s.err
}

func TestContributionServiceList(t *testing.T) {
	Convey("只按当前登录用户查询并返回统一摘要", t, func() {
		now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.Local)
		repository := &contributionRepositoryStub{
			rows:  []dao.ContentSummary{{TargetID: 8, TargetType: model.FavoriteTargetTask, Title: "整理房间", CreatedAt: now}},
			total: 3,
		}
		ctx := auth.WithUser(context.Background(), &model.User{ID: 6})

		response, err := NewContributionService(new(gorm.DB), repository).List(ctx, &proto.ListContributionsRequest{
			TargetType: "task", Page: 1, PageSize: 2,
		})

		So(err, ShouldBeNil)
		So(repository.userID, ShouldEqual, uint64(6))
		So(repository.targetType, ShouldEqual, "task")
		So(response.Items[0].TargetId, ShouldEqual, "8")
		So(response.HasMore, ShouldBeTrue)
	})

	Convey("未登录和非法类型会被拒绝", t, func() {
		svc := NewContributionService(new(gorm.DB), new(contributionRepositoryStub))
		_, err := svc.List(context.Background(), &proto.ListContributionsRequest{})
		So(errors.Is(err, hecode.ErrUnauthorized), ShouldBeTrue)

		ctx := auth.WithUser(context.Background(), &model.User{ID: 1})
		_, err = svc.List(ctx, &proto.ListContributionsRequest{TargetType: "unknown"})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})
}
