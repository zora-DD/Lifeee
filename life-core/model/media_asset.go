package model

import "time"

const TableNameMediaAsset = "media_assets"

const (
	MediaStatusActive  = "active"
	MediaStatusDeleted = "deleted"

	MediaBizTypeAvatar      = "avatar"
	MediaBizTypeRestaurant  = "restaurant"
	MediaBizTypePlace       = "place"
	MediaBizTypeMemory      = "memory"
	MediaBizTypeActivityLog = "activity_log"
	MediaBizTypeHomeDish    = "home_dish"
	MediaBizTypeSnack       = "snack"
)

// MediaAsset 对应 media_assets 上传文件表。
type MediaAsset struct {
	ID        uint64    `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	UserID    uint64    `gorm:"column:user_id;type:bigint unsigned;not null" json:"userId,string"`
	BizType   string    `gorm:"column:biz_type;type:varchar(32);not null" json:"bizType"`
	ObjectKey string    `gorm:"column:object_key;type:varchar(255);not null" json:"objectKey"`
	FileURL   string    `gorm:"column:file_url;type:varchar(512);not null" json:"fileUrl"`
	MimeType  string    `gorm:"column:mime_type;type:varchar(64);not null" json:"mimeType"`
	FileSize  uint64    `gorm:"column:file_size;type:bigint unsigned;not null" json:"fileSize"`
	Width     *uint32   `gorm:"column:width;type:int unsigned" json:"width"`
	Height    *uint32   `gorm:"column:height;type:int unsigned" json:"height"`
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:active" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
}

func (*MediaAsset) TableName() string { return TableNameMediaAsset }
