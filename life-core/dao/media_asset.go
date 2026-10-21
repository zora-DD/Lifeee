package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"life-core/model"
)

// MediaAssetDao 封装 media_assets 表操作。
type MediaAssetDao struct{}

func NewMediaAssetDao() *MediaAssetDao { return &MediaAssetDao{} }

func (d *MediaAssetDao) Create(ctx context.Context, db *gorm.DB, row *model.MediaAsset) error {
	if row == nil {
		return errors.New("dao.MediaAssetDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *MediaAssetDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*model.MediaAsset, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.MediaAsset
	err := db.WithContext(ctx).First(&row, "id = ?", id).Error
	return &row, err
}

func (d *MediaAssetDao) GetByObjectKey(ctx context.Context, db *gorm.DB, objectKey string) (*model.MediaAsset, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.MediaAsset
	err := db.WithContext(ctx).Where("object_key = ?", objectKey).First(&row).Error
	return &row, err
}

func (d *MediaAssetDao) UpdateStatus(ctx context.Context, db *gorm.DB, id uint64, status string) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.MediaAsset{}).Where("id = ?", id).Update("status", status).Error
}

type MediaAssetListQuery struct {
	UserID   uint64
	BizType  string
	Status   string
	Page     int
	PageSize int
}

func (d *MediaAssetDao) List(ctx context.Context, db *gorm.DB, in MediaAssetListQuery) ([]model.MediaAsset, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.MediaAsset{})
	if in.UserID != 0 {
		query = query.Where("user_id = ?", in.UserID)
	}
	if in.BizType != "" {
		query = query.Where("biz_type = ?", in.BizType)
	}
	if in.Status != "" {
		query = query.Where("status = ?", in.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []model.MediaAsset
	err := query.Order("created_at DESC").Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
