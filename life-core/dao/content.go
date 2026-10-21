package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"life-core/model"
)

// ContentSummary 是任务、吃喝和地点在个人内容列表中的统一摘要。
type ContentSummary struct {
	TargetID       uint64
	TargetType     string
	Title          string
	Category       string
	Status         string
	Priority       string
	DueAt          *time.Time
	Icon           string
	Visited        bool
	VisitCount     uint32
	TotalSpent     string
	CoverURL       string
	CoverObjectKey string
	CreatedAt      time.Time
}

// FavoriteContent 在统一内容摘要上补充收藏记录信息。
type FavoriteContent struct {
	ContentSummary
	FavoriteID        uint64
	FavoriteCreatedAt time.Time
}

// ContentDao 负责跨内容类型的统一查询。
type ContentDao struct{}

func NewContentDao() *ContentDao { return &ContentDao{} }

// ListContributions 按创建时间统一分页查询当前用户创建的内容。
func (d *ContentDao) ListContributions(
	ctx context.Context,
	db *gorm.DB,
	userID uint64,
	targetType string,
	page int,
	pageSize int,
) ([]ContentSummary, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	branches, args := contributionBranches(userID, targetType)
	unionSQL := strings.Join(branches, " UNION ALL ")
	var total int64
	if err := db.WithContext(ctx).Raw("SELECT COUNT(1) FROM ("+unionSQL+") AS contents", args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(page, pageSize)
	rows := make([]ContentSummary, 0)
	query := "SELECT * FROM (" + unionSQL + ") AS contents ORDER BY created_at DESC, target_id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), limit, offset)
	if err := db.WithContext(ctx).Raw(query, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetByTarget 返回一条未删除内容的统一摘要。
func (d *ContentDao) GetByTarget(
	ctx context.Context,
	db *gorm.DB,
	targetType string,
	targetID uint64,
) (*ContentSummary, error) {
	if err := checkDB(db); err != nil {
		return nil, err
	}
	switch targetType {
	case model.FavoriteTargetTask:
		var row model.Task
		if err := db.WithContext(ctx).First(&row, "id = ?", targetID).Error; err != nil {
			return nil, err
		}
		return contentFromTask(&row), nil
	case model.FavoriteTargetRestaurant:
		var row model.Restaurant
		if err := db.WithContext(ctx).First(&row, "id = ?", targetID).Error; err != nil {
			return nil, err
		}
		return contentFromRestaurant(&row), nil
	case model.FavoriteTargetPlace:
		var row model.Place
		if err := db.WithContext(ctx).First(&row, "id = ?", targetID).Error; err != nil {
			return nil, err
		}
		return contentFromPlace(&row), nil
	default:
		return nil, fmt.Errorf("dao.ContentDao.GetByTarget: unsupported target type %q", targetType)
	}
}

// ListFavorites 只返回目标内容仍然存在的收藏，并按收藏时间倒序分页。
func (d *ContentDao) ListFavorites(
	ctx context.Context,
	db *gorm.DB,
	userID uint64,
	targetType string,
	page int,
	pageSize int,
) ([]FavoriteContent, int64, error) {
	if err := checkDB(db); err != nil {
		return nil, 0, err
	}
	contentSQL := `SELECT id AS target_id, 'task' AS target_type, title, category,
		status, priority, due_at, '' AS icon, 0 AS visited, 0 AS visit_count,
		'0.00' AS total_spent, '' AS cover_url, '' AS cover_object_key, created_at
		FROM tasks WHERE deleted_at IS NULL
		UNION ALL
		SELECT id AS target_id, 'restaurant' AS target_type, name AS title, category,
		preference AS status, '' AS priority, NULL AS due_at, '' AS icon, 0 AS visited, visit_count,
		CAST(total_spent AS CHAR) AS total_spent, COALESCE(cover_url, '') AS cover_url,
		COALESCE(cover_object_key, '') AS cover_object_key, created_at
		FROM restaurants WHERE deleted_at IS NULL
		UNION ALL
		SELECT id AS target_id, 'place' AS target_type, name AS title, category,
		'' AS status, '' AS priority, NULL AS due_at, COALESCE(icon, '') AS icon, visited, visit_count,
		'0.00' AS total_spent, COALESCE(cover_url, '') AS cover_url,
		COALESCE(cover_object_key, '') AS cover_object_key, created_at
		FROM places WHERE deleted_at IS NULL`
	whereSQL := " WHERE f.user_id = ?"
	args := []any{userID}
	if targetType != "" {
		whereSQL += " AND f.target_type = ?"
		args = append(args, targetType)
	}
	fromSQL := " FROM favorites f JOIN (" + contentSQL + ") AS c ON c.target_type = f.target_type AND c.target_id = f.target_id" + whereSQL
	var total int64
	if err := db.WithContext(ctx).Raw("SELECT COUNT(1)"+fromSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageLimit(page, pageSize)
	rows := make([]FavoriteContent, 0)
	query := `SELECT c.*, f.id AS favorite_id, f.created_at AS favorite_created_at` + fromSQL +
		" ORDER BY f.created_at DESC, f.id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), limit, offset)
	if err := db.WithContext(ctx).Raw(query, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func contributionBranches(userID uint64, targetType string) ([]string, []any) {
	queries := map[string]string{
		model.FavoriteTargetTask: `SELECT id AS target_id, 'task' AS target_type, title, category,
			status, priority, due_at, '' AS icon, 0 AS visited, 0 AS visit_count,
			'0.00' AS total_spent, '' AS cover_url, '' AS cover_object_key, created_at
			FROM tasks WHERE created_by = ? AND deleted_at IS NULL`,
		model.FavoriteTargetRestaurant: `SELECT id AS target_id, 'restaurant' AS target_type, name AS title, category,
			preference AS status, '' AS priority, NULL AS due_at, '' AS icon, 0 AS visited, visit_count,
			CAST(total_spent AS CHAR) AS total_spent, COALESCE(cover_url, '') AS cover_url,
			COALESCE(cover_object_key, '') AS cover_object_key, created_at
			FROM restaurants WHERE created_by = ? AND deleted_at IS NULL`,
		model.FavoriteTargetPlace: `SELECT id AS target_id, 'place' AS target_type, name AS title, category,
			'' AS status, '' AS priority, NULL AS due_at, COALESCE(icon, '') AS icon, visited, visit_count,
			'0.00' AS total_spent, COALESCE(cover_url, '') AS cover_url,
			COALESCE(cover_object_key, '') AS cover_object_key, created_at
			FROM places WHERE created_by = ? AND deleted_at IS NULL`,
	}
	types := []string{model.FavoriteTargetTask, model.FavoriteTargetRestaurant, model.FavoriteTargetPlace}
	if targetType != "" {
		types = []string{targetType}
	}
	branches := make([]string, 0, len(types))
	args := make([]any, 0, len(types))
	for _, itemType := range types {
		branches = append(branches, queries[itemType])
		args = append(args, userID)
	}
	return branches, args
}

func contentFromTask(row *model.Task) *ContentSummary {
	return &ContentSummary{
		TargetID: row.ID, TargetType: model.FavoriteTargetTask, Title: row.Title,
		Category: row.Category, Status: row.Status, Priority: row.Priority,
		DueAt: row.DueAt, CreatedAt: row.CreatedAt,
	}
}

func contentFromRestaurant(row *model.Restaurant) *ContentSummary {
	item := &ContentSummary{
		TargetID: row.ID, TargetType: model.FavoriteTargetRestaurant, Title: row.Name,
		Category: row.Category, Status: row.Preference, VisitCount: row.VisitCount,
		TotalSpent: row.TotalSpent, CreatedAt: row.CreatedAt,
	}
	if row.CoverURL != nil {
		item.CoverURL = *row.CoverURL
	}
	if row.CoverObjectKey != nil {
		item.CoverObjectKey = *row.CoverObjectKey
	}
	return item
}

func contentFromPlace(row *model.Place) *ContentSummary {
	item := &ContentSummary{
		TargetID: row.ID, TargetType: model.FavoriteTargetPlace, Title: row.Name,
		Category: row.Category, Visited: row.Visited, VisitCount: row.VisitCount,
		CreatedAt: row.CreatedAt,
	}
	if row.Icon != nil {
		item.Icon = *row.Icon
	}
	if row.CoverURL != nil {
		item.CoverURL = *row.CoverURL
	}
	if row.CoverObjectKey != nil {
		item.CoverObjectKey = *row.CoverObjectKey
	}
	return item
}
