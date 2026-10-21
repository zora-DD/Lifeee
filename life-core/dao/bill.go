package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"life-core/model"
)

// BillWithCreator 是账单及其记账人展示信息的查询结果。
type BillWithCreator struct {
	model.Bill      `gorm:"embedded"`
	CreatorNickname string `gorm:"column:creator_nickname"`
	CreatorEmoji    string `gorm:"column:creator_emoji"`
	CreatorColor    string `gorm:"column:creator_color"`
}

// BillCreator 是存在有效账单的记账人筛选项。
type BillCreator struct {
	ID          uint64 `gorm:"column:id"`
	Nickname    string `gorm:"column:nickname"`
	AvatarEmoji string `gorm:"column:avatar_emoji"`
	AvatarColor string `gorm:"column:avatar_color"`
}

type BillListQuery struct {
	Type      string
	Category  string
	Keyword   string
	CreatedBy uint64
	From      *time.Time
	To        *time.Time
	Page      int
	PageSize  int
}

type BillCategoryAggregate struct {
	Type     string `gorm:"column:type"`
	Category string `gorm:"column:category"`
	Amount   string `gorm:"column:amount"`
	Count    int64  `gorm:"column:count"`
}

type BillMonthlyAggregate struct {
	Month  int    `gorm:"column:month"`
	Type   string `gorm:"column:type"`
	Amount string `gorm:"column:amount"`
	Count  int64  `gorm:"column:count"`
}

type BillUserAggregate struct {
	CreatedBy   uint64 `gorm:"column:created_by"`
	Nickname    string `gorm:"column:nickname"`
	AvatarEmoji string `gorm:"column:avatar_emoji"`
	AvatarColor string `gorm:"column:avatar_color"`
	Type        string `gorm:"column:type"`
	Amount      string `gorm:"column:amount"`
	Count       int64  `gorm:"column:count"`
}

// BillDao 封装 bills 表操作。账单为登录用户间公开数据，created_by 仅参与筛选。
type BillDao struct{}

func NewBillDao() *BillDao { return &BillDao{} }

func (d *BillDao) Create(ctx context.Context, db *gorm.DB, row *model.Bill) error {
	if row == nil {
		return errors.New("dao.BillDao.Create: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(row).Error
}

func (d *BillDao) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*BillWithCreator, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var row BillWithCreator
	err := billQuery(db.WithContext(ctx)).Where("bills.id = ?", id).First(&row).Error
	return &row, err
}

func (d *BillDao) Update(ctx context.Context, db *gorm.DB, row *model.Bill) error {
	if row == nil {
		return errors.New("dao.BillDao.Update: nil row")
	}
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.Bill{}).Where("id = ?", row.ID).
		Select("type", "title", "amount", "category", "occurred_at", "note", "input_type", "raw_text").
		Updates(row).Error
}

func (d *BillDao) SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error {
	if err := checkDB(db); err != nil {
		return err
	}
	return db.WithContext(ctx).Delete(&model.Bill{}, id).Error
}

func (d *BillDao) List(ctx context.Context, db *gorm.DB, in BillListQuery) ([]BillWithCreator, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	query := applyBillFilters(db.WithContext(ctx).Model(&model.Bill{}), in)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(in.Page, in.PageSize)
	var rows []BillWithCreator
	err := billQuery(applyBillFilters(db.WithContext(ctx), in)).
		Order("bills.occurred_at DESC").Order("bills.id DESC").
		Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (d *BillDao) ListCreators(ctx context.Context, db *gorm.DB) ([]BillCreator, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var rows []BillCreator
	err := db.WithContext(ctx).Table("bills").
		Select("users.id, users.nickname, COALESCE(users.avatar_emoji, '') AS avatar_emoji, COALESCE(users.avatar_color, '') AS avatar_color").
		Joins("JOIN users ON users.id = bills.created_by").
		Where("bills.deleted_at IS NULL").
		Group("users.id, users.nickname, users.avatar_emoji, users.avatar_color").
		Order("MIN(bills.created_at) ASC").Scan(&rows).Error
	return rows, err
}

func (d *BillDao) Aggregate(ctx context.Context, db *gorm.DB, createdBy uint64, from, to time.Time) ([]BillCategoryAggregate, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	query := db.WithContext(ctx).Model(&model.Bill{}).
		Select("type, category, CAST(SUM(amount) AS CHAR) AS amount, COUNT(*) AS count").
		Where("occurred_at >= ? AND occurred_at < ?", from, to)
	if createdBy != 0 {
		query = query.Where("created_by = ?", createdBy)
	}
	var rows []BillCategoryAggregate
	err := query.Group("type, category").Order("type ASC, SUM(amount) DESC").Scan(&rows).Error
	return rows, err
}

func (d *BillDao) AggregateMonthly(ctx context.Context, db *gorm.DB, createdBy uint64, from, to time.Time) ([]BillMonthlyAggregate, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	query := db.WithContext(ctx).Model(&model.Bill{}).
		Select("MONTH(occurred_at) AS month, type, CAST(SUM(amount) AS CHAR) AS amount, COUNT(*) AS count").
		Where("occurred_at >= ? AND occurred_at < ?", from, to)
	if createdBy != 0 {
		query = query.Where("created_by = ?", createdBy)
	}
	var rows []BillMonthlyAggregate
	err := query.Group("MONTH(occurred_at), type").Order("month ASC, type ASC").Scan(&rows).Error
	return rows, err
}

func (d *BillDao) AggregateUsers(ctx context.Context, db *gorm.DB, from, to time.Time) ([]BillUserAggregate, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	var rows []BillUserAggregate
	err := db.WithContext(ctx).Table("bills").
		Select("bills.created_by, COALESCE(users.nickname, '') AS nickname, COALESCE(users.avatar_emoji, '') AS avatar_emoji, COALESCE(users.avatar_color, '') AS avatar_color, bills.type, CAST(SUM(bills.amount) AS CHAR) AS amount, COUNT(*) AS count").
		Joins("LEFT JOIN users ON users.id = bills.created_by").
		Where("bills.deleted_at IS NULL AND bills.occurred_at >= ? AND bills.occurred_at < ?", from, to).
		Group("bills.created_by, users.nickname, users.avatar_emoji, users.avatar_color, bills.type").
		Order("SUM(bills.amount) DESC").Scan(&rows).Error
	return rows, err
}

func billQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&model.Bill{}).
		Select("bills.*, COALESCE(users.nickname, '') AS creator_nickname, COALESCE(users.avatar_emoji, '') AS creator_emoji, COALESCE(users.avatar_color, '') AS creator_color").
		Joins("LEFT JOIN users ON users.id = bills.created_by")
}

func applyBillFilters(query *gorm.DB, in BillListQuery) *gorm.DB {
	if in.Type != "" {
		query = query.Where("bills.type = ?", in.Type)
	}
	if in.Category != "" {
		query = query.Where("bills.category = ?", in.Category)
	}
	if in.Keyword != "" {
		keyword := "%" + in.Keyword + "%"
		query = query.Where("(bills.title LIKE ? OR bills.note LIKE ?)", keyword, keyword)
	}
	if in.CreatedBy != 0 {
		query = query.Where("bills.created_by = ?", in.CreatedBy)
	}
	if in.From != nil {
		query = query.Where("bills.occurred_at >= ?", *in.From)
	}
	if in.To != nil {
		query = query.Where("bills.occurred_at < ?", *in.To)
	}
	return query
}
