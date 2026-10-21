package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func taskRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "title", "description", "status", "priority", "category",
		"due_at", "created_by", "created_at", "updated_at", "deleted_at",
		"creator_nickname", "creator_emoji", "creator_color",
	}).AddRow(7, "买水果", nil, "todo", "high", "生活", nil, 3, now, now, nil,
		"晚风收藏家", "🌿", "#759B89")
}

func TestTaskDaoCreate(t *testing.T) {
	Convey("Create 写入任务并回填主键", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*tasks.*").WillReturnResult(sqlmock.NewResult(7, 1))
		row := &model.Task{Title: "买水果", CreatedBy: 3}

		err := NewTaskDao().Create(context.Background(), db, row)

		So(err, ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(7))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestTaskDaoGetByID(t *testing.T) {
	Convey("GetByID 自动排除软删除记录", t, func() {
		db, mock := newDAOTestDB(t)
		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM .*tasks.*deleted_at.*IS NULL.*LIMIT.*").
			WithArgs(uint64(7), 1).
			WillReturnRows(taskRows(now))

		row, err := NewTaskDao().GetByID(context.Background(), db, 7)

		So(err, ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(7))
		So(row.Title, ShouldEqual, "买水果")
		So(row.CreatorNickname, ShouldEqual, "晚风收藏家")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestTaskDaoUpdateAndDelete(t *testing.T) {
	Convey("Update 更新允许编辑的任务字段", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*tasks.* SET .*title.* WHERE id = .* AND .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := NewTaskDao().Update(context.Background(), db, &model.Task{ID: 7, Title: "新标题"})

		So(err, ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("SoftDelete 使用 deleted_at 软删除", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*tasks.* SET .*deleted_at.* WHERE .*id.* AND .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := NewTaskDao().SoftDelete(context.Background(), db, 7)

		So(err, ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("UpdateStatus 只更新任务状态", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*tasks.* SET .*status.*updated_at.* WHERE id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := NewTaskDao().UpdateStatus(context.Background(), db, 7, model.TaskStatusDone)

		So(err, ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestTaskDaoList(t *testing.T) {
	Convey("List 应用筛选、统计总数并分页查询", t, func() {
		db, mock := newDAOTestDB(t)
		now := time.Now()
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*tasks.*").WillReturnRows(
			sqlmock.NewRows([]string{"count"}).AddRow(1),
		)
		mock.ExpectQuery("SELECT .* FROM .*tasks.* ORDER BY CASE WHEN tasks.status = 'done' THEN 1 ELSE 0 END ASC,tasks.created_at DESC,tasks.id DESC LIMIT .* OFFSET .*").
			WillReturnRows(taskRows(now))

		rows, total, err := NewTaskDao().List(context.Background(), db, TaskListQuery{
			Status:    model.TaskStatusTodo,
			Priority:  model.TaskPriorityHigh,
			Category:  "生活",
			Keyword:   "水果",
			CreatedBy: 3,
			Page:      2,
			PageSize:  10,
		})

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(rows[0].ID, ShouldEqual, uint64(7))
		So(rows[0].CreatorNickname, ShouldEqual, "晚风收藏家")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestTaskDaoStats(t *testing.T) {
	Convey("任务汇总一次返回待做和完成数量", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) AS overall_total.*status = 'todo'.*status = 'done'.*FROM .*tasks.*deleted_at.*IS NULL").
			WillReturnRows(sqlmock.NewRows([]string{"overall_total", "todo_count", "done_count"}).
				AddRow(7, 5, 2))

		stats, err := NewTaskDao().Stats(context.Background(), db, 0)

		So(err, ShouldBeNil)
		So(stats.OverallTotal, ShouldEqual, int64(7))
		So(stats.TodoCount, ShouldEqual, int64(5))
		So(stats.DoneCount, ShouldEqual, int64(2))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("任务汇总支持按创建人过滤", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) AS overall_total.*FROM .*tasks.*WHERE created_by = .*deleted_at.*IS NULL").
			WithArgs(uint64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"overall_total", "todo_count", "done_count"}).
				AddRow(4, 3, 1))

		stats, err := NewTaskDao().Stats(context.Background(), db, 3)

		So(err, ShouldBeNil)
		So(stats.OverallTotal, ShouldEqual, int64(4))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
