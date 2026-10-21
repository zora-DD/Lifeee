package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func consumptionRows(day time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "restaurant_id", "consumed_by", "consumed_at", "amount"}).
		AddRow(61, 51, 3, day, "42.00")
}

func TestRestaurantConsumptionDaoCRUD(t *testing.T) {
	ctx := context.Background()
	day := time.Date(2026, 9, 4, 0, 0, 0, 0, time.Local)

	Convey("消费明细可创建并查询", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*restaurant_consumptions.*").WillReturnResult(sqlmock.NewResult(61, 1))
		row := &model.RestaurantConsumption{RestaurantID: 51, ConsumedBy: 3, ConsumedAt: day}
		So(NewRestaurantConsumptionDao().Create(ctx, db, row), ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(61))

		mock.ExpectQuery("SELECT .* FROM .*restaurant_consumptions.*deleted_at.*IS NULL.*LIMIT.*").
			WithArgs(uint64(61), 1).WillReturnRows(consumptionRows(day))
		loaded, err := NewRestaurantConsumptionDao().GetByID(ctx, db, 61)
		So(err, ShouldBeNil)
		So(loaded.RestaurantID, ShouldEqual, uint64(51))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("消费明细可更新、单条删除和按店铺删除", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*restaurant_consumptions.* SET .*consumed_at.*amount.* WHERE id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*restaurant_consumptions.* SET .*deleted_at.* WHERE .*id.*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*restaurant_consumptions.* SET .*deleted_at.* WHERE restaurant_id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 2))
		d := NewRestaurantConsumptionDao()
		So(d.Update(ctx, db, &model.RestaurantConsumption{ID: 61, ConsumedAt: day}), ShouldBeNil)
		So(d.SoftDelete(ctx, db, 61), ShouldBeNil)
		So(d.SoftDeleteByRestaurant(ctx, db, 51), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestRestaurantConsumptionDaoLists(t *testing.T) {
	ctx := context.Background()
	day := time.Date(2026, 9, 4, 0, 0, 0, 0, time.Local)

	Convey("消费明细按店铺分页且顺序稳定", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*restaurant_consumptions.*restaurant_id.*").
			WithArgs(uint64(51)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT .* FROM .*restaurant_consumptions.* ORDER BY consumed_at DESC,id DESC LIMIT .* OFFSET .*").
			WithArgs(uint64(51), 10, 10).WillReturnRows(consumptionRows(day))

		rows, total, err := NewRestaurantConsumptionDao().ListByRestaurant(ctx, db, 51, 2, 10)

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("汇总重算读取店铺全部有效明细", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT .* FROM .*restaurant_consumptions.*restaurant_id.*deleted_at.*IS NULL.*ORDER BY consumed_at DESC,id DESC").
			WithArgs(uint64(51)).WillReturnRows(consumptionRows(day))

		rows, err := NewRestaurantConsumptionDao().ListForSummary(ctx, db, 51)

		So(err, ShouldBeNil)
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
