package service

import (
	"context"
	"fmt"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/proto"
)

// ContributionRepository 定义我的贡献所需的数据查询能力。
type ContributionRepository interface {
	ListContributions(context.Context, *gorm.DB, uint64, string, int, int) ([]dao.ContentSummary, int64, error)
}

// ContributionService 负责查询当前用户创建的公开内容。
type ContributionService struct {
	db         *gorm.DB
	repository ContributionRepository
}

func NewContributionService(db *gorm.DB, repository ContributionRepository) *ContributionService {
	return &ContributionService{db: db, repository: repository}
}

func (s *ContributionService) List(ctx context.Context, req *proto.ListContributionsRequest) (*proto.ListContributionsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	targetType, page, pageSize, err := normalizeContentListParam(req.GetTargetType(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	if s == nil || s.db == nil || s.repository == nil {
		return nil, fmt.Errorf("%w: contribution service dependencies are not initialized", hecode.ErrService)
	}
	rows, total, err := s.repository.ListContributions(ctx, s.db, userID, targetType, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("%w: list contributions: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.ContentSummary, 0, len(rows))
	for i := range rows {
		items = append(items, contentToProto(&rows[i]))
	}
	return &proto.ListContributionsResponse{
		Items: items, Page: int32(page), PageSize: int32(pageSize), Total: total,
		HasMore: int64(page)*int64(pageSize) < total,
	}, nil
}
