package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	TableNameSnack              = "snacks"
	TableNameSnackTastingRecord = "snack_tasting_records"
)

type Snack struct {
	ID             uint64         `gorm:"column:id;primaryKey;autoIncrement"`
	Name           string         `gorm:"column:name;type:varchar(120);not null"`
	Brand          string         `gorm:"column:brand;type:varchar(120);not null;default:''"`
	PurchasePlace  string         `gorm:"column:purchase_place;type:varchar(255);not null;default:''"`
	Category       string         `gorm:"column:category;type:varchar(32);not null"`
	Status         string         `gorm:"column:status;type:varchar(16);not null;default:wishlist"`
	CoverURL       *string        `gorm:"column:cover_url;type:varchar(512)"`
	CoverObjectKey *string        `gorm:"column:cover_object_key;type:varchar(255)"`
	Note           *string        `gorm:"column:note;type:varchar(2000)"`
	MemoryTag      *string        `gorm:"column:memory_tag;type:varchar(64)"`
	TastingCount   uint32         `gorm:"column:tasting_count;type:int unsigned;not null;default:0"`
	LastTastedAt   *time.Time     `gorm:"column:last_tasted_at;type:date"`
	CreatedBy      uint64         `gorm:"column:created_by;type:bigint unsigned;not null"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);index"`
}

func (*Snack) TableName() string { return TableNameSnack }

type SnackTastingRecord struct {
	ID        uint64         `gorm:"column:id;primaryKey;autoIncrement"`
	SnackID   uint64         `gorm:"column:snack_id;type:bigint unsigned;not null;index"`
	TastedBy  uint64         `gorm:"column:tasted_by;type:bigint unsigned;not null;index"`
	TastedAt  time.Time      `gorm:"column:tasted_at;type:date;not null;index"`
	Note      *string        `gorm:"column:note;type:varchar(1000)"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);index"`
}

func (*SnackTastingRecord) TableName() string { return TableNameSnackTastingRecord }
