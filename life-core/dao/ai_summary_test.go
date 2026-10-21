package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"life-core/model"
)

func TestAISummaryDaoListTasksForSummary(t *testing.T) {
	db, mock := newDAOTestDB(t)
	dayStart := time.Date(2026, 9, 14, 0, 0, 0, 0, time.Local)
	dayEnd := dayStart.AddDate(0, 0, 1)
	mock.ExpectQuery("SELECT .* FROM .*tasks.*status.*updated_at.*ORDER BY.*").
		WithArgs(model.TaskStatusTodo, model.TaskStatusDone, dayStart, dayEnd).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "status", "priority", "due_at", "updated_at"}).
			AddRow(7, "买水果", model.TaskStatusTodo, model.TaskPriorityHigh, dayStart.Add(time.Hour), dayStart))

	rows, err := NewAISummaryDao().ListTasksForSummary(context.Background(), db, dayStart, dayEnd)
	if err != nil {
		t.Fatalf("ListTasksForSummary() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ID != 7 {
		t.Fatalf("ListTasksForSummary() rows = %+v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAISummaryDaoGetLifeSnapshot(t *testing.T) {
	db, mock := newDAOTestDB(t)
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	to := from.AddDate(0, 1, 0)
	userID := uint64(9)

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*tasks.*created_by.*created_at.*").
		WithArgs(userID, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*tasks.*created_by.*status.*updated_at.*").
		WithArgs(userID, model.TaskStatusDone, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(6))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) AS count, COALESCE\\(SUM\\(duration_minutes\\), 0\\) AS minutes FROM .*activity_logs.*").
		WithArgs(userID, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count", "minutes"}).AddRow(12, 320))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) AS count, CAST\\(COALESCE\\(SUM\\(amount\\), 0\\) AS CHAR\\) AS amount FROM .*bills.*").
		WithArgs(userID, model.BillTypeExpense, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count", "amount"}).AddRow(9, "1280.50"))
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*restaurant_consumptions.*").
		WithArgs(userID, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*places.*").
		WithArgs(userID, true, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	snapshot, err := NewAISummaryDao().GetLifeSnapshot(context.Background(), db, userID, from, to)
	if err != nil {
		t.Fatalf("GetLifeSnapshot() error = %v", err)
	}
	if snapshot.TaskCreatedCount != 8 || snapshot.TaskDoneCount != 6 || snapshot.ActivityCount != 12 || snapshot.ActivityMinutes != 320 || snapshot.BillCount != 9 || snapshot.ExpenseTotal != "1280.50" || snapshot.RestaurantVisitCount != 3 || snapshot.VisitedPlaceCount != 2 {
		t.Fatalf("GetLifeSnapshot() = %+v", snapshot)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAISummaryDaoTaskSummaryPersistence(t *testing.T) {
	db, mock := newDAOTestDB(t)
	ctx := context.Background()
	day := time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local)
	payload := []byte(`{"date":"2026-09-16","summary":"今日总结"}`)

	mock.ExpectQuery("SELECT .* FROM .*task_ai_summaries.*summary_date.*LIMIT.*").
		WithArgs("2026-09-16", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "summary_date", "facts_hash", "payload_json", "ai_generated", "generated_at"}).
			AddRow(1, day, "0123456789abcdef", payload, true, day.Add(6*time.Hour)))
	row, err := NewAISummaryDao().GetTaskSummaryByDate(ctx, db, day)
	if err != nil || row.FactsHash != "0123456789abcdef" {
		t.Fatalf("GetTaskSummaryByDate() row=%+v error=%v", row, err)
	}

	mock.ExpectExec("INSERT INTO .*task_ai_summaries.*ON DUPLICATE KEY UPDATE.*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	err = NewAISummaryDao().UpsertTaskSummary(ctx, db, &model.TaskAISummary{
		SummaryDate: day, FactsHash: "fedcba9876543210", PayloadJSON: payload, GeneratedAt: day.Add(7 * time.Hour),
	})
	if err != nil {
		t.Fatalf("UpsertTaskSummary() error=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAISummaryDaoRejectsNilDB(t *testing.T) {
	dao := NewAISummaryDao()
	if _, err := dao.ListTasksForSummary(context.Background(), nil, time.Time{}, time.Time{}); err == nil {
		t.Fatal("nil DB 应返回错误")
	}
	if _, err := dao.GetLifeSnapshot(context.Background(), nil, 1, time.Time{}, time.Time{}); err == nil {
		t.Fatal("nil DB 应返回错误")
	}
	if _, err := dao.GetTaskSummaryByDate(context.Background(), nil, time.Time{}); err == nil {
		t.Fatal("nil DB 应返回错误")
	}
	if err := dao.UpsertTaskSummary(context.Background(), nil, &model.TaskAISummary{}); err == nil {
		t.Fatal("nil DB 应返回错误")
	}
	if err := dao.UpsertTaskSummary(context.Background(), new(gorm.DB), nil); err == nil {
		t.Fatal("nil row 应返回错误")
	}
}
