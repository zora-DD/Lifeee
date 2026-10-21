package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameTask = "tasks"

const (
	TaskStatusTodo = "todo"
	TaskStatusDone = "done"

	TaskPriorityHigh   = "high"
	TaskPriorityMedium = "medium"
	TaskPriorityLow    = "low"
)

// Task 对应 tasks 任务表。
type Task struct {
	ID          uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	Title       string         `gorm:"column:title;type:varchar(120);not null" json:"title"`
	Description *string        `gorm:"column:description;type:text" json:"description"`
	Status      string         `gorm:"column:status;type:varchar(16);not null;default:todo" json:"status"`
	Priority    string         `gorm:"column:priority;type:varchar(16);not null;default:medium" json:"priority"`
	Category    string         `gorm:"column:category;type:varchar(32);not null;default:其他" json:"category"`
	DueAt       *time.Time     `gorm:"column:due_at;type:datetime(3)" json:"dueAt"`
	CreatedBy   uint64         `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*Task) TableName() string { return TableNameTask }
