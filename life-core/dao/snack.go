package dao

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"life-core/model"
)

type SnackDao struct{}

func NewSnackDao() *SnackDao { return &SnackDao{} }

type SnackQuery struct {
	Category, Status, Keyword string
	Page, PageSize            int
}

func (d *SnackDao) Create(ctx context.Context, db *gorm.DB, row *model.Snack) error {
	if row == nil {
		return errors.New("nil snack")
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *SnackDao) Get(ctx context.Context, db *gorm.DB, id uint64) (*model.Snack, error) {
	var row model.Snack
	err := db.WithContext(ctx).First(&row, id).Error
	return &row, err
}

func (d *SnackDao) List(ctx context.Context, db *gorm.DB, query SnackQuery) ([]model.Snack, int64, error) {
	dbQuery := db.WithContext(ctx).Model(&model.Snack{})
	if query.Category != "" {
		dbQuery = dbQuery.Where("category = ?", query.Category)
	}
	if query.Status != "" {
		dbQuery = dbQuery.Where("status = ?", query.Status)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		dbQuery = dbQuery.Where("name LIKE ? OR brand LIKE ? OR purchase_place LIKE ? OR note LIKE ? OR memory_tag LIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(query.Page, query.PageSize)
	var rows []model.Snack
	err := dbQuery.Order("updated_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (d *SnackDao) ListCandidates(ctx context.Context, db *gorm.DB, query SnackQuery) ([]model.Snack, error) {
	rows, _, err := d.List(ctx, db, SnackQuery{Category: query.Category, Status: query.Status, Page: 1, PageSize: 100})
	return rows, err
}

func (d *SnackDao) Update(ctx context.Context, db *gorm.DB, row *model.Snack) error {
	return db.WithContext(ctx).Model(&model.Snack{}).Where("id = ?", row.ID).Select(
		"name", "brand", "purchase_place", "category", "status", "cover_url", "cover_object_key", "note", "memory_tag",
	).Updates(row).Error
}

func (d *SnackDao) Delete(ctx context.Context, db *gorm.DB, id uint64) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("snack_id = ?", id).Delete(&model.SnackTastingRecord{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Snack{}, id).Error
	})
}

func (d *SnackDao) RecordTaste(ctx context.Context, db *gorm.DB, snackID uint64, record *model.SnackTastingRecord) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var snack model.Snack
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&snack, snackID).Error; err != nil {
			return err
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		newCount := snack.TastingCount + 1
		status := snack.Status
		if status != "avoid" {
			if newCount >= 3 {
				status = "frequent"
			} else if status == "wishlist" || status == "" {
				status = "eaten"
			}
		}
		return tx.Model(&model.Snack{}).Where("id = ?", snackID).Updates(map[string]any{
			"tasting_count":  newCount,
			"last_tasted_at": record.TastedAt,
			"status":         status,
		}).Error
	})
}

func (d *SnackDao) ListRecords(ctx context.Context, db *gorm.DB, snackID uint64, limit int) ([]model.SnackTastingRecord, error) {
	var rows []model.SnackTastingRecord
	err := db.WithContext(ctx).Where("snack_id = ?", snackID).Order("tasted_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (d *SnackDao) ExistsByName(ctx context.Context, db *gorm.DB, name, brand string, exclude uint64) (bool, error) {
	query := db.WithContext(ctx).Model(&model.Snack{}).Where("name = ? AND brand = ?", name, brand)
	if exclude > 0 {
		query = query.Where("id <> ?", exclude)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}
