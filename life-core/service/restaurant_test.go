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

type restaurantRepositoryStub struct {
	items       []model.Restaurant
	item        *model.Restaurant
	total       int64
	err         error
	query       dao.RestaurantListQuery
	created     *model.Restaurant
	updated     *model.Restaurant
	summary     dao.RestaurantSummary
	duplicate   bool
	existsErr   error
	checkedName string
	excludedID  uint64
	stats       dao.RestaurantStats
}

func (s *restaurantRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.Restaurant) error {
	if s.err != nil {
		return s.err
	}
	row.ID = 8
	s.created = row
	s.item = row
	return nil
}

func (s *restaurantRepositoryStub) GetByID(_ context.Context, _ *gorm.DB, _ uint64) (*model.Restaurant, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.item == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return s.item, nil
}

func (s *restaurantRepositoryStub) ExistsByName(_ context.Context, _ *gorm.DB, name string, excludeID uint64) (bool, error) {
	s.checkedName = name
	s.excludedID = excludeID
	return s.duplicate, s.existsErr
}

func (s *restaurantRepositoryStub) GetByIDForUpdate(ctx context.Context, db *gorm.DB, id uint64) (*model.Restaurant, error) {
	return s.GetByID(ctx, db, id)
}

func (s *restaurantRepositoryStub) UpdateProfile(_ context.Context, _ *gorm.DB, row *model.Restaurant) error {
	if s.err != nil {
		return s.err
	}
	s.updated = row
	s.item = row
	return nil
}

func (s *restaurantRepositoryStub) UpdateSummary(_ context.Context, _ *gorm.DB, _ uint64, summary dao.RestaurantSummary) error {
	if s.err != nil {
		return s.err
	}
	s.summary = summary
	return nil
}

func (s *restaurantRepositoryStub) SoftDelete(_ context.Context, _ *gorm.DB, _ uint64) error {
	return s.err
}

func (s *restaurantRepositoryStub) List(_ context.Context, _ *gorm.DB, query dao.RestaurantListQuery) ([]model.Restaurant, int64, error) {
	s.query = query
	return s.items, s.total, s.err
}

func (s *restaurantRepositoryStub) Stats(_ context.Context, _ *gorm.DB) (dao.RestaurantStats, error) {
	return s.stats, s.err
}

type restaurantConsumptionRepositoryStub struct {
	items []model.RestaurantConsumption
	total int64
	err   error
}

func (s *restaurantConsumptionRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.RestaurantConsumption) error {
	if s.err == nil {
		row.ID = 12
	}
	return s.err
}

func (s *restaurantConsumptionRepositoryStub) GetByID(_ context.Context, _ *gorm.DB, _ uint64) (*model.RestaurantConsumption, error) {
	if s.err != nil {
		return nil, s.err
	}
	if len(s.items) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &s.items[0], nil
}

func (s *restaurantConsumptionRepositoryStub) Update(_ context.Context, _ *gorm.DB, _ *model.RestaurantConsumption) error {
	return s.err
}

func (s *restaurantConsumptionRepositoryStub) SoftDelete(_ context.Context, _ *gorm.DB, _ uint64) error {
	return s.err
}

func (s *restaurantConsumptionRepositoryStub) SoftDeleteByRestaurant(_ context.Context, _ *gorm.DB, _ uint64) error {
	return s.err
}

func (s *restaurantConsumptionRepositoryStub) ListByRestaurant(
	_ context.Context,
	_ *gorm.DB,
	_ uint64,
	_, _ int,
) ([]model.RestaurantConsumption, int64, error) {
	return s.items, s.total, s.err
}

func (s *restaurantConsumptionRepositoryStub) ListForSummary(
	_ context.Context,
	_ *gorm.DB,
	_ uint64,
) ([]model.RestaurantConsumption, error) {
	return s.items, s.err
}

func TestRestaurantList(t *testing.T) {
	Convey("列表参数会清洗并设置默认分页", t, func() {
		repository := new(restaurantRepositoryStub)
		svc := NewRestaurantService(new(gorm.DB), repository, new(restaurantConsumptionRepositoryStub))
		req := &proto.ListRestaurantsRequest{Category: " 小吃 ", Preference: " love ", Keyword: " 馄饨 ", Status: " frequent "}

		So(svc.InitAndCheckListParam(context.Background(), req), ShouldBeNil)
		response, err := svc.List(context.Background(), req)

		So(err, ShouldBeNil)
		So(repository.query.Category, ShouldEqual, "小吃")
		So(repository.query.Preference, ShouldEqual, model.RestaurantPreferenceLove)
		So(repository.query.Keyword, ShouldEqual, "馄饨")
		So(repository.query.Status, ShouldEqual, "frequent")
		So(response.Page, ShouldEqual, int32(1))
		So(response.PageSize, ShouldEqual, int32(20))
	})

	Convey("非法喜好和分页返回参数错误", t, func() {
		requests := []*proto.ListRestaurantsRequest{{Preference: "best"}, {Status: "unknown"}, {Page: -1}, {PageSize: 101}}
		for _, request := range requests {
			err := NewRestaurantService(nil, nil, nil).InitAndCheckListParam(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})

	Convey("列表响应包含分页结束标记", t, func() {
		repository := &restaurantRepositoryStub{
			items: []model.Restaurant{{ID: 8, Name: "巷口馄饨", Category: "小吃", Preference: model.RestaurantPreferenceLove, TotalSpent: "66.00"}},
			total: 3,
			stats: dao.RestaurantStats{OverallTotal: 8, TotalVisitCount: 21, TotalSpent: "688.50"},
		}
		svc := NewRestaurantService(new(gorm.DB), repository, new(restaurantConsumptionRepositoryStub))
		req := &proto.ListRestaurantsRequest{Page: 1, PageSize: 2}
		So(svc.InitAndCheckListParam(context.Background(), req), ShouldBeNil)

		response, err := svc.List(context.Background(), req)

		So(err, ShouldBeNil)
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].Name, ShouldEqual, "巷口馄饨")
		So(response.HasMore, ShouldBeTrue)
		So(response.OverallTotal, ShouldEqual, int64(8))
		So(response.TotalVisitCount, ShouldEqual, int64(21))
		So(response.TotalSpent, ShouldEqual, "688.50")
	})
}

func TestRestaurantCRUDValidation(t *testing.T) {
	Convey("自建封面只保存对象键", t, func() {
		row, err := buildRestaurant(
			"巷口小店", "小吃", "", model.RestaurantPreferenceGood,
			"", "", "", "https://old.example/signed.jpg", "restaurant/2026/09/a.jpg", "31.2304", "121.4737",
		)

		So(err, ShouldBeNil)
		So(row.CoverURL, ShouldBeNil)
		So(*row.CoverObjectKey, ShouldEqual, "restaurant/2026/09/a.jpg")
		So(*row.Latitude, ShouldEqual, "31.2304000")
		So(*row.Longitude, ShouldEqual, "121.4737000")
	})

	Convey("吃喝不能引用地点图片", t, func() {
		_, err := buildRestaurant(
			"巷口小店", "小吃", "", model.RestaurantPreferenceGood,
			"", "", "", "", "place/2026/09/a.jpg", "", "",
		)

		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("创建吃喝地点时设置默认喜好和创建人", t, func() {
		repository := new(restaurantRepositoryStub)
		svc := NewRestaurantService(new(gorm.DB), repository, new(restaurantConsumptionRepositoryStub))

		item, err := svc.Create(context.Background(), &proto.CreateRestaurantRequest{
			Name: " 巷口馄饨 ", Category: "小吃", Features: " 老店 · 性价比高 ", Latitude: "31.2304", Longitude: "121.4737",
		})

		So(err, ShouldBeNil)
		So(item.Id, ShouldEqual, "8")
		So(repository.created.Name, ShouldEqual, "巷口馄饨")
		So(repository.created.Preference, ShouldEqual, model.RestaurantPreferenceGood)
		So(repository.created.CreatedBy, ShouldEqual, defaultRestaurantCreator)
		So(repository.created.TotalSpent, ShouldEqual, "0.00")
		So(*repository.created.Features, ShouldEqual, "老店 · 性价比高")
		So(item.Latitude, ShouldEqual, "31.2304000")
		So(item.Longitude, ShouldEqual, "121.4737000")
		So(repository.checkedName, ShouldEqual, "巷口馄饨")
	})

	Convey("创建同名吃喝地点时返回明确提示", t, func() {
		repository := &restaurantRepositoryStub{duplicate: true}
		svc := NewRestaurantService(new(gorm.DB), repository, new(restaurantConsumptionRepositoryStub))

		item, err := svc.Create(context.Background(), &proto.CreateRestaurantRequest{Name: " 巷口馄饨 ", Category: "小吃"})

		So(item, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		So(err.Error(), ShouldContainSubstring, "已存在同名吃喝记录")
		So(repository.created, ShouldBeNil)
	})

	Convey("编辑改名时排除当前记录并检查重名", t, func() {
		repository := &restaurantRepositoryStub{
			item:      &model.Restaurant{ID: 8, Name: "旧店名"},
			duplicate: true,
		}
		svc := NewRestaurantService(new(gorm.DB), repository, new(restaurantConsumptionRepositoryStub))

		item, err := svc.Update(adminContext(), &proto.UpdateRestaurantRequest{Id: "8", Name: "新店名", Category: "其他"})

		So(item, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		So(repository.checkedName, ShouldEqual, "新店名")
		So(repository.excludedID, ShouldEqual, uint64(8))
		So(repository.updated, ShouldBeNil)
	})

	Convey("详情会返回消费记录", t, func() {
		amount := "28.50"
		repository := &restaurantRepositoryStub{item: &model.Restaurant{ID: 8, Name: "巷口馄饨", TotalSpent: amount}}
		consumptions := &restaurantConsumptionRepositoryStub{items: []model.RestaurantConsumption{{
			ID: 12, RestaurantID: 8, ConsumedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.Local), Amount: &amount,
		}}}
		svc := NewRestaurantService(new(gorm.DB), repository, consumptions)

		item, err := svc.Get(context.Background(), &proto.GetRestaurantRequest{Id: "8"})

		So(err, ShouldBeNil)
		So(item.Consumptions, ShouldHaveLength, 1)
		So(item.Consumptions[0].Amount, ShouldEqual, amount)
		So(item.Consumptions[0].ConsumedAt, ShouldEqual, "2026-09-02")
	})

	Convey("不存在的详情返回资源不存在", t, func() {
		svc := NewRestaurantService(new(gorm.DB), new(restaurantRepositoryStub), new(restaurantConsumptionRepositoryStub))

		item, err := svc.Get(context.Background(), &proto.GetRestaurantRequest{Id: "8"})

		So(item, ShouldBeNil)
		So(errors.Is(err, hecode.ErrNotFound), ShouldBeTrue)
	})
}

func TestRestaurantConsumptionSummary(t *testing.T) {
	Convey("金额按分精确累计并取最近一条作为摘要", t, func() {
		firstAmount := "12.30"
		secondAmount := "7.05"
		meal := "牛肉锅"
		note := "微辣"
		repository := new(restaurantRepositoryStub)
		consumptions := &restaurantConsumptionRepositoryStub{items: []model.RestaurantConsumption{
			{ConsumedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.Local), Amount: &firstAmount, Meal: &meal, Note: &note},
			{ConsumedAt: time.Date(2026, 8, 20, 0, 0, 0, 0, time.Local), Amount: &secondAmount},
		}}
		svc := NewRestaurantService(new(gorm.DB), repository, consumptions)

		err := svc.recalculateSummary(context.Background(), new(gorm.DB), 8)

		So(err, ShouldBeNil)
		So(repository.summary.VisitCount, ShouldEqual, uint32(2))
		So(repository.summary.TotalSpent, ShouldEqual, "19.35")
		So(repository.summary.LastVisitedAt.Format("2006-01-02"), ShouldEqual, "2026-09-02")
		So(*repository.summary.LastMeal, ShouldEqual, meal)
	})

	Convey("消费金额和日期校验正确", t, func() {
		valid, err := buildRestaurantConsumption("2026-09-02", "18.5", "咖啡", "")
		So(err, ShouldBeNil)
		So(*valid.Amount, ShouldEqual, "18.50")

		invalidInputs := [][2]string{{"明天", "10"}, {"2026-09-02", "-1"}, {"2026-09-02", "1.234"}}
		for _, input := range invalidInputs {
			_, err = buildRestaurantConsumption(input[0], input[1], "", "")
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})
}

func TestRestaurantRecommendation(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.Local)

	Convey("少消费、较久未吃和更喜欢都会提高推荐权重", t, func() {
		recent := now.AddDate(0, 0, -1)
		old := now.AddDate(0, 0, -60)
		base := model.Restaurant{Preference: model.RestaurantPreferenceOkay, VisitCount: 8, LastVisitedAt: &recent}
		fewerVisits := base
		fewerVisits.VisitCount = 1
		longAgo := base
		longAgo.LastVisitedAt = &old
		preferred := base
		preferred.Preference = model.RestaurantPreferenceMust

		So(restaurantRecommendationWeight(&fewerVisits, now), ShouldBeGreaterThan, restaurantRecommendationWeight(&base, now))
		So(restaurantRecommendationWeight(&longAgo, now), ShouldBeGreaterThan, restaurantRecommendationWeight(&base, now))
		So(restaurantRecommendationWeight(&preferred, now), ShouldBeGreaterThan, restaurantRecommendationWeight(&base, now))
	})

	Convey("加权抽样会按累计权重命中候选店", t, func() {
		rows := []model.Restaurant{
			{ID: 1, Preference: model.RestaurantPreferenceGood},
			{ID: 2, Preference: model.RestaurantPreferenceGood},
		}
		firstWeight := restaurantRecommendationWeight(&rows[0], now)
		selected, err := chooseWeightedRestaurant(rows, now, func(max int64) (int64, error) {
			return firstWeight, nil
		})

		So(err, ShouldBeNil)
		So(selected.ID, ShouldEqual, uint64(2))
	})

	Convey("推荐接口保留分类、喜好和状态筛选", t, func() {
		repository := &restaurantRepositoryStub{items: []model.Restaurant{{ID: 8, Name: "巷口馄饨"}}, total: 1}
		svc := NewRestaurantService(new(gorm.DB), repository, new(restaurantConsumptionRepositoryStub))

		item, err := svc.Recommend(context.Background(), &proto.RecommendRestaurantRequest{
			Category: " 小吃 ", Preference: model.RestaurantPreferenceLove, Status: "frequent",
		})

		So(err, ShouldBeNil)
		So(item.Id, ShouldEqual, "8")
		So(repository.query.Category, ShouldEqual, "小吃")
		So(repository.query.Preference, ShouldEqual, model.RestaurantPreferenceLove)
		So(repository.query.Status, ShouldEqual, "frequent")
	})

	Convey("推荐接口拒绝未知状态", t, func() {
		svc := NewRestaurantService(new(gorm.DB), new(restaurantRepositoryStub), new(restaurantConsumptionRepositoryStub))
		_, err := svc.Recommend(context.Background(), &proto.RecommendRestaurantRequest{Status: "unknown"})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})
}
