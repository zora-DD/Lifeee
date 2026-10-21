package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vaynedu/hollow/pkg/hecode"
	"github.com/vaynedu/hollow/pkg/hlog"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

const (
	defaultPlacePageSize = 20
	maxPlacePageSize     = 100
	defaultPlaceCreator  = uint64(1)
	maxPlaceNameLen      = 120
	maxPlaceCategoryLen  = 32
	maxPlaceIconLen      = 16
	maxPlaceFeatureLen   = 1000
	maxPlaceNoteLen      = 2000
	maxPlaceAddressLen   = 255
	maxPlaceBestTimeLen  = 120
	maxPlaceDistanceKM   = 100000
	maxPlaceDriveMinutes = 525600
)

// PlaceRepository 定义地点服务依赖的数据访问能力。
type PlaceRepository interface {
	Create(context.Context, *gorm.DB, *model.Place) error
	GetByID(context.Context, *gorm.DB, uint64) (*model.Place, error)
	Update(context.Context, *gorm.DB, *model.Place) error
	SoftDelete(context.Context, *gorm.DB, uint64) error
	List(context.Context, *gorm.DB, dao.PlaceListQuery) ([]model.Place, int64, error)
	Stats(context.Context, *gorm.DB) (dao.PlaceStats, error)
}

// PlaceService 负责地点记录的参数校验和业务编排。
type PlaceService struct {
	db         *gorm.DB
	repository PlaceRepository
	filter     dao.PlaceListQuery
	ready      bool
}

func NewPlaceService(db *gorm.DB, repository PlaceRepository) *PlaceService {
	return &PlaceService{db: db, repository: repository}
}

// InitAndCheckListParam 清洗并校验地点列表参数。
func (s *PlaceService) InitAndCheckListParam(_ context.Context, req *proto.ListPlacesRequest) error {
	if req == nil {
		return fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	status := strings.TrimSpace(req.GetStatus())
	var visited *bool
	switch status {
	case "":
	case "visited":
		value := true
		visited = &value
	case "wishlist":
		value := false
		visited = &value
	default:
		return fmt.Errorf("%w: unsupported status", hecode.ErrInvalidParam)
	}
	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultPlacePageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxPlacePageSize {
		return fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	s.filter = dao.PlaceListQuery{
		Category:  strings.TrimSpace(req.GetCategory()),
		Visited:   visited,
		Keyword:   strings.TrimSpace(req.GetKeyword()),
		CreatedBy: req.GetCreatedBy(),
		Page:      page,
		PageSize:  pageSize,
	}
	s.ready = true
	return nil
}

func (s *PlaceService) List(ctx context.Context, req *proto.ListPlacesRequest) (*proto.ListPlacesResponse, error) {
	logger := hlog.FromContext(ctx).Named("PlaceService.List").
		With(hlog.String("category", req.GetCategory())).
		With(hlog.String("status", req.GetStatus())).
		With(hlog.Int32("page", req.GetPage())).
		With(hlog.Int32("page_size", req.GetPageSize()))
	if s == nil || !s.ready {
		return nil, fmt.Errorf("%w: InitAndCheckListParam must be called first", hecode.ErrService)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	rows, total, err := s.repository.List(ctx, s.db, s.filter)
	if err != nil {
		logger.Error("repository List failed", hlog.Err(err))
		return nil, fmt.Errorf("%w: list places: %v", hecode.ErrDBQuery, err)
	}
	stats, err := s.repository.Stats(ctx, s.db)
	if err != nil {
		logger.Error("repository Stats failed", hlog.Err(err))
		return nil, fmt.Errorf("%w: summarize places: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.PlaceItem, 0, len(rows))
	for i := range rows {
		items = append(items, placeToProto(&rows[i]))
	}
	return &proto.ListPlacesResponse{
		Items:         items,
		Page:          int32(s.filter.Page),
		PageSize:      int32(s.filter.PageSize),
		Total:         total,
		HasMore:       int64(s.filter.Page)*int64(s.filter.PageSize) < total,
		OverallTotal:  stats.OverallTotal,
		VisitedCount:  stats.VisitedCount,
		WishlistCount: stats.WishlistCount,
	}, nil
}

func (s *PlaceService) Get(ctx context.Context, req *proto.GetPlaceRequest) (*proto.PlaceItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parsePlaceID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return placeToProto(row), nil
}

func (s *PlaceService) Create(ctx context.Context, req *proto.CreatePlaceRequest) (*proto.PlaceItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	row, err := buildPlace(
		req.GetName(), req.GetCategory(), req.GetIcon(), req.GetVisited(), req.GetVisitCount(),
		req.GetLastVisitedAt(), req.GetFeature(), req.GetNote(), req.GetAddress(), req.GetBestTime(),
		req.GetCoverUrl(), req.GetCoverObjectKey(), req.GetLatitude(), req.GetLongitude(), req.GetGuideUrl(),
		req.GetDistanceKm(), req.GetDriveDurationMinutes(),
	)
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row.CreatedBy = authctx.UserIDOrDefault(ctx, defaultPlaceCreator)
	if err := s.repository.Create(ctx, s.db, row); err != nil {
		return nil, fmt.Errorf("%w: create place: %v", hecode.ErrDBInsert, err)
	}
	return placeToProto(row), nil
}

func (s *PlaceService) Update(ctx context.Context, req *proto.UpdatePlaceRequest) (*proto.PlaceItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parsePlaceID(req.GetId())
	if err != nil {
		return nil, err
	}
	input, err := buildPlace(
		req.GetName(), req.GetCategory(), req.GetIcon(), req.GetVisited(), req.GetVisitCount(),
		req.GetLastVisitedAt(), req.GetFeature(), req.GetNote(), req.GetAddress(), req.GetBestTime(),
		req.GetCoverUrl(), req.GetCoverObjectKey(), req.GetLatitude(), req.GetLongitude(), req.GetGuideUrl(),
		req.GetDistanceKm(), req.GetDriveDurationMinutes(),
	)
	if err != nil {
		return nil, err
	}
	current, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireOwnerOrAdmin(ctx, current.CreatedBy); err != nil {
		return nil, err
	}
	current.Name = input.Name
	current.Category = input.Category
	current.Icon = input.Icon
	current.Visited = input.Visited
	current.VisitCount = input.VisitCount
	current.LastVisitedAt = input.LastVisitedAt
	current.Feature = input.Feature
	current.Note = input.Note
	current.Address = input.Address
	current.BestTime = input.BestTime
	current.DistanceKM = input.DistanceKM
	current.DriveMinutes = input.DriveMinutes
	current.GuideURL = input.GuideURL
	current.CoverURL = input.CoverURL
	current.CoverObjectKey = input.CoverObjectKey
	current.Latitude = input.Latitude
	current.Longitude = input.Longitude
	if err := s.repository.Update(ctx, s.db, current); err != nil {
		return nil, fmt.Errorf("%w: update place: %v", hecode.ErrDBUpdate, err)
	}
	updated, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return placeToProto(updated), nil
}

func (s *PlaceService) Delete(ctx context.Context, req *proto.DeletePlaceRequest) (*proto.DeletePlaceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parsePlaceID(req.GetId())
	if err != nil {
		return nil, err
	}
	current, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireOwnerOrAdmin(ctx, current.CreatedBy); err != nil {
		return nil, err
	}
	if err := s.repository.SoftDelete(ctx, s.db, id); err != nil {
		return nil, fmt.Errorf("%w: delete place: %v", hecode.ErrDBDelete, err)
	}
	return &proto.DeletePlaceResponse{Deleted: true}, nil
}

func (s *PlaceService) checkDependencies() error {
	if s == nil || s.db == nil || s.repository == nil {
		return fmt.Errorf("%w: place service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func (s *PlaceService) getByID(ctx context.Context, id uint64) (*model.Place, error) {
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row, err := s.repository.GetByID(ctx, s.db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: place %d", hecode.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get place: %v", hecode.ErrDBQuery, err)
	}
	return row, nil
}

func parsePlaceID(value string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid place id", hecode.ErrInvalidParam)
	}
	return id, nil
}

func buildPlace(
	name, category, icon string,
	visited bool,
	visitCount uint32,
	lastVisitedAt, feature, note, address, bestTime, coverURL, coverObjectKey, latitude, longitude, guideURL string,
	distanceKM, driveMinutes uint32,
) (*model.Place, error) {
	name = strings.TrimSpace(name)
	category = strings.TrimSpace(category)
	icon = strings.TrimSpace(icon)
	feature = strings.TrimSpace(feature)
	note = strings.TrimSpace(note)
	address = strings.TrimSpace(address)
	bestTime = strings.TrimSpace(bestTime)
	coverURL = strings.TrimSpace(coverURL)
	coverObjectKey = strings.TrimSpace(coverObjectKey)
	guideURL = strings.TrimSpace(guideURL)
	if coverObjectKey != "" {
		if err := validateObjectKey(coverObjectKey); err != nil || !strings.HasPrefix(coverObjectKey, model.MediaBizTypePlace+"/") {
			return nil, fmt.Errorf("%w: invalid place cover", hecode.ErrInvalidParam)
		}
		// 自建媒体只持久化对象键，签名访问地址在响应阶段动态生成。
		coverURL = ""
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", hecode.ErrInvalidParam)
	}
	if feature == "" {
		return nil, fmt.Errorf("%w: feature is required", hecode.ErrInvalidParam)
	}
	if category == "" {
		category = "其他"
	}
	if utf8.RuneCountInString(name) > maxPlaceNameLen ||
		utf8.RuneCountInString(category) > maxPlaceCategoryLen ||
		utf8.RuneCountInString(icon) > maxPlaceIconLen ||
		utf8.RuneCountInString(feature) > maxPlaceFeatureLen ||
		utf8.RuneCountInString(note) > maxPlaceNoteLen ||
		utf8.RuneCountInString(address) > maxPlaceAddressLen ||
		utf8.RuneCountInString(bestTime) > maxPlaceBestTimeLen ||
		len(guideURL) > 512 || len(coverURL) > 512 || len(coverObjectKey) > 255 {
		return nil, fmt.Errorf("%w: place field is too long", hecode.ErrInvalidParam)
	}
	if guideURL != "" {
		parsedGuideURL, err := url.ParseRequestURI(guideURL)
		if err != nil || parsedGuideURL.Scheme != "https" || parsedGuideURL.Host != "mp.weixin.qq.com" ||
			(parsedGuideURL.Path != "/s" && !strings.HasPrefix(parsedGuideURL.Path, "/s/")) {
			return nil, fmt.Errorf("%w: guideUrl must be a WeChat article URL", hecode.ErrInvalidParam)
		}
	}
	if distanceKM > maxPlaceDistanceKM || driveMinutes > maxPlaceDriveMinutes {
		return nil, fmt.Errorf("%w: invalid travel reference", hecode.ErrInvalidParam)
	}
	latitudeValue, err := normalizeCoordinate(latitude, -90, 90, "latitude")
	if err != nil {
		return nil, err
	}
	longitudeValue, err := normalizeCoordinate(longitude, -180, 180, "longitude")
	if err != nil {
		return nil, err
	}
	row := &model.Place{
		Name:           name,
		Category:       category,
		Icon:           optionalString(icon),
		Visited:        visited,
		Feature:        optionalString(feature),
		Note:           optionalString(note),
		Address:        optionalString(address),
		BestTime:       optionalString(bestTime),
		DistanceKM:     distanceKM,
		DriveMinutes:   driveMinutes,
		GuideURL:       optionalString(guideURL),
		CoverURL:       optionalString(coverURL),
		CoverObjectKey: optionalString(coverObjectKey),
		Latitude:       latitudeValue,
		Longitude:      longitudeValue,
	}
	if !visited {
		return row, nil
	}
	if visitCount == 0 {
		visitCount = 1
	}
	visitedAt, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(lastVisitedAt), time.Local)
	if err != nil {
		return nil, fmt.Errorf("%w: lastVisitedAt must be YYYY-MM-DD when visited", hecode.ErrInvalidParam)
	}
	row.VisitCount = visitCount
	row.LastVisitedAt = &visitedAt
	return row, nil
}

func normalizeCoordinate(value string, minValue, maxValue float64, field string) (*string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < minValue || parsed > maxValue {
		return nil, fmt.Errorf("%w: invalid %s", hecode.ErrInvalidParam, field)
	}
	normalized := strconv.FormatFloat(parsed, 'f', 7, 64)
	return &normalized, nil
}

func placeToProto(row *model.Place) *proto.PlaceItem {
	if row == nil {
		return nil
	}
	item := &proto.PlaceItem{
		Id:                   strconv.FormatUint(row.ID, 10),
		Name:                 row.Name,
		Category:             row.Category,
		Visited:              row.Visited,
		VisitCount:           row.VisitCount,
		CreatedBy:            strconv.FormatUint(row.CreatedBy, 10),
		CreatedAt:            formatTime(row.CreatedAt),
		UpdatedAt:            formatTime(row.UpdatedAt),
		DistanceKm:           row.DistanceKM,
		DriveDurationMinutes: row.DriveMinutes,
	}
	if row.Icon != nil {
		item.Icon = *row.Icon
	}
	if row.LastVisitedAt != nil {
		item.LastVisitedAt = row.LastVisitedAt.Format("2006-01-02")
	}
	if row.Feature != nil {
		item.Feature = *row.Feature
	}
	if row.Note != nil {
		item.Note = *row.Note
	}
	if row.Address != nil {
		item.Address = *row.Address
	}
	if row.BestTime != nil {
		item.BestTime = *row.BestTime
	}
	if row.GuideURL != nil {
		item.GuideUrl = *row.GuideURL
	}
	legacyCoverURL := ""
	if row.CoverURL != nil {
		legacyCoverURL = *row.CoverURL
	}
	if row.CoverObjectKey != nil {
		item.CoverObjectKey = *row.CoverObjectKey
	}
	item.CoverUrl = configuredMediaURL(item.CoverObjectKey, legacyCoverURL)
	if row.Latitude != nil {
		item.Latitude = *row.Latitude
	}
	if row.Longitude != nil {
		item.Longitude = *row.Longitude
	}
	return item
}
