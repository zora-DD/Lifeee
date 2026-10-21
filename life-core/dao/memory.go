package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"life-core/model"
)

// MemoryDao 封装 memories 表操作。
type MemoryDao struct{}

func NewMemoryDao() *MemoryDao { return &MemoryDao{} }

func (d *MemoryDao) Create(ctx context.Context, db *gorm.DB, row *model.Memory) error {
	if row == nil {
		return errors.New("dao.MemoryDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *MemoryDao) GetBySourceKey(ctx context.Context, db *gorm.DB, sourceKey string) (*model.Memory, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.Memory
	err := db.WithContext(ctx).Where("source_key = ?", sourceKey).First(&row).Error
	return &row, err
}

type MemoryListQuery struct {
	Year     int
	Page     int
	PageSize int
}

func (d *MemoryDao) List(ctx context.Context, db *gorm.DB, in MemoryListQuery) ([]model.Memory, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.Memory{})
	if in.Year > 0 {
		query = query.Where("YEAR(memory_date) = ?", in.Year)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []model.Memory
	err := query.Order("memory_date DESC").Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (d *MemoryDao) ListYears(ctx context.Context, db *gorm.DB) ([]int, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var years []int
	err := db.WithContext(ctx).Model(&model.Memory{}).
		Distinct("YEAR(memory_date)").Order("YEAR(memory_date) DESC").Pluck("YEAR(memory_date)", &years).Error
	return years, err
}

// MemoryPhotoDao 封装 memory_photos 表操作。
type MemoryPhotoDao struct{}

func NewMemoryPhotoDao() *MemoryPhotoDao { return &MemoryPhotoDao{} }

func (d *MemoryPhotoDao) CreateBatch(ctx context.Context, db *gorm.DB, rows []model.MemoryPhoto) error {
	if len(rows) == 0 {
		return errors.New("dao.MemoryPhotoDao.CreateBatch: empty rows")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(&rows).Error
}

func (d *MemoryPhotoDao) ListByMemoryIDs(ctx context.Context, db *gorm.DB, memoryIDs []uint64) ([]model.MemoryPhoto, error) {
	if len(memoryIDs) == 0 {
		return []model.MemoryPhoto{}, nil
	}
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var rows []model.MemoryPhoto
	err := db.WithContext(ctx).Where("memory_id IN ?", memoryIDs).
		Order("memory_id ASC").Order("sort_order ASC").Order("id ASC").Find(&rows).Error
	return rows, err
}
