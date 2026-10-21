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

type favoriteRepositoryStub struct {
	row       *model.Favorite
	getErr    error
	createErr error
	deleteErr error
	created   *model.Favorite
	deleted   bool
}

func (s *favoriteRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.Favorite) error {
	s.created = row
	if s.createErr == nil {
		s.row, s.getErr = row, nil
	}
	return s.createErr
}

func (s *favoriteRepositoryStub) GetByUserTarget(_ context.Context, _ *gorm.DB, _ uint64, _ string, _ uint64) (*model.Favorite, error) {
	return s.row, s.getErr
}

func (s *favoriteRepositoryStub) DeleteByUserTarget(_ context.Context, _ *gorm.DB, _ uint64, _ string, _ uint64) error {
	s.deleted = true
	return s.deleteErr
}

type favoriteContentRepositoryStub struct {
	content    *dao.ContentSummary
	contentErr error
	rows       []dao.FavoriteContent
	total      int64
}

func (s *favoriteContentRepositoryStub) GetByTarget(context.Context, *gorm.DB, string, uint64) (*dao.ContentSummary, error) {
	return s.content, s.contentErr
}

func (s *favoriteContentRepositoryStub) ListFavorites(context.Context, *gorm.DB, uint64, string, int, int) ([]dao.FavoriteContent, int64, error) {
	return s.rows, s.total, nil
}

func favoriteTestContext() context.Context {
	return auth.WithUser(context.Background(), &model.User{ID: 7})
}

func TestFavoriteService(t *testing.T) {
	Convey("收藏存在内容并返回收藏状态", t, func() {
		favorites := &favoriteRepositoryStub{getErr: gorm.ErrRecordNotFound}
		contents := &favoriteContentRepositoryStub{content: &dao.ContentSummary{TargetID: 9}}
		svc := NewFavoriteService(new(gorm.DB), favorites, contents)

		response, err := svc.Modify(favoriteTestContext(), &proto.ModifyFavoriteRequest{
			TargetType: model.FavoriteTargetTask, TargetId: "9", Favorited: true,
		})

		So(err, ShouldBeNil)
		So(response.Favorited, ShouldBeTrue)
		So(favorites.created.UserID, ShouldEqual, uint64(7))
		So(favorites.created.TargetID, ShouldEqual, uint64(9))
	})

	Convey("重复收藏和重复取消保持幂等", t, func() {
		favorites := &favoriteRepositoryStub{row: &model.Favorite{ID: 1}}
		contents := &favoriteContentRepositoryStub{content: &dao.ContentSummary{TargetID: 9}}
		svc := NewFavoriteService(new(gorm.DB), favorites, contents)

		response, err := svc.Modify(favoriteTestContext(), &proto.ModifyFavoriteRequest{
			TargetType: model.FavoriteTargetTask, TargetId: "9", Favorited: true,
		})
		So(err, ShouldBeNil)
		So(response.Favorited, ShouldBeTrue)
		So(favorites.created, ShouldBeNil)

		response, err = svc.Modify(favoriteTestContext(), &proto.ModifyFavoriteRequest{
			TargetType: model.FavoriteTargetTask, TargetId: "9", Favorited: false,
		})
		So(err, ShouldBeNil)
		So(response.Favorited, ShouldBeFalse)
		So(favorites.deleted, ShouldBeTrue)
	})

	Convey("收藏不存在内容返回资源不存在", t, func() {
		favorites := &favoriteRepositoryStub{getErr: gorm.ErrRecordNotFound}
		contents := &favoriteContentRepositoryStub{contentErr: gorm.ErrRecordNotFound}
		response, err := NewFavoriteService(new(gorm.DB), favorites, contents).Modify(
			favoriteTestContext(),
			&proto.ModifyFavoriteRequest{TargetType: model.FavoriteTargetPlace, TargetId: "99", Favorited: true},
		)
		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrNotFound), ShouldBeTrue)
	})

	Convey("列表返回收藏时间和内容摘要", t, func() {
		now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.Local)
		contents := &favoriteContentRepositoryStub{
			rows: []dao.FavoriteContent{{
				FavoriteID: 3, FavoriteCreatedAt: now,
				ContentSummary: dao.ContentSummary{TargetID: 9, TargetType: model.FavoriteTargetRestaurant, Title: "小馆"},
			}},
			total: 1,
		}
		response, err := NewFavoriteService(new(gorm.DB), new(favoriteRepositoryStub), contents).
			List(favoriteTestContext(), &proto.ListFavoritesRequest{})
		So(err, ShouldBeNil)
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].Id, ShouldEqual, "3")
		So(response.Items[0].Target.Title, ShouldEqual, "小馆")
	})
}
