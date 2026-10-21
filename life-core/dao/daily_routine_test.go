package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestDailyRoutineDao(t *testing.T) {
	ctx := context.Background()

	Convey("每日事项支持创建和按用户查询", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*daily_routines.*").WillReturnResult(sqlmock.NewResult(9, 1))
		row := &model.DailyRoutine{Title: "抹药", TimeOfDay: "08:00", Enabled: true, CreatedBy: 1}
		So(NewDailyRoutineDao().Create(ctx, db, row), ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(9))

		mock.ExpectQuery("SELECT .* FROM .*daily_routines.*created_by.*enabled.*deleted_at.*ORDER BY time_of_day ASC,id ASC").
			WithArgs(uint64(1), true).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "time_of_day", "enabled", "created_by"}).
				AddRow(9, "抹药", "08:00", true, 1))
		rows, err := NewDailyRoutineDao().ListByUser(ctx, db, 1, false)
		So(err, ShouldBeNil)
		So(rows, ShouldHaveLength, 1)
		So(rows[0].Title, ShouldEqual, "抹药")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("同一天重复打卡使用唯一键幂等更新", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*daily_checkins.*ON DUPLICATE KEY UPDATE.*").
			WillReturnResult(sqlmock.NewResult(12, 1))
		now := time.Now()
		row := &model.DailyCheckin{
			RoutineID: 9, CheckinDate: now, Completed: true, CheckedAt: &now, CreatedBy: 1,
		}
		So(NewDailyRoutineDao().UpsertCheckin(ctx, db, row), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
