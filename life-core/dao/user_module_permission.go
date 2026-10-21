package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"life-core/model"
)

// UserModulePermissionDao 封装用户模块权限读写。
type UserModulePermissionDao struct{}

func NewUserModulePermissionDao() *UserModulePermissionDao { return &UserModulePermissionDao{} }

func (d *UserModulePermissionDao) ListByUserID(ctx context.Context, db *gorm.DB, userID uint64) ([]model.UserModulePermission, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var rows []model.UserModulePermission
	err := db.WithContext(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&rows).Error
	return rows, err
}

// UpsertAll 写入完整模块权限集合，由唯一键 (user_id, module) 保证幂等。
func (d *UserModulePermissionDao) UpsertAll(ctx context.Context, db *gorm.DB, rows []model.UserModulePermission) error {
	if len(rows) == 0 {
		return errors.New("dao.UserModulePermissionDao.UpsertAll: empty rows")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "module"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"permission_level", "granted_by", "updated_at",
		}),
	}).Create(&rows).Error
}
