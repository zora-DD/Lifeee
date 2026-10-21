package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestActivityLogDaoCRUD(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 4, 9, 30, 0, 0, time.Local)

	Convey("生活记录可创建并回填主键", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*activity_logs.*").WillReturnResult(sqlmock.NewResult(11, 1))
		row := &model.ActivityLog{Content: "收拾房间", OccurredAt: now, CreatedBy: 2}

		err := NewActivityLogDao().Create(ctx, db, row)

		So(err, ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(11))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("按 ID 查询自动排除软删除记录", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT .* FROM .*activity_logs.*deleted_at.*IS NULL.*LIMIT.*").
			WithArgs(uint64(11), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "content", "occurred_at", "created_by", "creator_nickname", "creator_emoji", "creator_color"}).
				AddRow(11, "收拾房间", now, 2, "晚风收藏家", "🌿", "#759B89"))

		row, err := NewActivityLogDao().GetByID(ctx, db, 11)

		So(err, ShouldBeNil)
		So(row.Content, ShouldEqual, "收拾房间")
		So(row.CreatorNickname, ShouldEqual, "晚风收藏家")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("更新和删除均使用受限字段与软删除", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*activity_logs.* SET .*content.*occurred_at.* WHERE id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*activity_logs.* SET .*deleted_at.* WHERE .*id.*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		d := NewActivityLogDao()

		So(d.Update(ctx, db, &model.ActivityLog{ID: 11, Content: "已整理", OccurredAt: now}), ShouldBeNil)
		So(d.SoftDelete(ctx, db, 11), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestActivityLogDaoList(t *testing.T) {
	Convey("列表查询应用创建人、时间和输入方式筛选", t, func() {
		db, mock := newDAOTestDB(t)
		from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		to := from.AddDate(0, 1, 0)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*activity_logs.*created_by.*occurred_at.*input_type.*").
			WithArgs(uint64(2), from, to, model.ActivityInputTypeVoice).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT .* FROM .*activity_logs.*LEFT JOIN users.* ORDER BY activity_logs.occurred_at DESC,activity_logs.id DESC LIMIT .* OFFSET .*").
			WithArgs(uint64(2), from, to, model.ActivityInputTypeVoice, 10, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "content", "occurred_at", "input_type", "created_by", "creator_nickname", "creator_emoji", "creator_color"}).
				AddRow(12, "语音记录", from.Add(time.Hour), model.ActivityInputTypeVoice, 2, "云朵漫游者", "☁️", "#8AA6B8"))

		rows, total, err := NewActivityLogDao().List(context.Background(), db, ActivityLogListQuery{
			CreatedBy: 2, From: &from, To: &to, InputType: model.ActivityInputTypeVoice, Page: 2, PageSize: 10,
		})

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(rows[0].CreatorNickname, ShouldEqual, "云朵漫游者")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("统计查询只读取轻量字段并按时间正序", t, func() {
		db, mock := newDAOTestDB(t)
		from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		to := from.AddDate(0, 1, 0)
		mock.ExpectQuery("SELECT .*occurred_at.*,.*duration_minutes.* FROM .*activity_logs.*occurred_at.*created_by.*ORDER BY occurred_at ASC,id ASC").
			WithArgs(from, to, uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"occurred_at", "duration_minutes"}).AddRow(from, 30))

		rows, err := NewActivityLogDao().ListForStats(context.Background(), db, 2, from, to)

		So(err, ShouldBeNil)
		So(rows, ShouldHaveLength, 1)
		So(*rows[0].DurationMinutes, ShouldEqual, uint16(30))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
