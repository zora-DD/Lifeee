package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"life-core/model"
)

// PlaceDao 封装 places 表操作。
type PlaceDao struct{}

func NewPlaceDao() *PlaceDao { return &PlaceDao{} }

func (d *PlaceDao) Create(ctx context.Context, db *gorm.DB, row *model.Place) error {
	if row == nil {
		return errors.New("dao.PlaceDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *PlaceDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*model.Place, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.Place
	err := db.WithContext(ctx).First(&row, "id = ?", id).Error
	return &row, err
}

func (d *PlaceDao) Update(ctx context.Context, db *gorm.DB, row *model.Place) error {
	if row == nil {
		return errors.New("dao.PlaceDao.Update: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.Place{}).Where("id = ?", row.ID).
		Select("name", "category", "icon", "visited", "visit_count", "last_visited_at", "feature", "note", "address", "best_time", "distance_km", "drive_duration_minutes", "guide_url", "cover_url", "cover_object_key", "latitude", "longitude").
		Updates(row).Error
}

func (d *PlaceDao) SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Delete(&model.Place{}, id).Error
}

type PlaceListQuery struct {
	Category  string
	Visited   *bool
	Keyword   string
	CreatedBy uint64
	Page      int
	PageSize  int
}

// PlaceStats 是地点列表顶部所需的全量状态汇总。
type PlaceStats struct {
	OverallTotal  int64 `gorm:"column:overall_total"`
	VisitedCount  int64 `gorm:"column:visited_count"`
	WishlistCount int64 `gorm:"column:wishlist_count"`
}

func (d *PlaceDao) Stats(ctx context.Context, db *gorm.DB) (PlaceStats, error) {
	if err := checkDB(db); err != nil {
		return PlaceStats{}, err
	}
	var stats PlaceStats
	err := db.WithContext(ctx).Model(&model.Place{}).
		Select("COUNT(*) AS overall_total, COALESCE(SUM(CASE WHEN visited = 1 THEN 1 ELSE 0 END), 0) AS visited_count, COALESCE(SUM(CASE WHEN visited = 0 THEN 1 ELSE 0 END), 0) AS wishlist_count").
		Scan(&stats).Error
	return stats, err
}

func (d *PlaceDao) List(ctx context.Context, db *gorm.DB, in PlaceListQuery) ([]model.Place, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.Place{})
	if in.Category != "" {
		query = query.Where("category = ?", in.Category)
	}
	if in.Visited != nil {
		query = query.Where("visited = ?", *in.Visited)
	}
	if in.Keyword != "" {
		keyword := "%" + in.Keyword + "%"
		query = query.Where("name LIKE ? OR feature LIKE ? OR address LIKE ?", keyword, keyword, keyword)
	}
	if in.CreatedBy != 0 {
		query = query.Where("created_by = ?", in.CreatedBy)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []model.Place
	err := query.Order("visited DESC").Order("last_visited_at DESC").Order("created_at DESC").Order("id DESC").
		Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
