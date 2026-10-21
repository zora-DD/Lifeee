package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestContentDaoListContributions(t *testing.T) {
	Convey("我的贡献合并三类内容并统一分页", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("(?s)SELECT COUNT\\(1\\) FROM \\(SELECT .*tasks.*UNION ALL.*restaurants.*UNION ALL.*places.*\\) AS contents").
			WithArgs(uint64(3), uint64(3), uint64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("(?s)SELECT \\* FROM \\(SELECT .*tasks.*UNION ALL.*restaurants.*UNION ALL.*places.*\\) AS contents ORDER BY created_at DESC, target_id DESC LIMIT .* OFFSET .*").
			WithArgs(uint64(3), uint64(3), uint64(3), 10, 10).
			WillReturnRows(contentSummaryRows())

		rows, total, err := NewContentDao().ListContributions(context.Background(), db, 3, "", 2, 10)

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(rows[0].Title, ShouldEqual, "买水果")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("指定类型时只查询对应内容表", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("(?s)SELECT COUNT\\(1\\) FROM \\(SELECT .* FROM tasks .*\\) AS contents").
			WithArgs(uint64(3)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("(?s)SELECT \\* FROM \\(SELECT .* FROM tasks .*\\) AS contents ORDER BY created_at DESC, target_id DESC LIMIT .* OFFSET .*").
			WithArgs(uint64(3), 20, 0).WillReturnRows(contentSummaryRowsEmpty())

		rows, total, err := NewContentDao().ListContributions(context.Background(), db, 3, model.FavoriteTargetTask, 1, 20)

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(0))
		So(rows, ShouldBeEmpty)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestContentDaoGetByTarget(t *testing.T) {
	now := time.Now()

	Convey("可将任务、吃喝和地点映射为统一摘要", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT .* FROM .*tasks.*deleted_at.*IS NULL.*LIMIT.*").WithArgs(uint64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "category", "status", "priority", "created_at"}).
				AddRow(1, "买水果", "生活", model.TaskStatusTodo, model.TaskPriorityHigh, now))
		mock.ExpectQuery("SELECT .* FROM .*restaurants.*deleted_at.*IS NULL.*LIMIT.*").WithArgs(uint64(2), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "category", "preference", "visit_count", "total_spent", "cover_url", "created_at"}).
				AddRow(2, "小馆子", "小吃", model.RestaurantPreferenceLove, 2, "88.00", "https://img.example/restaurant.jpg", now))
		mock.ExpectQuery("SELECT .* FROM .*places.*deleted_at.*IS NULL.*LIMIT.*").WithArgs(uint64(3), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "category", "visited", "visit_count", "icon", "cover_url", "created_at"}).
				AddRow(3, "湖边", "自然风景", false, 0, "湖", "https://img.example/place.jpg", now))
		d := NewContentDao()
		ctx := context.Background()

		task, err := d.GetByTarget(ctx, db, model.FavoriteTargetTask, 1)
		So(err, ShouldBeNil)
		restaurant, err := d.GetByTarget(ctx, db, model.FavoriteTargetRestaurant, 2)
		So(err, ShouldBeNil)
		place, err := d.GetByTarget(ctx, db, model.FavoriteTargetPlace, 3)
		So(err, ShouldBeNil)
		So(task.TargetType, ShouldEqual, model.FavoriteTargetTask)
		So(restaurant.TotalSpent, ShouldEqual, "88.00")
		So(restaurant.CoverURL, ShouldEqual, "https://img.example/restaurant.jpg")
		So(place.Title, ShouldEqual, "湖边")
		So(place.Icon, ShouldEqual, "湖")
		So(place.CoverURL, ShouldEqual, "https://img.example/place.jpg")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("不支持的内容类型直接返回错误", t, func() {
		db, _ := newDAOTestDB(t)
		row, err := NewContentDao().GetByTarget(context.Background(), db, "unknown", 1)
		So(row, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}

func TestContentDaoListFavorites(t *testing.T) {
	Convey("我的收藏只返回存在的内容并按收藏时间分页", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("(?s)SELECT COUNT\\(1\\) FROM favorites f JOIN \\(SELECT .*tasks.*UNION ALL.*restaurants.*UNION ALL.*places.*\\) AS c .* WHERE f.user_id = .* AND f.target_type = .*").
			WithArgs(uint64(3), model.FavoriteTargetTask).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("(?s)SELECT c\\.\\*, f\\.id AS favorite_id, f\\.created_at AS favorite_created_at FROM favorites f JOIN .* ORDER BY f.created_at DESC, f.id DESC LIMIT .* OFFSET .*").
			WithArgs(uint64(3), model.FavoriteTargetTask, 10, 0).
			WillReturnRows(sqlmock.NewRows([]string{
				"target_id", "target_type", "title", "category", "status", "priority", "created_at", "favorite_id", "favorite_created_at",
			}).AddRow(1, model.FavoriteTargetTask, "买水果", "生活", model.TaskStatusTodo, model.TaskPriorityHigh, time.Now(), 81, time.Now()))

		rows, total, err := NewContentDao().ListFavorites(context.Background(), db, 3, model.FavoriteTargetTask, 1, 10)

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(rows[0].FavoriteID, ShouldEqual, uint64(81))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func contentSummaryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"target_id", "target_type", "title", "category", "status", "priority", "created_at",
	}).AddRow(1, model.FavoriteTargetTask, "买水果", "生活", model.TaskStatusTodo, model.TaskPriorityHigh, time.Now())
}

func contentSummaryRowsEmpty() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"target_id", "target_type", "title", "category", "status", "priority", "created_at",
	})
}
