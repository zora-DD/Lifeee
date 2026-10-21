package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func restaurantRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "name", "category", "preference", "visit_count", "total_spent", "created_by", "created_at"}).
		AddRow(51, "小馆子", "小吃", model.RestaurantPreferenceLove, 2, "88.00", 3, now)
}

func TestRestaurantDaoCRUD(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	Convey("吃喝记录可创建并查询", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*restaurants.*").WillReturnResult(sqlmock.NewResult(51, 1))
		row := &model.Restaurant{Name: "小馆子", Category: "小吃", CreatedBy: 3}
		So(NewRestaurantDao().Create(ctx, db, row), ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(51))

		mock.ExpectQuery("SELECT .* FROM .*restaurants.*deleted_at.*IS NULL.*LIMIT.*").
			WithArgs(uint64(51), 1).WillReturnRows(restaurantRows(now))
		loaded, err := NewRestaurantDao().GetByID(ctx, db, 51)
		So(err, ShouldBeNil)
		So(loaded.Name, ShouldEqual, "小馆子")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("事务内查询使用 FOR UPDATE 行锁", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT .* FROM .*restaurants.*deleted_at.*IS NULL.*LIMIT .* FOR UPDATE").
			WithArgs(uint64(51), 1).WillReturnRows(restaurantRows(now))

		row, err := NewRestaurantDao().GetByIDForUpdate(ctx, db, 51)

		So(err, ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(51))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("店铺资料和消费汇总分开更新", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*restaurants.* SET .*name.*category.*preference.* WHERE id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*restaurants.* SET .*last_amount.*last_meal.*last_note.*last_visited_at.*total_spent.*visit_count.* WHERE id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		d := NewRestaurantDao()
		So(d.UpdateProfile(ctx, db, &model.Restaurant{ID: 51, Name: "新店名", Category: "火锅", Preference: model.RestaurantPreferenceMust}), ShouldBeNil)
		So(d.UpdateSummary(ctx, db, 51, RestaurantSummary{VisitCount: 3, TotalSpent: "128.00"}), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("吃喝记录使用软删除", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*restaurants.* SET .*deleted_at.* WHERE .*id.*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		So(NewRestaurantDao().SoftDelete(ctx, db, 51), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestRestaurantDaoExistsByName(t *testing.T) {
	Convey("名称查询自动排除软删除记录并支持排除当前 ID", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*restaurants.*name.*id <>.*deleted_at.*IS NULL").
			WithArgs("小馆子", uint64(51)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := NewRestaurantDao().ExistsByName(context.Background(), db, "小馆子", 51)

		So(err, ShouldBeNil)
		So(exists, ShouldBeTrue)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestRestaurantDaoList(t *testing.T) {
	Convey("吃喝列表应用筛选并按最近消费时间排序", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*restaurants.*category.*preference.*name LIKE.*cuisine LIKE.*address LIKE.*created_by.*").
			WithArgs("小吃", model.RestaurantPreferenceLove, "%馆%", "%馆%", "%馆%", uint64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT .* FROM .*restaurants.* ORDER BY last_visited_at IS NULL ASC,last_visited_at DESC,id DESC LIMIT .*").
			WithArgs("小吃", model.RestaurantPreferenceLove, "%馆%", "%馆%", "%馆%", uint64(3), 10).
			WillReturnRows(restaurantRows(time.Now()))

		rows, total, err := NewRestaurantDao().List(context.Background(), db, RestaurantListQuery{
			Category: "小吃", Preference: model.RestaurantPreferenceLove, Keyword: "馆", CreatedBy: 3, Page: 1, PageSize: 10,
		})

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("吃喝状态按消费次数派生筛选", t, func() {
		tests := []struct {
			status string
			where  string
		}{
			{status: "wishlist", where: "visit_count = 0"},
			{status: "visited", where: "visit_count > 0 AND visit_count < 3"},
			{status: "frequent", where: "visit_count >= 3"},
		}
		for _, test := range tests {
			db, mock := newDAOTestDB(t)
			mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*restaurants.*" + test.where).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectQuery("SELECT .* FROM .*restaurants.*" + test.where + ".*ORDER BY").
				WithArgs(10).
				WillReturnRows(restaurantRows(time.Now()))

			_, _, err := NewRestaurantDao().List(context.Background(), db, RestaurantListQuery{Status: test.status, Page: 1, PageSize: 10})

			So(err, ShouldBeNil)
			So(mock.ExpectationsWereMet(), ShouldBeNil)
		}
	})
}

func TestRestaurantDaoStats(t *testing.T) {
	Convey("吃喝汇总不加载列表明细", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) AS overall_total.*SUM\\(visit_count\\).*SUM\\(total_spent\\).*FROM .*restaurants.*deleted_at.*IS NULL").
			WillReturnRows(sqlmock.NewRows([]string{"overall_total", "total_visit_count", "total_spent"}).
				AddRow(8, 21, "688.50"))

		stats, err := NewRestaurantDao().Stats(context.Background(), db)

		So(err, ShouldBeNil)
		So(stats.OverallTotal, ShouldEqual, int64(8))
		So(stats.TotalVisitCount, ShouldEqual, int64(21))
		So(stats.TotalSpent, ShouldEqual, "688.50")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
