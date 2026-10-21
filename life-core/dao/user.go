package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"life-core/model"
)

// UserDao 封装 users 表操作。
type UserDao struct{}

func NewUserDao() *UserDao { return &UserDao{} }

func (d *UserDao) Create(ctx context.Context, db *gorm.DB, row *model.User) error {
	if row == nil {
		return errors.New("dao.UserDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *UserDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*model.User, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.User
	err := db.WithContext(ctx).First(&row, "id = ?", id).Error
	return &row, err
}

func (d *UserDao) GetByOpenID(ctx context.Context, db *gorm.DB, openID string) (*model.User, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.User
	err := db.WithContext(ctx).Where("wx_open_id = ?", openID).First(&row).Error
	return &row, err
}

func (d *UserDao) GetByUnionID(ctx context.Context, db *gorm.DB, unionID string) (*model.User, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row model.User
	err := db.WithContext(ctx).Where("wx_union_id = ?", unionID).First(&row).Error
	return &row, err
}

// Count 返回用户总数，用于首次启动时安全地引导管理员账号。
func (d *UserDao) Count(ctx context.Context, db *gorm.DB) (int64, error) {
	if err := checkDB(db); err != nil {
		return 0, err
	}
	var count int64
	err := db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}

// UpdateProfile 更新用户可编辑资料，不修改 OpenID、状态和角色。
func (d *UserDao) UpdateProfile(ctx context.Context, db *gorm.DB, row *model.User) error {
	if row == nil {
		return errors.New("dao.UserDao.UpdateProfile: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.User{}).Where("id = ?", row.ID).
		Select("wx_union_id", "nickname", "avatar_url", "avatar_emoji", "avatar_color", "signature").
		Updates(row).Error
}

func (d *UserDao) UpdateStatus(ctx context.Context, db *gorm.DB, id uint64, status string) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

func (d *UserDao) UpdateApplicationNote(ctx context.Context, db *gorm.DB, id uint64, note string) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("application_note", note).Error
}

func (d *UserDao) UpdateRole(ctx context.Context, db *gorm.DB, id uint64, role string) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("role", role).Error
}

func (d *UserDao) UpdateLastLoginAt(ctx context.Context, db *gorm.DB, id uint64, at time.Time) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("last_login_at", at).Error
}

type UserListQuery struct {
	Status   string
	Role     string
	Keyword  string
	Page     int
	PageSize int
}

func (d *UserDao) List(ctx context.Context, db *gorm.DB, in UserListQuery) ([]model.User, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&model.User{})
	if in.Status != "" {
		query = query.Where("status = ?", in.Status)
	}
	if in.Role != "" {
		query = query.Where("role = ?", in.Role)
	}
	if in.Keyword != "" {
		query = query.Where("nickname LIKE ?", "%"+in.Keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []model.User
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
