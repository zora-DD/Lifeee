package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"life-core/model"
)

// TaskDao 封装 tasks 表操作，普通查询会自动排除软删除数据。
type TaskDao struct{}

func NewTaskDao() *TaskDao { return &TaskDao{} }

// TaskWithCreator 是任务及其创建人展示资料的查询结果。
type TaskWithCreator struct {
	model.Task      `gorm:"embedded"`
	CreatorNickname string `gorm:"column:creator_nickname"`
	CreatorEmoji    string `gorm:"column:creator_emoji"`
	CreatorColor    string `gorm:"column:creator_color"`
}

func (d *TaskDao) Create(ctx context.Context, db *gorm.DB, row *model.Task) error {
	if row == nil {
		return errors.New("dao.TaskDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *TaskDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*TaskWithCreator, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row TaskWithCreator
	err := taskQuery(db.WithContext(ctx)).Where("tasks.id = ?", id).First(&row).Error
	return &row, err
}

func (d *TaskDao) Update(ctx context.Context, db *gorm.DB, row *model.Task) error {
	if row == nil {
		return errors.New("dao.TaskDao.Update: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", row.ID).
		Select("title", "description", "priority", "category", "due_at").Updates(row).Error
}

func (d *TaskDao) UpdateStatus(ctx context.Context, db *gorm.DB, id uint64, status string) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", id).Update("status", status).Error
}

func (d *TaskDao) SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Delete(&model.Task{}, id).Error
}

type TaskListQuery struct {
	Status    string
	Priority  string
	Category  string
	Keyword   string
	CreatedBy uint64
	DueFrom   *time.Time
	DueTo     *time.Time
	Page      int
	PageSize  int
}

// TaskStats 是任务页顶部所需的全量状态汇总。
type TaskStats struct {
	OverallTotal int64 `gorm:"column:overall_total"`
	TodoCount    int64 `gorm:"column:todo_count"`
	DoneCount    int64 `gorm:"column:done_count"`
}

func (d *TaskDao) Stats(ctx context.Context, db *gorm.DB, createdBy uint64) (TaskStats, error) {
	if err := checkDB(db); err != nil {
		return TaskStats{}, err
	}
	var stats TaskStats
	query := db.WithContext(ctx).Model(&model.Task{})
	if createdBy != 0 {
		query = query.Where("created_by = ?", createdBy)
	}
	err := query.Select("COUNT(*) AS overall_total, COALESCE(SUM(CASE WHEN status = 'todo' THEN 1 ELSE 0 END), 0) AS todo_count, COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0) AS done_count").
		Scan(&stats).Error
	return stats, err
}

func (d *TaskDao) List(ctx context.Context, db *gorm.DB, in TaskListQuery) ([]TaskWithCreator, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := applyTaskFilters(db.WithContext(ctx).Model(&model.Task{}), in)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []TaskWithCreator
	err := applyTaskFilters(taskQuery(db.WithContext(ctx)), in).
		Order("CASE WHEN tasks.status = 'done' THEN 1 ELSE 0 END ASC").
		Order("tasks.created_at DESC").Order("tasks.id DESC").
		Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func applyTaskFilters(query *gorm.DB, in TaskListQuery) *gorm.DB {
	if in.Status != "" {
		query = query.Where("tasks.status = ?", in.Status)
	}
	if in.Priority != "" {
		query = query.Where("tasks.priority = ?", in.Priority)
	}
	if in.Category != "" {
		query = query.Where("tasks.category = ?", in.Category)
	}
	if in.Keyword != "" {
		keyword := "%" + in.Keyword + "%"
		query = query.Where("tasks.title LIKE ? OR tasks.description LIKE ?", keyword, keyword)
	}
	if in.CreatedBy != 0 {
		query = query.Where("tasks.created_by = ?", in.CreatedBy)
	}
	if in.DueFrom != nil {
		query = query.Where("tasks.due_at >= ?", *in.DueFrom)
	}
	if in.DueTo != nil {
		query = query.Where("tasks.due_at <= ?", *in.DueTo)
	}
	return query
}

func taskQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&model.Task{}).
		Select("tasks.*, COALESCE(users.nickname, '') AS creator_nickname, COALESCE(users.avatar_emoji, '') AS creator_emoji, COALESCE(users.avatar_color, '') AS creator_color").
		Joins("LEFT JOIN users ON users.id = tasks.created_by")
}
