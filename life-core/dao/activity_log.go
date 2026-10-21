package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"life-core/model"
)

// ActivityLogDao 封装 activity_logs 表操作。
type ActivityLogDao struct{}

func NewActivityLogDao() *ActivityLogDao { return &ActivityLogDao{} }

// ActivityLogWithCreator 是生活记录及其创建人展示资料的查询结果。
type ActivityLogWithCreator struct {
	model.ActivityLog `gorm:"embedded"`
	CreatorNickname   string `gorm:"column:creator_nickname"`
	CreatorEmoji      string `gorm:"column:creator_emoji"`
	CreatorColor      string `gorm:"column:creator_color"`
}

func (d *ActivityLogDao) Create(ctx context.Context, db *gorm.DB, row *model.ActivityLog) error {
	if row == nil {
		return errors.New("dao.ActivityLogDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *ActivityLogDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*ActivityLogWithCreator, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row ActivityLogWithCreator
	err := activityLogQuery(db.WithContext(ctx)).Where("activity_logs.id = ?", id).First(&row).Error
	return &row, err
}

func (d *ActivityLogDao) Update(ctx context.Context, db *gorm.DB, row *model.ActivityLog) error {
	if row == nil {
		return errors.New("dao.ActivityLogDao.Update: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.ActivityLog{}).Where("id = ?", row.ID).
		Select("content", "occurred_at", "duration_minutes", "input_type", "audio_url").Updates(row).Error
}

func (d *ActivityLogDao) SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Delete(&model.ActivityLog{}, id).Error
}

type ActivityLogListQuery struct {
	CreatedBy uint64
	From      *time.Time
	To        *time.Time
	InputType string
	Page      int
	PageSize  int
}

func (d *ActivityLogDao) List(ctx context.Context, db *gorm.DB, in ActivityLogListQuery) ([]ActivityLogWithCreator, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.ActivityLog{})
	if in.CreatedBy != 0 {
		query = query.Where("activity_logs.created_by = ?", in.CreatedBy)
	}
	if in.From != nil {
		query = query.Where("activity_logs.occurred_at >= ?", *in.From)
	}
	if in.To != nil {
		query = query.Where("activity_logs.occurred_at < ?", *in.To)
	}
	if in.InputType != "" {
		query = query.Where("activity_logs.input_type = ?", in.InputType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []ActivityLogWithCreator
	err := activityLogQuery(query).
		Order("activity_logs.occurred_at DESC").Order("activity_logs.id DESC").
		Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func activityLogQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&model.ActivityLog{}).
		Select("activity_logs.*, COALESCE(users.nickname, '') AS creator_nickname, COALESCE(users.avatar_emoji, '') AS creator_emoji, COALESCE(users.avatar_color, '') AS creator_color").
		Joins("LEFT JOIN users ON users.id = activity_logs.created_by")
}

// ListForStats 返回指定时间范围内用于统计的轻量字段。
func (d *ActivityLogDao) ListForStats(
	ctx context.Context,
	db *gorm.DB,
	createdBy uint64,
	from time.Time,
	to time.Time,
) ([]model.ActivityLog, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	query := db.WithContext(ctx).Model(&model.ActivityLog{}).
		Select("occurred_at", "duration_minutes").
		Where("occurred_at >= ? AND occurred_at < ?", from, to)
	if createdBy != 0 {
		query = query.Where("created_by = ?", createdBy)
	}
	var rows []model.ActivityLog
	err := query.Order("occurred_at ASC").Order("id ASC").Find(&rows).Error
	return rows, err
}
