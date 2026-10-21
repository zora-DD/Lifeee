package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"life-core/model"
)

// LifeSummarySnapshot 是个人生活报告所需的聚合事实。
type LifeSummarySnapshot struct {
	TaskCreatedCount     int64
	TaskDoneCount        int64
	ActivityCount        int64
	ActivityMinutes      int64
	BillCount            int64
	ExpenseTotal         string
	RestaurantVisitCount int64
	VisitedPlaceCount    int64
}

// AISummaryDao 封装 AI 总结事实查询和每日任务总结持久化。
type AISummaryDao struct{}

func NewAISummaryDao() *AISummaryDao { return &AISummaryDao{} }

// GetTaskSummaryByDate 查询指定日期的任务总结。
func (d *AISummaryDao) GetTaskSummaryByDate(ctx context.Context, db *gorm.DB, summaryDate time.Time) (*model.TaskAISummary, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.TaskAISummary
	err := db.WithContext(ctx).Where("summary_date = ?", summaryDate.Format("2006-01-02")).First(&row).Error
	return &row, err
}

// UpsertTaskSummary 按总结日期幂等写入；事实变化或手动刷新时覆盖当天结果。
func (d *AISummaryDao) UpsertTaskSummary(ctx context.Context, db *gorm.DB, row *model.TaskAISummary) error {
	if row == nil {
		return errors.New("dao.AISummaryDao.UpsertTaskSummary: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "summary_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"facts_hash", "payload_json", "ai_generated", "generated_at", "updated_at",
		}),
	}).Create(row).Error
}

// ListTasksForSummary 返回全部未完成任务，以及当天完成的任务。
func (d *AISummaryDao) ListTasksForSummary(ctx context.Context, db *gorm.DB, dayStart, dayEnd time.Time) ([]model.Task, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var rows []model.Task
	err := db.WithContext(ctx).Model(&model.Task{}).
		Select("id", "title", "status", "priority", "due_at", "updated_at").
		Where("status = ? OR (status = ? AND updated_at >= ? AND updated_at < ?)",
			model.TaskStatusTodo, model.TaskStatusDone, dayStart, dayEnd).
		Order("CASE WHEN status = 'done' THEN 1 ELSE 0 END ASC").
		Order("due_at IS NULL ASC").Order("due_at ASC").Order("id DESC").
		Find(&rows).Error
	return rows, err
}

// GetLifeSnapshot 汇总指定用户在时间范围内的任务、生活记录、账单、吃喝和地点事实。
func (d *AISummaryDao) GetLifeSnapshot(
	ctx context.Context,
	db *gorm.DB,
	userID uint64,
	from time.Time,
	to time.Time,
) (*LifeSummarySnapshot, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	snapshot := new(LifeSummarySnapshot)

	taskQuery := db.WithContext(ctx).Model(&model.Task{}).
		Where("created_by = ? AND created_at >= ? AND created_at < ?", userID, from, to)
	if err := taskQuery.Count(&snapshot.TaskCreatedCount).Error; err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Model(&model.Task{}).
		Where("created_by = ? AND status = ? AND updated_at >= ? AND updated_at < ?", userID, model.TaskStatusDone, from, to).
		Count(&snapshot.TaskDoneCount).Error; err != nil {
		return nil, err
	}

	var activity struct {
		Count   int64 `gorm:"column:count"`
		Minutes int64 `gorm:"column:minutes"`
	}
	if err := db.WithContext(ctx).Model(&model.ActivityLog{}).
		Select("COUNT(*) AS count, COALESCE(SUM(duration_minutes), 0) AS minutes").
		Where("created_by = ? AND occurred_at >= ? AND occurred_at < ?", userID, from, to).
		Scan(&activity).Error; err != nil {
		return nil, err
	}
	snapshot.ActivityCount = activity.Count
	snapshot.ActivityMinutes = activity.Minutes

	var bill struct {
		Count  int64  `gorm:"column:count"`
		Amount string `gorm:"column:amount"`
	}
	if err := db.WithContext(ctx).Model(&model.Bill{}).
		Select("COUNT(*) AS count, CAST(COALESCE(SUM(amount), 0) AS CHAR) AS amount").
		Where("created_by = ? AND type = ? AND occurred_at >= ? AND occurred_at < ?",
			userID, model.BillTypeExpense, from, to).
		Scan(&bill).Error; err != nil {
		return nil, err
	}
	snapshot.BillCount = bill.Count
	snapshot.ExpenseTotal = bill.Amount
	if snapshot.ExpenseTotal == "" {
		snapshot.ExpenseTotal = "0.00"
	}

	if err := db.WithContext(ctx).Model(&model.RestaurantConsumption{}).
		Where("consumed_by = ? AND consumed_at >= ? AND consumed_at < ?", userID, from, to).
		Count(&snapshot.RestaurantVisitCount).Error; err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Model(&model.Place{}).
		Where("created_by = ? AND visited = ? AND last_visited_at >= ? AND last_visited_at < ?", userID, true, from, to).
		Count(&snapshot.VisitedPlaceCount).Error; err != nil {
		return nil, err
	}
	return snapshot, nil
}
