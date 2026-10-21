package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type FavoriteRepository interface {
	Create(context.Context, *gorm.DB, *model.Favorite) error
	GetByUserTarget(context.Context, *gorm.DB, uint64, string, uint64) (*model.Favorite, error)
	DeleteByUserTarget(context.Context, *gorm.DB, uint64, string, uint64) error
}

type FavoriteContentRepository interface {
	GetByTarget(context.Context, *gorm.DB, string, uint64) (*dao.ContentSummary, error)
	ListFavorites(context.Context, *gorm.DB, uint64, string, int, int) ([]dao.FavoriteContent, int64, error)
}

// FavoriteService 负责收藏的幂等增删、状态和列表查询。
type FavoriteService struct {
	db        *gorm.DB
	favorites FavoriteRepository
	contents  FavoriteContentRepository
}

func NewFavoriteService(db *gorm.DB, favorites FavoriteRepository, contents FavoriteContentRepository) *FavoriteService {
	return &FavoriteService{db: db, favorites: favorites, contents: contents}
}

func (s *FavoriteService) List(ctx context.Context, req *proto.ListFavoritesRequest) (*proto.ListFavoritesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	targetType, page, pageSize, err := normalizeContentListParam(req.GetTargetType(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	rows, total, err := s.contents.ListFavorites(ctx, s.db, userID, targetType, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("%w: list favorites: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.FavoriteItem, 0, len(rows))
	for i := range rows {
		items = append(items, &proto.FavoriteItem{
			Id:        strconv.FormatUint(rows[i].FavoriteID, 10),
			Target:    contentToProto(&rows[i].ContentSummary),
			CreatedAt: formatTime(rows[i].FavoriteCreatedAt),
		})
	}
	return &proto.ListFavoritesResponse{
		Items: items, Page: int32(page), PageSize: int32(pageSize), Total: total,
		HasMore: int64(page)*int64(pageSize) < total,
	}, nil
}

func (s *FavoriteService) Status(ctx context.Context, req *proto.GetFavoriteStatusRequest) (*proto.FavoriteStatusResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	targetType, targetID, err := parseFavoriteTarget(req.GetTargetType(), req.GetTargetId())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	_, err = s.favorites.GetByUserTarget(ctx, s.db, userID, targetType, targetID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return favoriteStatus(targetType, targetID, false), nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get favorite status: %v", hecode.ErrDBQuery, err)
	}
	return favoriteStatus(targetType, targetID, true), nil
}

func (s *FavoriteService) Modify(ctx context.Context, req *proto.ModifyFavoriteRequest) (*proto.FavoriteStatusResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	targetType, targetID, err := parseFavoriteTarget(req.GetTargetType(), req.GetTargetId())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	if !req.GetFavorited() {
		if err := s.favorites.DeleteByUserTarget(ctx, s.db, userID, targetType, targetID); err != nil {
			return nil, fmt.Errorf("%w: delete favorite: %v", hecode.ErrDBDelete, err)
		}
		return favoriteStatus(targetType, targetID, false), nil
	}
	if _, err := s.contents.GetByTarget(ctx, s.db, targetType, targetID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s %d", hecode.ErrNotFound, targetType, targetID)
	} else if err != nil {
		return nil, fmt.Errorf("%w: get favorite target: %v", hecode.ErrDBQuery, err)
	}
	if _, err := s.favorites.GetByUserTarget(ctx, s.db, userID, targetType, targetID); err == nil {
		return favoriteStatus(targetType, targetID, true), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: get favorite status: %v", hecode.ErrDBQuery, err)
	}
	row := &model.Favorite{UserID: userID, TargetType: targetType, TargetID: targetID}
	if err := s.favorites.Create(ctx, s.db, row); err != nil {
		// 并发收藏撞唯一键时再次查询，已存在即视为成功。
		if _, getErr := s.favorites.GetByUserTarget(ctx, s.db, userID, targetType, targetID); getErr == nil {
			return favoriteStatus(targetType, targetID, true), nil
		}
		return nil, fmt.Errorf("%w: create favorite: %v", hecode.ErrDBInsert, err)
	}
	return favoriteStatus(targetType, targetID, true), nil
}

func (s *FavoriteService) checkDependencies() error {
	if s == nil || s.db == nil || s.favorites == nil || s.contents == nil {
		return fmt.Errorf("%w: favorite service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func currentUserID(ctx context.Context) (uint64, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return 0, hecode.ErrUnauthorized
	}
	return userID, nil
}

func parseFavoriteTarget(targetType, targetIDText string) (string, uint64, error) {
	targetType = strings.TrimSpace(targetType)
	if !isFavoriteTargetType(targetType) {
		return "", 0, fmt.Errorf("%w: unsupported targetType", hecode.ErrInvalidParam)
	}
	targetID, err := strconv.ParseUint(strings.TrimSpace(targetIDText), 10, 64)
	if err != nil || targetID == 0 {
		return "", 0, fmt.Errorf("%w: invalid targetId", hecode.ErrInvalidParam)
	}
	return targetType, targetID, nil
}

func favoriteStatus(targetType string, targetID uint64, favorited bool) *proto.FavoriteStatusResponse {
	return &proto.FavoriteStatusResponse{
		TargetType: targetType, TargetId: strconv.FormatUint(targetID, 10), Favorited: favorited,
	}
}
