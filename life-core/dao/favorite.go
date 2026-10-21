package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"life-core/model"
)

// FavoriteDao 封装 favorites 表操作。
// 幂等由唯一键 (user_id, target_type, target_id) 保证，关联目标有效性由 service 层校验。
type FavoriteDao struct{}

func NewFavoriteDao() *FavoriteDao { return &FavoriteDao{} }

func (d *FavoriteDao) Create(ctx context.Context, db *gorm.DB, row *model.Favorite) error {
	if row == nil {
		return errors.New("dao.FavoriteDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *FavoriteDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*model.Favorite, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.Favorite
	err := db.WithContext(ctx).First(&row, "id = ?", id).Error
	return &row, err
}

func (d *FavoriteDao) GetByUserTarget(
	ctx context.Context,
	db *gorm.DB,
	userID uint64,
	targetType string,
	targetID uint64,
) (*model.Favorite, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.Favorite
	err := db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		First(&row).Error
	return &row, err
}

func (d *FavoriteDao) DeleteByUserTarget(
	ctx context.Context,
	db *gorm.DB,
	userID uint64,
	targetType string,
	targetID uint64,
) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&model.Favorite{}).Error
}

// DeleteByTarget 在业务内容删除事务中清理对应收藏。
func (d *FavoriteDao) DeleteByTarget(ctx context.Context, tx *gorm.DB, targetType string, targetID uint64) error {
	if err := checkDB(tx); err != nil {
		return err
	}
	return tx.WithContext(ctx).Where("target_type = ? AND target_id = ?", targetType, targetID).
		Delete(&model.Favorite{}).Error
}

func (d *FavoriteDao) ListByUser(
	ctx context.Context,
	db *gorm.DB,
	userID uint64,
	targetType string,
	page int,
	pageSize int,
) ([]model.Favorite, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.Favorite{}).Where("user_id = ?", userID)
	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(page, pageSize)
	var rows []model.Favorite
	err := query.Order("created_at DESC").Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
