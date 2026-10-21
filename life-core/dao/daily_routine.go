package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"life-core/model"
)

// DailyRoutineDao 封装个人每日事项和打卡记录操作。
type DailyRoutineDao struct{}

func NewDailyRoutineDao() *DailyRoutineDao { return &DailyRoutineDao{} }

func (d *DailyRoutineDao) Create(ctx context.Context, db *gorm.DB, row *model.DailyRoutine) error {
	if row == nil {
		return errors.New("dao.DailyRoutineDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *DailyRoutineDao) GetByIDAndUser(ctx context.Context, db *gorm.DB, id, userID uint64) (*model.DailyRoutine, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.DailyRoutine
	err := db.WithContext(ctx).Where("id = ? AND created_by = ?", id, userID).First(&row).Error
	return &row, err
}

func (d *DailyRoutineDao) ListByUser(ctx context.Context, db *gorm.DB, userID uint64, includeDisabled bool) ([]model.DailyRoutine, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	query := db.WithContext(ctx).Where("created_by = ?", userID)
	if !includeDisabled {
		query = query.Where("enabled = ?", true)
	}
	var rows []model.DailyRoutine
	err := query.Order("time_of_day ASC").Order("id ASC").Find(&rows).Error
	return rows, err
}

func (d *DailyRoutineDao) Update(ctx context.Context, db *gorm.DB, row *model.DailyRoutine) error {
	if row == nil {
		return errors.New("dao.DailyRoutineDao.Update: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	result := db.WithContext(ctx).Model(&model.DailyRoutine{}).
		Where("id = ? AND created_by = ?", row.ID, row.CreatedBy).
		Select("title", "time_of_day", "repeat_unit", "repeat_interval", "start_date", "enabled").Updates(row)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *DailyRoutineDao) SoftDelete(ctx context.Context, db *gorm.DB, id, userID uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	result := db.WithContext(ctx).Where("id = ? AND created_by = ?", id, userID).Delete(&model.DailyRoutine{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *DailyRoutineDao) ListCheckins(
	ctx context.Context,
	db *gorm.DB,
	userID uint64,
	routineIDs []uint64,
	from, to time.Time,
) ([]model.DailyCheckin, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	if len(routineIDs) == 0 {
		return []model.DailyCheckin{}, nil
	}
	var rows []model.DailyCheckin
	query := db.WithContext(ctx).
		Where("created_by = ? AND routine_id IN ? AND checkin_date <= ?", userID, routineIDs, to)
	if !from.IsZero() {
		query = query.Where("checkin_date >= ?", from)
	}
	err := query.Order("checkin_date ASC").Find(&rows).Error
	return rows, err
}

func (d *DailyRoutineDao) UpsertCheckin(ctx context.Context, db *gorm.DB, row *model.DailyCheckin) error {
	if row == nil {
		return errors.New("dao.DailyRoutineDao.UpsertCheckin: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "created_by"}, {Name: "routine_id"}, {Name: "checkin_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"completed", "checked_at", "updated_at"}),
	}).Create(row).Error
}
