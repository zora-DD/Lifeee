package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"life-core/model"
)

// RestaurantDao 封装 restaurants 表操作。
type RestaurantDao struct{}

func NewRestaurantDao() *RestaurantDao { return &RestaurantDao{} }

func (d *RestaurantDao) Create(ctx context.Context, db *gorm.DB, row *model.Restaurant) error {
	if row == nil {
		return errors.New("dao.RestaurantDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *RestaurantDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*model.Restaurant, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.Restaurant
	err := db.WithContext(ctx).First(&row, "id = ?", id).Error
	return &row, err
}

// ExistsByName 判断是否存在同名吃喝记录；excludeID 用于编辑时排除当前记录。
func (d *RestaurantDao) ExistsByName(ctx context.Context, db *gorm.DB, name string, excludeID uint64) (bool, error) {
	if err := checkDB(db); err != nil {
		return false, err
	}
	query := db.WithContext(ctx).Model(&model.Restaurant{}).Where("name = ?", name)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByIDForUpdate 在事务中锁定吃喝地点，供消费明细与汇总字段同步使用。
func (d *RestaurantDao) GetByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*model.Restaurant, error) {
	if err := checkDB(tx); err != nil {
		return nil, err
	}
	var row model.Restaurant
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return &row, err
}

// UpdateProfile 只更新店铺资料，消费汇总字段通过 UpdateSummary 单独维护。
func (d *RestaurantDao) UpdateProfile(ctx context.Context, db *gorm.DB, row *model.Restaurant) error {
	if row == nil {
		return errors.New("dao.RestaurantDao.UpdateProfile: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.Restaurant{}).Where("id = ?", row.ID).
		Select("name", "category", "cuisine", "preference", "address", "latitude", "longitude", "business_hours", "features", "cover_url", "cover_object_key").
		Updates(row).Error
}

type RestaurantSummary struct {
	VisitCount    uint32
	TotalSpent    string
	LastVisitedAt *time.Time
	LastAmount    *string
	LastMeal      *string
	LastNote      *string
}

// UpdateSummary 应与消费明细变更放在同一个事务中执行。
func (d *RestaurantDao) UpdateSummary(ctx context.Context, tx *gorm.DB, id uint64, summary RestaurantSummary) error {
	if err := checkDB(tx); err != nil {
		return err
	}
	values := map[string]any{
		"visit_count":     summary.VisitCount,
		"total_spent":     summary.TotalSpent,
		"last_visited_at": summary.LastVisitedAt,
		"last_amount":     summary.LastAmount,
		"last_meal":       summary.LastMeal,
		"last_note":       summary.LastNote,
	}
	return tx.WithContext(ctx).Model(&model.Restaurant{}).Where("id = ?", id).Updates(values).Error
}

func (d *RestaurantDao) SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Delete(&model.Restaurant{}, id).Error
}

type RestaurantListQuery struct {
	Category   string
	Preference string
	Keyword    string
	CreatedBy  uint64
	Status     string
	Page       int
	PageSize   int
}

// RestaurantStats 是吃喝列表顶部所需的全量轻量汇总，不加载店铺明细和封面地址。
type RestaurantStats struct {
	OverallTotal    int64  `gorm:"column:overall_total"`
	TotalVisitCount int64  `gorm:"column:total_visit_count"`
	TotalSpent      string `gorm:"column:total_spent"`
}

func (d *RestaurantDao) Stats(ctx context.Context, db *gorm.DB) (RestaurantStats, error) {
	if err := checkDB(db); err != nil {
		return RestaurantStats{}, err
	}
	var stats RestaurantStats
	err := db.WithContext(ctx).Model(&model.Restaurant{}).
		Select("COUNT(*) AS overall_total, COALESCE(SUM(visit_count), 0) AS total_visit_count, COALESCE(SUM(total_spent), 0.00) AS total_spent").
		Scan(&stats).Error
	return stats, err
}

func (d *RestaurantDao) List(ctx context.Context, db *gorm.DB, in RestaurantListQuery) ([]model.Restaurant, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.Restaurant{})
	if in.Category != "" {
		query = query.Where("category = ?", in.Category)
	}
	if in.Preference != "" {
		query = query.Where("preference = ?", in.Preference)
	}
	if in.Keyword != "" {
		keyword := "%" + in.Keyword + "%"
		query = query.Where("name LIKE ? OR cuisine LIKE ? OR address LIKE ?", keyword, keyword, keyword)
	}
	if in.CreatedBy != 0 {
		query = query.Where("created_by = ?", in.CreatedBy)
	}
	switch in.Status {
	case "wishlist":
		query = query.Where("visit_count = 0")
	case "visited":
		query = query.Where("visit_count > 0 AND visit_count < 3")
	case "frequent":
		query = query.Where("visit_count >= 3")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []model.Restaurant
	err := query.Order("last_visited_at IS NULL ASC").Order("last_visited_at DESC").Order("id DESC").
		Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
