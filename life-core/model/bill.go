package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameBill = "bills"

const (
	BillTypeExpense = "expense"
	BillTypeIncome  = "income"

	BillInputTypeText   = "text"
	BillInputTypeVoice  = "voice"
	BillInputTypeManual = "manual"
)

// Bill 对应 bills 公开账单表；created_by 仅用于展示和筛选，不做数据隔离。
type Bill struct {
	ID         uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	Type       string         `gorm:"column:type;type:varchar(16);not null;default:expense" json:"type"`
	Title      string         `gorm:"column:title;type:varchar(120);not null" json:"title"`
	Amount     string         `gorm:"column:amount;type:decimal(12,2);not null" json:"amount"`
	Category   string         `gorm:"column:category;type:varchar(32);not null" json:"category"`
	OccurredAt time.Time      `gorm:"column:occurred_at;type:datetime(3);not null" json:"occurredAt"`
	Note       *string        `gorm:"column:note;type:varchar(1000)" json:"note"`
	InputType  string         `gorm:"column:input_type;type:varchar(16);not null;default:manual" json:"inputType"`
	RawText    *string        `gorm:"column:raw_text;type:varchar(2000)" json:"rawText"`
	CreatedBy  uint64         `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt  time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*Bill) TableName() string { return TableNameBill }
