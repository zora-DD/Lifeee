package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"life-core/model"
)

// RestaurantConsumptionDao 封装 restaurant_consumptions 表操作。
// 消费明细增删改后，service 层需在同一事务中重新计算 restaurants 汇总字段。
type RestaurantConsumptionDao struct{}

func NewRestaurantConsumptionDao() *RestaurantConsumptionDao {
	return &RestaurantConsumptionDao{}
}

func (d *RestaurantConsumptionDao) Create(ctx context.Context, db *gorm.DB, row *model.RestaurantConsumption) error {
	if row == nil {
		return errors.New("dao.RestaurantConsumptionDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *RestaurantConsumptionDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*model.RestaurantConsumption, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.RestaurantConsumption
	err := db.WithContext(ctx).First(&row, "id = ?", id).Error
	return &row, err
}

func (d *RestaurantConsumptionDao) Update(ctx context.Context, db *gorm.DB, row *model.RestaurantConsumption) error {
	if row == nil {
		return errors.New("dao.RestaurantConsumptionDao.Update: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.RestaurantConsumption{}).Where("id = ?", row.ID).
		Select("consumed_at", "amount", "meal", "note").Updates(row).Error
}

func (d *RestaurantConsumptionDao) SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Delete(&model.RestaurantConsumption{}, id).Error
}

// SoftDeleteByRestaurant 软删除指定吃喝地点下的全部消费记录。
func (d *RestaurantConsumptionDao) SoftDeleteByRestaurant(ctx context.Context, db *gorm.DB, restaurantID uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Where("restaurant_id = ?", restaurantID).
		Delete(&model.RestaurantConsumption{}).Error
}

func (d *RestaurantConsumptionDao) ListByRestaurant(
	ctx context.Context,
	db *gorm.DB,
	restaurantID uint64,
	page int,
	pageSize int,
) ([]model.RestaurantConsumption, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.RestaurantConsumption{}).
		Where("restaurant_id = ?", restaurantID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(page, pageSize)
	var rows []model.RestaurantConsumption
	err := query.Order("consumed_at DESC").Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

// ListForSummary 返回某店铺全部有效消费明细，供 service 层重算汇总。
func (d *RestaurantConsumptionDao) ListForSummary(ctx context.Context, tx *gorm.DB, restaurantID uint64) ([]model.RestaurantConsumption, error) {
	if err := checkDB(tx); err != nil {
		return nil, err
	}
	var rows []model.RestaurantConsumption
	err := tx.WithContext(ctx).Where("restaurant_id = ?", restaurantID).
		Order("consumed_at DESC").Order("id DESC").Find(&rows).Error
	return rows, err
}
