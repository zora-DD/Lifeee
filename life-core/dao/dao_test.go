package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"life-core/model"
)

func newDAOTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}

func TestPageLimit(t *testing.T) {
	Convey("分页参数使用默认值", t, func() {
		offset, limit := pageLimit(0, 0)
		So(offset, ShouldEqual, 0)
		So(limit, ShouldEqual, defaultPageSize)
	})

	Convey("分页偏移计算正确", t, func() {
		offset, limit := pageLimit(3, 10)
		So(offset, ShouldEqual, 20)
		So(limit, ShouldEqual, 10)
	})

	Convey("每页数量不能超过上限", t, func() {
		offset, limit := pageLimit(2, maxPageSize+1)
		So(offset, ShouldEqual, maxPageSize)
		So(limit, ShouldEqual, maxPageSize)
	})
}

func TestTaskDaoNilArguments(t *testing.T) {
	d := NewTaskDao()
	ctx := context.Background()

	Convey("Create 拒绝 nil 数据", t, func() {
		So(d.Create(ctx, nil, nil), ShouldNotBeNil)
		So(d.Create(ctx, nil, &model.Task{}), ShouldNotBeNil)
	})

	Convey("Update 拒绝 nil 数据", t, func() {
		So(d.Update(ctx, nil, nil), ShouldNotBeNil)
		So(d.Update(ctx, nil, &model.Task{}), ShouldNotBeNil)
	})

	Convey("所有数据库操作拒绝 nil DB", t, func() {
		So(d.Create(ctx, nil, &model.Task{}), ShouldNotBeNil)
		_, err := d.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		So(d.UpdateStatus(ctx, nil, 1, model.TaskStatusDone), ShouldNotBeNil)
		So(d.SoftDelete(ctx, nil, 1), ShouldNotBeNil)
		_, _, err = d.List(ctx, nil, TaskListQuery{})
		So(err, ShouldNotBeNil)
	})
}

func TestRestaurantDaoNilArguments(t *testing.T) {
	ctx := context.Background()
	restaurantDao := NewRestaurantDao()
	consumptionDao := NewRestaurantConsumptionDao()

	Convey("吃喝 DAO 拒绝 nil 参数", t, func() {
		So(restaurantDao.Create(ctx, nil, nil), ShouldNotBeNil)
		So(restaurantDao.Create(ctx, nil, &model.Restaurant{}), ShouldNotBeNil)
		_, err := restaurantDao.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		_, err = restaurantDao.GetByIDForUpdate(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		So(restaurantDao.UpdateProfile(ctx, nil, nil), ShouldNotBeNil)
		So(restaurantDao.UpdateProfile(ctx, nil, &model.Restaurant{}), ShouldNotBeNil)
		So(restaurantDao.UpdateSummary(ctx, nil, 1, RestaurantSummary{}), ShouldNotBeNil)
		So(restaurantDao.SoftDelete(ctx, nil, 1), ShouldNotBeNil)
		_, _, err = restaurantDao.List(ctx, nil, RestaurantListQuery{})
		So(err, ShouldNotBeNil)

		So(consumptionDao.Create(ctx, nil, nil), ShouldNotBeNil)
		So(consumptionDao.Create(ctx, nil, &model.RestaurantConsumption{}), ShouldNotBeNil)
		_, err = consumptionDao.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		So(consumptionDao.Update(ctx, nil, nil), ShouldNotBeNil)
		So(consumptionDao.Update(ctx, nil, &model.RestaurantConsumption{}), ShouldNotBeNil)
		So(consumptionDao.SoftDelete(ctx, nil, 1), ShouldNotBeNil)
		So(consumptionDao.SoftDeleteByRestaurant(ctx, nil, 1), ShouldNotBeNil)
		_, _, err = consumptionDao.ListByRestaurant(ctx, nil, 1, 1, 20)
		So(err, ShouldNotBeNil)
		_, err = consumptionDao.ListForSummary(ctx, nil, 1)
		So(err, ShouldNotBeNil)
	})
}

func TestPlaceDaoNilArguments(t *testing.T) {
	d := NewPlaceDao()
	ctx := context.Background()

	Convey("地点 DAO 拒绝 nil 参数", t, func() {
		So(d.Create(ctx, nil, nil), ShouldNotBeNil)
		So(d.Create(ctx, nil, &model.Place{}), ShouldNotBeNil)
		_, err := d.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		So(d.Update(ctx, nil, nil), ShouldNotBeNil)
		So(d.Update(ctx, nil, &model.Place{}), ShouldNotBeNil)
		So(d.SoftDelete(ctx, nil, 1), ShouldNotBeNil)
		_, _, err = d.List(ctx, nil, PlaceListQuery{})
		So(err, ShouldNotBeNil)
	})
}

func TestOtherDAONilArguments(t *testing.T) {
	ctx := context.Background()

	Convey("生活记录 DAO 拒绝 nil 参数", t, func() {
		d := NewActivityLogDao()
		So(d.Create(ctx, nil, nil), ShouldNotBeNil)
		So(d.Create(ctx, nil, &model.ActivityLog{}), ShouldNotBeNil)
		_, err := d.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		So(d.Update(ctx, nil, nil), ShouldNotBeNil)
		So(d.Update(ctx, nil, &model.ActivityLog{}), ShouldNotBeNil)
		So(d.SoftDelete(ctx, nil, 1), ShouldNotBeNil)
		_, _, err = d.List(ctx, nil, ActivityLogListQuery{})
		So(err, ShouldNotBeNil)
		_, err = d.ListForStats(ctx, nil, 1, time.Time{}, time.Time{})
		So(err, ShouldNotBeNil)
	})

	Convey("账单 DAO 拒绝 nil 参数", t, func() {
		d := NewBillDao()
		So(d.Create(ctx, nil, nil), ShouldNotBeNil)
		So(d.Create(ctx, nil, &model.Bill{}), ShouldNotBeNil)
		_, err := d.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		So(d.Update(ctx, nil, nil), ShouldNotBeNil)
		So(d.Update(ctx, nil, &model.Bill{}), ShouldNotBeNil)
		So(d.SoftDelete(ctx, nil, 1), ShouldNotBeNil)
		_, _, err = d.List(ctx, nil, BillListQuery{})
		So(err, ShouldNotBeNil)
		_, err = d.ListCreators(ctx, nil)
		So(err, ShouldNotBeNil)
		_, err = d.Aggregate(ctx, nil, 1, time.Time{}, time.Time{})
		So(err, ShouldNotBeNil)
		_, err = d.AggregateMonthly(ctx, nil, 1, time.Time{}, time.Time{})
		So(err, ShouldNotBeNil)
		_, err = d.AggregateUsers(ctx, nil, time.Time{}, time.Time{})
		So(err, ShouldNotBeNil)
	})

	Convey("收藏与内容 DAO 拒绝 nil DB", t, func() {
		favorite := NewFavoriteDao()
		So(favorite.Create(ctx, nil, nil), ShouldNotBeNil)
		So(favorite.Create(ctx, nil, &model.Favorite{}), ShouldNotBeNil)
		_, err := favorite.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		_, err = favorite.GetByUserTarget(ctx, nil, 1, model.FavoriteTargetTask, 2)
		So(err, ShouldNotBeNil)
		So(favorite.DeleteByUserTarget(ctx, nil, 1, model.FavoriteTargetTask, 2), ShouldNotBeNil)
		So(favorite.DeleteByTarget(ctx, nil, model.FavoriteTargetTask, 2), ShouldNotBeNil)
		_, _, err = favorite.ListByUser(ctx, nil, 1, "", 1, 20)
		So(err, ShouldNotBeNil)

		content := NewContentDao()
		_, _, err = content.ListContributions(ctx, nil, 1, "", 1, 20)
		So(err, ShouldNotBeNil)
		_, err = content.GetByTarget(ctx, nil, model.FavoriteTargetTask, 1)
		So(err, ShouldNotBeNil)
		_, _, err = content.ListFavorites(ctx, nil, 1, "", 1, 20)
		So(err, ShouldNotBeNil)
	})

	Convey("媒体与用户 DAO 拒绝 nil 参数", t, func() {
		media := NewMediaAssetDao()
		So(media.Create(ctx, nil, nil), ShouldNotBeNil)
		So(media.Create(ctx, nil, &model.MediaAsset{}), ShouldNotBeNil)
		_, err := media.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		_, err = media.GetByObjectKey(ctx, nil, "key")
		So(err, ShouldNotBeNil)
		So(media.UpdateStatus(ctx, nil, 1, model.MediaStatusDeleted), ShouldNotBeNil)
		_, _, err = media.List(ctx, nil, MediaAssetListQuery{})
		So(err, ShouldNotBeNil)

		user := NewUserDao()
		So(user.Create(ctx, nil, nil), ShouldNotBeNil)
		So(user.Create(ctx, nil, &model.User{}), ShouldNotBeNil)
		_, err = user.GetByID(ctx, nil, 1)
		So(err, ShouldNotBeNil)
		_, err = user.GetByOpenID(ctx, nil, "openid")
		So(err, ShouldNotBeNil)
		_, err = user.GetByUnionID(ctx, nil, "unionid")
		So(err, ShouldNotBeNil)
		_, err = user.Count(ctx, nil)
		So(err, ShouldNotBeNil)
		So(user.UpdateProfile(ctx, nil, nil), ShouldNotBeNil)
		So(user.UpdateProfile(ctx, nil, &model.User{}), ShouldNotBeNil)
		So(user.UpdateStatus(ctx, nil, 1, model.UserStatusApproved), ShouldNotBeNil)
		So(user.UpdateRole(ctx, nil, 1, model.UserRoleAdmin), ShouldNotBeNil)
		So(user.UpdateLastLoginAt(ctx, nil, 1, time.Now()), ShouldNotBeNil)
		_, _, err = user.List(ctx, nil, UserListQuery{})
		So(err, ShouldNotBeNil)
	})
}
