package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

const (
	defaultMemoryPageSize = 20
	maxMemoryPageSize     = 100
	maxMemoryImages       = 9
)

type MemoryRepository interface {
	Create(context.Context, *gorm.DB, *model.Memory) error
	List(context.Context, *gorm.DB, dao.MemoryListQuery) ([]model.Memory, int64, error)
	ListYears(context.Context, *gorm.DB) ([]int, error)
}

type MemoryPhotoRepository interface {
	CreateBatch(context.Context, *gorm.DB, []model.MemoryPhoto) error
	ListByMemoryIDs(context.Context, *gorm.DB, []uint64) ([]model.MemoryPhoto, error)
}

// MemoryService 负责管理员时光相册的查询和新增。
type MemoryService struct {
	db       *gorm.DB
	memories MemoryRepository
	photos   MemoryPhotoRepository
}

func NewMemoryService(db *gorm.DB, memories MemoryRepository, photos MemoryPhotoRepository) *MemoryService {
	return &MemoryService{db: db, memories: memories, photos: photos}
}

func (s *MemoryService) List(ctx context.Context, req *proto.ListMemoriesRequest) (*proto.ListMemoriesResponse, error) {
	if err := requireApprovedAdmin(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	year := int(req.GetYear())
	if year != 0 && (year < 1900 || year > 2100) {
		return nil, fmt.Errorf("%w: invalid memory year", hecode.ErrInvalidParam)
	}
	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultMemoryPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxMemoryPageSize {
		return nil, fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	rows, total, err := s.memories.List(ctx, s.db, dao.MemoryListQuery{Year: year, Page: page, PageSize: pageSize})
	if err != nil {
		return nil, fmt.Errorf("%w: list memories: %v", hecode.ErrDBQuery, err)
	}
	ids := make([]uint64, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	photoRows, err := s.photos.ListByMemoryIDs(ctx, s.db, ids)
	if err != nil {
		return nil, fmt.Errorf("%w: list memory photos: %v", hecode.ErrDBQuery, err)
	}
	photoMap := make(map[uint64][]model.MemoryPhoto, len(rows))
	for i := range photoRows {
		photoMap[photoRows[i].MemoryID] = append(photoMap[photoRows[i].MemoryID], photoRows[i])
	}
	items := make([]*proto.MemoryItem, 0, len(rows))
	for i := range rows {
		items = append(items, memoryToProto(&rows[i], photoMap[rows[i].ID]))
	}
	years, err := s.memories.ListYears(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("%w: list memory years: %v", hecode.ErrDBQuery, err)
	}
	protoYears := make([]int32, 0, len(years))
	for _, item := range years {
		protoYears = append(protoYears, int32(item))
	}
	return &proto.ListMemoriesResponse{
		Items: items, Page: int32(page), PageSize: int32(pageSize), Total: total,
		HasMore: int64(page)*int64(pageSize) < total, Years: protoYears,
	}, nil
}

func (s *MemoryService) Create(ctx context.Context, req *proto.CreateMemoryRequest) (*proto.MemoryItem, error) {
	user, err := approvedAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	date, precision, err := parseMemoryDate(req.GetDate())
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.GetTitle())
	place := strings.TrimSpace(req.GetPlace())
	description := strings.TrimSpace(req.GetDescription())
	sourceKey := strings.TrimSpace(req.GetSourceKey())
	if title == "" || utf8.RuneCountInString(title) > 160 || utf8.RuneCountInString(place) > 255 ||
		utf8.RuneCountInString(description) > 1000 || len(sourceKey) > 96 {
		return nil, fmt.Errorf("%w: invalid memory content", hecode.ErrInvalidParam)
	}
	if len(req.GetImages()) == 0 || len(req.GetImages()) > maxMemoryImages {
		return nil, fmt.Errorf("%w: memory requires 1 to %d images", hecode.ErrInvalidParam, maxMemoryImages)
	}
	row := &model.Memory{
		MemoryDate: date, DatePrecision: precision, Place: optionalString(place), Title: title,
		Description: optionalString(description), SourceKey: optionalString(sourceKey), CreatedBy: user.ID,
	}
	photos := make([]model.MemoryPhoto, 0, len(req.GetImages()))
	for index, image := range req.GetImages() {
		mediaID, parseErr := strconv.ParseUint(strings.TrimSpace(image.GetMediaId()), 10, 64)
		objectKey := strings.TrimSpace(image.GetObjectKey())
		if parseErr != nil || mediaID == 0 || validateObjectKey(objectKey) != nil || !strings.HasPrefix(objectKey, model.MediaBizTypeMemory+"/") {
			return nil, fmt.Errorf("%w: invalid memory image", hecode.ErrInvalidParam)
		}
		photos = append(photos, model.MemoryPhoto{
			MediaAssetID: mediaID, ObjectKey: objectKey, AltText: optionalString(strings.TrimSpace(image.GetAltText())),
			Width: image.GetWidth(), Height: image.GetHeight(), SortOrder: uint32(index),
		})
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.memories.Create(ctx, tx, row); err != nil {
			return err
		}
		for i := range photos {
			photos[i].MemoryID = row.ID
		}
		return s.photos.CreateBatch(ctx, tx, photos)
	}); err != nil {
		return nil, fmt.Errorf("%w: create memory: %v", hecode.ErrDBInsert, err)
	}
	return memoryToProto(row, photos), nil
}

func (s *MemoryService) checkDependencies() error {
	if s == nil || s.db == nil || s.memories == nil || s.photos == nil {
		return fmt.Errorf("%w: memory service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func approvedAdmin(ctx context.Context) (*model.User, error) {
	user, ok := authctx.UserFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	if user.Status != model.UserStatusApproved || user.Role != model.UserRoleAdmin {
		return nil, hecode.ErrPermissionDenied
	}
	return user, nil
}

func requireApprovedAdmin(ctx context.Context) error {
	_, err := approvedAdmin(ctx)
	return err
}

func parseMemoryDate(value string) (time.Time, string, error) {
	value = strings.TrimSpace(value)
	formats := []struct {
		layout    string
		precision string
	}{
		{layout: "2006-01-02", precision: model.MemoryDatePrecisionDay},
		{layout: "2006-01", precision: model.MemoryDatePrecisionMonth},
		{layout: "2006", precision: model.MemoryDatePrecisionYear},
	}
	for _, format := range formats {
		if parsed, err := time.ParseInLocation(format.layout, value, time.Local); err == nil {
			return parsed, format.precision, nil
		}
	}
	return time.Time{}, "", fmt.Errorf("%w: date must be YYYY-MM-DD, YYYY-MM or YYYY", hecode.ErrInvalidParam)
}

func memoryToProto(row *model.Memory, photos []model.MemoryPhoto) *proto.MemoryItem {
	if row == nil {
		return nil
	}
	item := &proto.MemoryItem{
		Id: strconv.FormatUint(row.ID, 10), Date: formatMemoryDate(row.MemoryDate, row.DatePrecision),
		DatePrecision: row.DatePrecision, Title: row.Title, CreatedBy: strconv.FormatUint(row.CreatedBy, 10),
		CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt), Images: make([]*proto.MemoryPhotoItem, 0, len(photos)),
	}
	if row.Place != nil {
		item.Place = *row.Place
	}
	if row.Description != nil {
		item.Description = *row.Description
	}
	for i := range photos {
		photo := photos[i]
		photoItem := &proto.MemoryPhotoItem{
			Id: strconv.FormatUint(photo.ID, 10), MediaId: strconv.FormatUint(photo.MediaAssetID, 10),
			ObjectKey: photo.ObjectKey, Url: configuredMediaURL(photo.ObjectKey, ""), Width: photo.Width,
			Height: photo.Height, SortOrder: photo.SortOrder,
		}
		if photo.AltText != nil {
			photoItem.AltText = *photo.AltText
		}
		item.Images = append(item.Images, photoItem)
	}
	return item
}

func formatMemoryDate(value time.Time, precision string) string {
	switch precision {
	case model.MemoryDatePrecisionYear:
		return value.Format("2006")
	case model.MemoryDatePrecisionMonth:
		return value.Format("2006-01")
	default:
		return value.Format("2006-01-02")
	}
}
