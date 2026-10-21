package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vaynedu/hollow/pkg/hecode"

	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

const (
	defaultContentPageSize = 20
	maxContentPageSize     = 100
)

func normalizeContentListParam(targetType string, page, pageSize int32) (string, int, int, error) {
	targetType = strings.TrimSpace(targetType)
	if targetType != "" && !isFavoriteTargetType(targetType) {
		return "", 0, 0, fmt.Errorf("%w: unsupported targetType", hecode.ErrInvalidParam)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = defaultContentPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxContentPageSize {
		return "", 0, 0, fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	return targetType, int(page), int(pageSize), nil
}

func isFavoriteTargetType(targetType string) bool {
	return targetType == model.FavoriteTargetTask ||
		targetType == model.FavoriteTargetRestaurant ||
		targetType == model.FavoriteTargetPlace
}

func contentToProto(row *dao.ContentSummary) *proto.ContentSummary {
	if row == nil {
		return nil
	}
	item := &proto.ContentSummary{
		TargetId: strconv.FormatUint(row.TargetID, 10), TargetType: row.TargetType,
		Title: row.Title, Category: row.Category, Status: row.Status, Priority: row.Priority,
		Icon: row.Icon, Visited: row.Visited, VisitCount: row.VisitCount,
		TotalSpent: row.TotalSpent,
		CoverUrl:   configuredMediaURL(row.CoverObjectKey, row.CoverURL), CreatedAt: formatTime(row.CreatedAt),
	}
	if row.DueAt != nil {
		item.DueAt = formatTime(*row.DueAt)
	}
	return item
}
