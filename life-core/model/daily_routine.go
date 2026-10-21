package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	TableNameDailyRoutine = "daily_routines"
	TableNameDailyCheckin = "daily_checkins"
)

// DailyRoutine 保存个人每日事项模板，不生成普通任务记录。
type DailyRoutine struct {
	ID             uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	Title          string         `gorm:"column:title;type:varchar(120);not null" json:"title"`
	TimeOfDay      string         `gorm:"column:time_of_day;type:char(5);not null" json:"timeOfDay"`
	RepeatUnit     string         `gorm:"column:repeat_unit;type:varchar(16);not null;default:day" json:"repeatUnit"`
	RepeatInterval int32          `gorm:"column:repeat_interval;type:int unsigned;not null;default:1" json:"repeatInterval"`
	StartDate      *time.Time     `gorm:"column:start_date;type:date" json:"startDate"`
	Enabled        bool           `gorm:"column:enabled;type:tinyint(1);not null;default:1" json:"enabled"`
	CreatedBy      uint64         `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*DailyRoutine) TableName() string { return TableNameDailyRoutine }

// DailyCheckin 保存某个每日事项在某一天的完成状态。
type DailyCheckin struct {
	ID          uint64     `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	RoutineID   uint64     `gorm:"column:routine_id;type:bigint unsigned;not null" json:"routineId,string"`
	CheckinDate time.Time  `gorm:"column:checkin_date;type:date;not null" json:"checkinDate"`
	Completed   bool       `gorm:"column:completed;type:tinyint(1);not null;default:0" json:"completed"`
	CheckedAt   *time.Time `gorm:"column:checked_at;type:datetime(3)" json:"checkedAt"`
	CreatedBy   uint64     `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
}

func (*DailyCheckin) TableName() string { return TableNameDailyCheckin }
