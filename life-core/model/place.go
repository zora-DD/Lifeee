package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNamePlace = "places"

// Place 对应 places 地点表。
type Place struct {
	ID             uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	Name           string         `gorm:"column:name;type:varchar(120);not null" json:"name"`
	Category       string         `gorm:"column:category;type:varchar(32);not null" json:"category"`
	Icon           *string        `gorm:"column:icon;type:varchar(16)" json:"icon"`
	Visited        bool           `gorm:"column:visited;type:tinyint(1);not null;default:0" json:"visited"`
	VisitCount     uint32         `gorm:"column:visit_count;type:int unsigned;not null;default:0" json:"visitCount"`
	LastVisitedAt  *time.Time     `gorm:"column:last_visited_at;type:date" json:"lastVisitedAt"`
	Feature        *string        `gorm:"column:feature;type:varchar(1000)" json:"feature"`
	Note           *string        `gorm:"column:note;type:varchar(2000)" json:"note"`
	Address        *string        `gorm:"column:address;type:varchar(255)" json:"address"`
	BestTime       *string        `gorm:"column:best_time;type:varchar(120)" json:"bestTime"`
	DistanceKM     uint32         `gorm:"column:distance_km;type:int unsigned;not null;default:0" json:"distanceKm"`
	DriveMinutes   uint32         `gorm:"column:drive_duration_minutes;type:int unsigned;not null;default:0" json:"driveDurationMinutes"`
	GuideURL       *string        `gorm:"column:guide_url;type:varchar(512)" json:"guideUrl"`
	CoverURL       *string        `gorm:"column:cover_url;type:varchar(512)" json:"coverUrl"`
	CoverObjectKey *string        `gorm:"column:cover_object_key;type:varchar(255)" json:"coverObjectKey"`
	Latitude       *string        `gorm:"column:latitude;type:decimal(10,7)" json:"latitude"`
	Longitude      *string        `gorm:"column:longitude;type:decimal(10,7)" json:"longitude"`
	CreatedBy      uint64         `gorm:"column:created_by;type:bigint unsigned;not null" json:"createdBy,string"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3)" json:"-"`
}

func (*Place) TableName() string { return TableNamePlace }
