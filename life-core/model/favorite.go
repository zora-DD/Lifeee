package model

import "time"

const TableNameFavorite = "favorites"

const (
	FavoriteTargetTask       = "task"
	FavoriteTargetRestaurant = "restaurant"
	FavoriteTargetPlace      = "place"
)

// Favorite 对应 favorites 收藏表。
type Favorite struct {
	ID         uint64    `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	UserID     uint64    `gorm:"column:user_id;type:bigint unsigned;not null" json:"userId,string"`
	TargetType string    `gorm:"column:target_type;type:varchar(16);not null" json:"targetType"`
	TargetID   uint64    `gorm:"column:target_id;type:bigint unsigned;not null" json:"targetId,string"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
}

func (*Favorite) TableName() string { return TableNameFavorite }
