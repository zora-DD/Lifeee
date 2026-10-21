package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	TableNameMemory      = "memories"
	TableNameMemoryPhoto = "memory_photos"

	MemoryDatePrecisionDay   = "day"
	MemoryDatePrecisionMonth = "month"
	MemoryDatePrecisionYear  = "year"
)

// Memory 对应 memories 时光记录表。
type Memory struct {
	ID            uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	MemoryDate    time.Time      `gorm:"column:memory_date;type:date;not null" json:"memoryDate"`
	DatePrecision string         `gorm:"column:date_precision;type:varchar(8);not null;default:day" json:"datePrecision"`
	Place         *string        `gorm:"column:place;type:varchar(255)" json:"place"`
	Title         string         `gorm:"column:title;type:varchar(160);not null" json:"title"`
	Description   *string        `gorm:"column:description;type:varchar(1000)" json:"description"`
	SourceKey     *string        `gorm:"column:source_key;type:varchar(96);uniqueIndex:uk_memories_source_key" json:"sourceKey"`
	CreatedBy     uint64         `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt     time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*Memory) TableName() string { return TableNameMemory }

// MemoryPhoto 对应 memory_photos 时光图片表，通过业务层逻辑关联 memories 和 media_assets。
type MemoryPhoto struct {
	ID           uint64    `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	MemoryID     uint64    `gorm:"column:memory_id;type:bigint unsigned;not null" json:"memoryId,string"`
	MediaAssetID uint64    `gorm:"column:media_asset_id;type:bigint unsigned;not null" json:"mediaAssetId,string"`
	ObjectKey    string    `gorm:"column:object_key;type:varchar(255);not null" json:"objectKey"`
	SourceName   *string   `gorm:"column:source_name;type:varchar(255)" json:"sourceName"`
	AltText      *string   `gorm:"column:alt_text;type:varchar(500)" json:"altText"`
	Width        uint32    `gorm:"column:width;type:int unsigned;not null;default:0" json:"width"`
	Height       uint32    `gorm:"column:height;type:int unsigned;not null;default:0" json:"height"`
	SortOrder    uint32    `gorm:"column:sort_order;type:int unsigned;not null;default:0" json:"sortOrder"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
}

func (*MemoryPhoto) TableName() string { return TableNameMemoryPhoto }
