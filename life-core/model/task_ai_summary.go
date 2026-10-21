package model

import "time"

const TableNameTaskAISummary = "task_ai_summaries"

// TaskAISummary 保存每日公开任务的 AI 总结，同一天始终只有一条记录。
type TaskAISummary struct {
	ID          uint64    `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	SummaryDate time.Time `gorm:"column:summary_date;type:date;not null;uniqueIndex:uk_task_ai_summaries_date" json:"summaryDate"`
	FactsHash   string    `gorm:"column:facts_hash;type:char(16);not null" json:"factsHash"`
	PayloadJSON []byte    `gorm:"column:payload_json;type:json;not null" json:"payloadJson"`
	AIGenerated bool      `gorm:"column:ai_generated;type:tinyint(1);not null;default:0" json:"aiGenerated"`
	GeneratedAt time.Time `gorm:"column:generated_at;type:datetime(3);not null" json:"generatedAt"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
}

func (*TaskAISummary) TableName() string { return TableNameTaskAISummary }
