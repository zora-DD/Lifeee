package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	TableNameHomeDish          = "home_dishes"
	TableNameHomeCookingRecord = "home_cooking_records"
)

type HomeDish struct {
	ID               uint64         `gorm:"column:id;primaryKey;autoIncrement"`
	Name             string         `gorm:"column:name;type:varchar(120);not null"`
	Category         string         `gorm:"column:category;type:varchar(32);not null"`
	Ingredients      string         `gorm:"column:ingredients;type:varchar(1000);not null;default:''"`
	Steps            string         `gorm:"column:steps;type:text;not null"`
	Difficulty       string         `gorm:"column:difficulty;type:varchar(16);not null;default:normal"`
	EstimatedMinutes uint32         `gorm:"column:estimated_minutes;type:int unsigned;not null;default:0"`
	CoverURL         *string        `gorm:"column:cover_url;type:varchar(512)"`
	CoverObjectKey   *string        `gorm:"column:cover_object_key;type:varchar(255)"`
	Experience       *string        `gorm:"column:experience;type:varchar(2000)"`
	Rating           uint32         `gorm:"column:rating;type:tinyint unsigned;not null;default:0"`
	CookCount        uint32         `gorm:"column:cook_count;type:int unsigned;not null;default:0"`
	LastCookedAt     *time.Time     `gorm:"column:last_cooked_at;type:date"`
	CreatedBy        uint64         `gorm:"column:created_by;type:bigint unsigned;not null"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);index"`
}

func (*HomeDish) TableName() string { return TableNameHomeDish }

type HomeCookingRecord struct {
	ID              uint64         `gorm:"column:id;primaryKey;autoIncrement"`
	DishID          uint64         `gorm:"column:dish_id;type:bigint unsigned;not null;index"`
	CookedBy        uint64         `gorm:"column:cooked_by;type:bigint unsigned;not null;index"`
	CookedAt        time.Time      `gorm:"column:cooked_at;type:date;not null;index"`
	DurationMinutes uint32         `gorm:"column:duration_minutes;type:int unsigned;not null;default:0"`
	Rating          uint32         `gorm:"column:rating;type:tinyint unsigned;not null;default:0"`
	Note            *string        `gorm:"column:note;type:varchar(1000)"`
	PhotoURL        *string        `gorm:"column:photo_url;type:varchar(512)"`
	PhotoObjectKey  *string        `gorm:"column:photo_object_key;type:varchar(255)"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);index"`
}

func (*HomeCookingRecord) TableName() string { return TableNameHomeCookingRecord }
