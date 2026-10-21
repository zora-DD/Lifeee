package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestPlaceDaoCRUD(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	Convey("地点可创建并按 ID 查询", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*places.*").WillReturnResult(sqlmock.NewResult(41, 1))
		row := &model.Place{Name: "湖边公园", Category: "自然风景", CreatedBy: 3}
		So(NewPlaceDao().Create(ctx, db, row), ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(41))

		mock.ExpectQuery("SELECT .* FROM .*places.*deleted_at.*IS NULL.*LIMIT.*").
			WithArgs(uint64(41), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "category", "visited", "created_by", "created_at"}).
				AddRow(41, "湖边公园", "自然风景", false, 3, now))
		loaded, err := NewPlaceDao().GetByID(ctx, db, 41)
		So(err, ShouldBeNil)
		So(loaded.Name, ShouldEqual, "湖边公园")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("地点更新限定业务字段并支持软删除", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*places.* SET .*name.*category.*visited.*distance_km.*drive_duration_minutes.*guide_url.* WHERE id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*places.* SET .*deleted_at.* WHERE .*id.*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		d := NewPlaceDao()
		guideURL := "https://mp.weixin.qq.com/s?mid=1"
		So(d.Update(ctx, db, &model.Place{ID: 41, Name: "新名称", Category: "城市探索", Visited: true, GuideURL: &guideURL}), ShouldBeNil)
		So(d.SoftDelete(ctx, db, 41), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestPlaceDaoList(t *testing.T) {
	Convey("地点列表应用所有筛选项并按到访状态排序", t, func() {
		db, mock := newDAOTestDB(t)
		visited := false
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*places.*category.*visited.*name LIKE.*feature LIKE.*address LIKE.*created_by.*").
			WithArgs("自然风景", false, "%湖%", "%湖%", "%湖%", uint64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT .* FROM .*places.* ORDER BY visited DESC,last_visited_at DESC,created_at DESC,id DESC LIMIT .* OFFSET .*").
			WithArgs("自然风景", false, "%湖%", "%湖%", "%湖%", uint64(3), 10, 20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "category", "visited"}).AddRow(41, "湖边公园", "自然风景", false))

		rows, total, err := NewPlaceDao().List(context.Background(), db, PlaceListQuery{
			Category: "自然风景", Visited: &visited, Keyword: "湖", CreatedBy: 3, Page: 3, PageSize: 10,
		})

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestPlaceDaoStats(t *testing.T) {
	Convey("地点汇总一次返回去过和想去数量", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) AS overall_total.*visited = 1.*visited = 0.*FROM .*places.*deleted_at.*IS NULL").
			WillReturnRows(sqlmock.NewRows([]string{"overall_total", "visited_count", "wishlist_count"}).
				AddRow(9, 4, 5))

		stats, err := NewPlaceDao().Stats(context.Background(), db)

		So(err, ShouldBeNil)
		So(stats.OverallTotal, ShouldEqual, int64(9))
		So(stats.VisitedCount, ShouldEqual, int64(4))
		So(stats.WishlistCount, ShouldEqual, int64(5))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
