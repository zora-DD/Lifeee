package service

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type placeRepositoryStub struct {
	items   []model.Place
	item    *model.Place
	total   int64
	err     error
	query   dao.PlaceListQuery
	created *model.Place
	updated *model.Place
	deleted uint64
	stats   dao.PlaceStats
}

func (s *placeRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.Place) error {
	if s.err != nil {
		return s.err
	}
	row.ID = 6
	s.created = row
	s.item = row
	return nil
}

func (s *placeRepositoryStub) GetByID(_ context.Context, _ *gorm.DB, _ uint64) (*model.Place, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.item == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return s.item, nil
}

func (s *placeRepositoryStub) Update(_ context.Context, _ *gorm.DB, row *model.Place) error {
	if s.err != nil {
		return s.err
	}
	s.updated = row
	s.item = row
	return nil
}

func (s *placeRepositoryStub) SoftDelete(_ context.Context, _ *gorm.DB, id uint64) error {
	s.deleted = id
	return s.err
}

func (s *placeRepositoryStub) List(_ context.Context, _ *gorm.DB, query dao.PlaceListQuery) ([]model.Place, int64, error) {
	s.query = query
	return s.items, s.total, s.err
}

func (s *placeRepositoryStub) Stats(_ context.Context, _ *gorm.DB) (dao.PlaceStats, error) {
	return s.stats, s.err
}

func TestPlaceList(t *testing.T) {
	Convey("状态筛选转换为明确的去过标记", t, func() {
		repository := new(placeRepositoryStub)
		svc := NewPlaceService(new(gorm.DB), repository)
		req := &proto.ListPlacesRequest{Category: " 自然风景 ", Status: "visited", Keyword: " 湖 ", PageSize: 2}

		So(svc.InitAndCheckListParam(context.Background(), req), ShouldBeNil)
		response, err := svc.List(context.Background(), req)

		So(err, ShouldBeNil)
		So(*repository.query.Visited, ShouldBeTrue)
		So(repository.query.Category, ShouldEqual, "自然风景")
		So(repository.query.Keyword, ShouldEqual, "湖")
		So(response.Page, ShouldEqual, int32(1))
		So(response.PageSize, ShouldEqual, int32(2))
	})

	Convey("非法状态和分页返回参数错误", t, func() {
		requests := []*proto.ListPlacesRequest{{Status: "unknown"}, {Page: -1}, {PageSize: 101}}
		for _, request := range requests {
			err := NewPlaceService(nil, nil).InitAndCheckListParam(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})

	Convey("列表响应包含 has_more", t, func() {
		repository := &placeRepositoryStub{
			items: []model.Place{{ID: 6, Name: "云栖湖畔", Category: "自然风景"}},
			total: 3,
			stats: dao.PlaceStats{OverallTotal: 9, VisitedCount: 4, WishlistCount: 5},
		}
		svc := NewPlaceService(new(gorm.DB), repository)
		req := &proto.ListPlacesRequest{Page: 1, PageSize: 2}
		So(svc.InitAndCheckListParam(context.Background(), req), ShouldBeNil)

		response, err := svc.List(context.Background(), req)

		So(err, ShouldBeNil)
		So(response.Items, ShouldHaveLength, 1)
		So(response.HasMore, ShouldBeTrue)
		So(response.OverallTotal, ShouldEqual, int64(9))
		So(response.VisitedCount, ShouldEqual, int64(4))
		So(response.WishlistCount, ShouldEqual, int64(5))
	})
}

func TestPlaceCRUD(t *testing.T) {
	Convey("自建封面只保存对象键", t, func() {
		row, err := buildPlace(
			"云栖湖畔", "自然风景", "🌿", false, 0, "", "湖边日落",
			"", "", "", "https://old.example/signed.jpg", "place/2026/09/a.jpg", "", "", "", 0, 0,
		)

		So(err, ShouldBeNil)
		So(row.CoverURL, ShouldBeNil)
		So(*row.CoverObjectKey, ShouldEqual, "place/2026/09/a.jpg")
	})

	Convey("地点不能引用吃喝图片", t, func() {
		_, err := buildPlace(
			"云栖湖畔", "自然风景", "🌿", false, 0, "", "湖边日落",
			"", "", "", "", "restaurant/2026/09/a.jpg", "", "", "", 0, 0,
		)

		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("创建已去地点时保留次数、日期和坐标", t, func() {
		repository := new(placeRepositoryStub)
		svc := NewPlaceService(new(gorm.DB), repository)

		item, err := svc.Create(context.Background(), &proto.CreatePlaceRequest{
			Name: " 云栖湖畔 ", Category: "自然风景", Icon: "🌿", Visited: true,
			VisitCount: 3, LastVisitedAt: "2026-08-17", Feature: " 湖边日落 ",
			Latitude: "30.12345678", Longitude: "120.76543219",
			DistanceKm: 320, DriveDurationMinutes: 270,
		})

		So(err, ShouldBeNil)
		So(item.Id, ShouldEqual, "6")
		So(repository.created.Name, ShouldEqual, "云栖湖畔")
		So(repository.created.VisitCount, ShouldEqual, uint32(3))
		So(repository.created.LastVisitedAt.Format("2006-01-02"), ShouldEqual, "2026-08-17")
		So(*repository.created.Latitude, ShouldEqual, "30.1234568")
		So(item.DistanceKm, ShouldEqual, uint32(320))
		So(item.DriveDurationMinutes, ShouldEqual, uint32(270))
	})

	Convey("未去过地点会清空次数和最近日期", t, func() {
		row, err := buildPlace("旧仓艺术街区", "城市探索", "🎨", false, 5, "2026-08-17", "旧厂房改造", "", "", "", "", "", "", "", "", 0, 0)

		So(err, ShouldBeNil)
		So(row.Visited, ShouldBeFalse)
		So(row.VisitCount, ShouldEqual, uint32(0))
		So(row.LastVisitedAt, ShouldBeNil)
	})

	Convey("已去过地点必须提供合法日期", t, func() {
		_, err := buildPlace("地点", "其他", "📍", true, 1, "", "特色", "", "", "", "", "", "", "", "", 0, 0)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("微信公众号攻略链接可保存并返回", t, func() {
		guideURL := "https://mp.weixin.qq.com/s?__biz=test&mid=1"
		repository := new(placeRepositoryStub)
		svc := NewPlaceService(new(gorm.DB), repository)

		item, err := svc.Create(context.Background(), &proto.CreatePlaceRequest{
			Name: "皖浙天路", Feature: "自驾风景线", GuideUrl: guideURL,
		})

		So(err, ShouldBeNil)
		So(repository.created.GuideURL, ShouldNotBeNil)
		So(*repository.created.GuideURL, ShouldEqual, guideURL)
		So(item.GuideUrl, ShouldEqual, guideURL)
	})

	Convey("攻略链接只允许 HTTPS 微信公众号文章", t, func() {
		invalidURLs := []string{
			"http://mp.weixin.qq.com/s?mid=1",
			"https://example.com/s?mid=1",
			"https://mp.weixin.qq.com/not-an-article",
		}
		for _, guideURL := range invalidURLs {
			_, err := buildPlace("地点", "其他", "📍", false, 0, "", "特色", "", "", "", "", "", "", "", guideURL, 0, 0)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})

	Convey("详情不存在返回资源不存在", t, func() {
		item, err := NewPlaceService(new(gorm.DB), new(placeRepositoryStub)).Get(context.Background(), &proto.GetPlaceRequest{Id: "6"})

		So(item, ShouldBeNil)
		So(errors.Is(err, hecode.ErrNotFound), ShouldBeTrue)
	})

	Convey("编辑和删除调用 DAO", t, func() {
		lastVisitedAt := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)
		repository := &placeRepositoryStub{item: &model.Place{ID: 6, Name: "旧名称", Feature: optionalString("旧特色"), LastVisitedAt: &lastVisitedAt}}
		svc := NewPlaceService(new(gorm.DB), repository)

		item, err := svc.Update(adminContext(), &proto.UpdatePlaceRequest{
			Id: "6", Name: "新名称", Category: "城市探索", Visited: false, Feature: "新特色",
		})
		So(err, ShouldBeNil)
		So(item.Name, ShouldEqual, "新名称")
		So(repository.updated.VisitCount, ShouldEqual, uint32(0))
		So(repository.updated.LastVisitedAt, ShouldBeNil)

		response, err := svc.Delete(adminContext(), &proto.DeletePlaceRequest{Id: "6"})
		So(err, ShouldBeNil)
		So(response.Deleted, ShouldBeTrue)
		So(repository.deleted, ShouldEqual, uint64(6))
	})
}
