package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameRestaurant = "restaurants"

const (
	RestaurantPreferenceOkay = "okay"
	RestaurantPreferenceGood = "good"
	RestaurantPreferenceLove = "love"
	RestaurantPreferenceMust = "must"
)

// Restaurant 对应 restaurants 吃喝地点表。
type Restaurant struct {
	ID             uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	Name           string         `gorm:"column:name;type:varchar(120);not null" json:"name"`
	Category       string         `gorm:"column:category;type:varchar(32);not null" json:"category"`
	Cuisine        *string        `gorm:"column:cuisine;type:varchar(64)" json:"cuisine"`
	Preference     string         `gorm:"column:preference;type:varchar(16);not null;default:good" json:"preference"`
	Address        *string        `gorm:"column:address;type:varchar(255)" json:"address"`
	Latitude       *string        `gorm:"column:latitude;type:decimal(10,7)" json:"latitude"`
	Longitude      *string        `gorm:"column:longitude;type:decimal(10,7)" json:"longitude"`
	BusinessHours  *string        `gorm:"column:business_hours;type:varchar(120)" json:"businessHours"`
	Features       *string        `gorm:"column:features;type:varchar(500)" json:"features"`
	CoverURL       *string        `gorm:"column:cover_url;type:varchar(512)" json:"coverUrl"`
	CoverObjectKey *string        `gorm:"column:cover_object_key;type:varchar(255)" json:"coverObjectKey"`
	VisitCount     uint32         `gorm:"column:visit_count;type:int unsigned;not null;default:0" json:"visitCount"`
	TotalSpent     string         `gorm:"column:total_spent;type:decimal(12,2);not null;default:0.00" json:"totalSpent"`
	LastVisitedAt  *time.Time     `gorm:"column:last_visited_at;type:date" json:"lastVisitedAt"`
	LastAmount     *string        `gorm:"column:last_amount;type:decimal(10,2)" json:"lastAmount"`
	LastMeal       *string        `gorm:"column:last_meal;type:varchar(500)" json:"lastMeal"`
	LastNote       *string        `gorm:"column:last_note;type:varchar(1000)" json:"lastNote"`
	CreatedBy      uint64         `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*Restaurant) TableName() string { return TableNameRestaurant }
