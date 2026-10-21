package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"life-core/model"
)

type HomeDishDao struct{}

func NewHomeDishDao() *HomeDishDao { return &HomeDishDao{} }

type HomeDishQuery struct {
	Category, Difficulty, Keyword, Status string
	MaxMinutes, Page, PageSize            int
}

func (d *HomeDishDao) Create(ctx context.Context, db *gorm.DB, row *model.HomeDish) error {
	if row == nil {
		return errors.New("nil home dish")
	}
	return db.WithContext(ctx).Create(row).Error
}
func (d *HomeDishDao) Get(ctx context.Context, db *gorm.DB, id uint64) (*model.HomeDish, error) {
	var row model.HomeDish
	err := db.WithContext(ctx).First(&row, id).Error
	return &row, err
}
func (d *HomeDishDao) List(ctx context.Context, db *gorm.DB, q HomeDishQuery) ([]model.HomeDish, int64, error) {
	query := db.WithContext(ctx).Model(&model.HomeDish{})
	if q.Category != "" {
		query = query.Where("category=?", q.Category)
	}
	if q.Difficulty != "" {
		query = query.Where("difficulty=?", q.Difficulty)
	}
	switch q.Status {
	case "wishlist":
		query = query.Where("cook_count = 0")
	case "eaten":
		query = query.Where("cook_count BETWEEN 1 AND 2")
	case "frequent":
		query = query.Where("cook_count >= 3")
	}
	if q.MaxMinutes > 0 {
		query = query.Where("estimated_minutes<=?", q.MaxMinutes)
	}
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR ingredients LIKE ? OR experience LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(q.Page, q.PageSize)
	var rows []model.HomeDish
	err := query.Order("updated_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
func (d *HomeDishDao) Update(ctx context.Context, db *gorm.DB, row *model.HomeDish) error {
	return db.WithContext(ctx).Model(&model.HomeDish{}).Where("id=?", row.ID).Select("name", "category", "ingredients", "steps", "difficulty", "estimated_minutes", "cover_url", "cover_object_key", "experience", "rating").Updates(row).Error
}
func (d *HomeDishDao) Delete(ctx context.Context, db *gorm.DB, id uint64) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("dish_id = ?", id).Delete(&model.HomeCookingRecord{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.HomeDish{}, id).Error
	})
}
func (d *HomeDishDao) ListCandidates(ctx context.Context, db *gorm.DB, q HomeDishQuery) ([]model.HomeDish, error) {
	rows, _, err := d.List(ctx, db, HomeDishQuery{Category: q.Category, Difficulty: q.Difficulty, Keyword: q.Keyword, Status: q.Status, MaxMinutes: q.MaxMinutes, Page: 1, PageSize: 100})
	return rows, err
}
func (d *HomeDishDao) RecordCook(ctx context.Context, db *gorm.DB, dishID, userID uint64, record *model.HomeCookingRecord) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dish model.HomeDish
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dish, dishID).Error; err != nil {
			return err
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		updates := map[string]any{"cook_count": gorm.Expr("cook_count + 1"), "last_cooked_at": record.CookedAt}
		if record.Rating > 0 {
			updates["rating"] = record.Rating
		}
		return tx.Model(&model.HomeDish{}).Where("id=?", dishID).Updates(updates).Error
	})
}
func (d *HomeDishDao) ListRecords(ctx context.Context, db *gorm.DB, dishID uint64, limit int) ([]model.HomeCookingRecord, error) {
	var rows []model.HomeCookingRecord
	err := db.WithContext(ctx).Where("dish_id=?", dishID).Order("cooked_at DESC,id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
func (d *HomeDishDao) ExistsByName(ctx context.Context, db *gorm.DB, name string, exclude uint64) (bool, error) {
	q := db.WithContext(ctx).Model(&model.HomeDish{}).Where("name=?", name)
	if exclude > 0 {
		q = q.Where("id<>?", exclude)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}
func daysSince(last *time.Time, now time.Time) int {
	if last == nil {
		return 365
	}
	days := int(now.Sub(*last).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
