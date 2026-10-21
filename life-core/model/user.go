package model

import "time"

const TableNameUser = "users"

const (
	UserStatusPending  = "pending"
	UserStatusApproved = "approved"
	UserStatusRejected = "rejected"
	UserStatusDisabled = "disabled"

	UserRoleUser  = "user"
	UserRoleAdmin = "admin"
)

// User 对应 users 用户表。
type User struct {
	ID              uint64     `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	WxOpenID        string     `gorm:"column:wx_open_id;type:varchar(64);not null;uniqueIndex:uk_users_wx_open_id" json:"wxOpenId"`
	WxUnionID       *string    `gorm:"column:wx_union_id;type:varchar(64)" json:"wxUnionId"`
	Nickname        string     `gorm:"column:nickname;type:varchar(64);not null" json:"nickname"`
	AvatarURL       *string    `gorm:"column:avatar_url;type:varchar(512)" json:"avatarUrl"`
	AvatarEmoji     *string    `gorm:"column:avatar_emoji;type:varchar(16)" json:"avatarEmoji"`
	AvatarColor     *string    `gorm:"column:avatar_color;type:varchar(16)" json:"avatarColor"`
	Signature       *string    `gorm:"column:signature;type:varchar(160)" json:"signature"`
	ApplicationNote *string    `gorm:"column:application_note;type:varchar(200)" json:"applicationNote"`
	Status          string     `gorm:"column:status;type:varchar(16);not null;default:pending" json:"status"`
	Role            string     `gorm:"column:role;type:varchar(16);not null;default:user" json:"role"`
	LastLoginAt     *time.Time `gorm:"column:last_login_at;type:datetime(3)" json:"lastLoginAt"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
}

func (*User) TableName() string { return TableNameUser }
