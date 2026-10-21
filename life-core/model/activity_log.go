package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameActivityLog = "activity_logs"

const (
	ActivityInputTypeText  = "text"
	ActivityInputTypeVoice = "voice"
)

// ActivityLog 对应 activity_logs 生活流水记录表。
type ActivityLog struct {
	ID              uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	Content         string         `gorm:"column:content;type:varchar(2000);not null" json:"content"`
	OccurredAt      time.Time      `gorm:"column:occurred_at;type:datetime(3);not null" json:"occurredAt"`
	DurationMinutes *uint16        `gorm:"column:duration_minutes;type:smallint unsigned" json:"durationMinutes"`
	InputType       string         `gorm:"column:input_type;type:varchar(16);not null;default:text" json:"inputType"`
	AudioURL        *string        `gorm:"column:audio_url;type:varchar(512)" json:"audioUrl"`
	CreatedBy       uint64         `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*ActivityLog) TableName() string { return TableNameActivityLog }
