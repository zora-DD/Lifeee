package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameRestaurantConsumption = "restaurant_consumptions"

// RestaurantConsumption 对应 restaurant_consumptions 消费记录表。
type RestaurantConsumption struct {
	ID           uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	RestaurantID uint64         `gorm:"column:restaurant_id;type:bigint unsigned;not null" json:"restaurantId,string"`
	ConsumedBy   uint64         `gorm:"column:consumed_by;type:bigint unsigned;not null" json:"consumedBy,string"`
	ConsumedAt   time.Time      `gorm:"column:consumed_at;type:date;not null" json:"consumedAt"`
	Amount       *string        `gorm:"column:amount;type:decimal(10,2)" json:"amount"`
	Meal         *string        `gorm:"column:meal;type:varchar(500)" json:"meal"`
	Note         *string        `gorm:"column:note;type:varchar(1000)" json:"note"`
	CreatedAt    time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*RestaurantConsumption) TableName() string { return TableNameRestaurantConsumption }
