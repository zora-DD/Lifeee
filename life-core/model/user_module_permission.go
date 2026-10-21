package model

import "time"

const TableNameUserModulePermission = "user_module_permissions"

const (
	PermissionModuleTask       = "task"
	PermissionModuleActivity   = "activity_log"
	PermissionModuleBill       = "bill"
	PermissionModuleRestaurant = "restaurant"
	PermissionModulePlace      = "place"

	PermissionLevelNone  = "none"
	PermissionLevelRead  = "read"
	PermissionLevelWrite = "write"
)

// PermissionModules 返回所有可配置业务模块，顺序同时用于管理端展示。
func PermissionModules() []string {
	return []string{
		PermissionModuleTask,
		PermissionModuleActivity,
		PermissionModuleBill,
		PermissionModuleRestaurant,
		PermissionModulePlace,
	}
}

// UserModulePermission 对应用户模块权限表，不使用数据库外键。
type UserModulePermission struct {
	ID        uint64    `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id,string"`
	UserID    uint64    `gorm:"column:user_id;type:bigint unsigned;not null;uniqueIndex:uk_user_module_permissions_user_module" json:"userId,string"`
	Module    string    `gorm:"column:module;type:varchar(32);not null;uniqueIndex:uk_user_module_permissions_user_module" json:"module"`
	Level     string    `gorm:"column:permission_level;type:varchar(16);not null;default:none" json:"level"`
	GrantedBy uint64    `gorm:"column:granted_by;type:bigint unsigned;not null" json:"grantedBy,string"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime:milli" json:"updatedAt"`
}

func (*UserModulePermission) TableName() string { return TableNameUserModulePermission }
