package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
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
	defaultRestaurantPageSize = 20
	maxRestaurantPageSize     = 100
	defaultRestaurantCreator  = uint64(1)
	maxRestaurantNameLength   = 120
	maxRestaurantCategoryLen  = 32
	maxRestaurantCuisineLen   = 64
	maxRestaurantAddressLen   = 255
	maxRestaurantHoursLen     = 120
	maxRestaurantFeaturesLen  = 500
	maxRestaurantMealLen      = 500
	maxRestaurantNoteLen      = 1000
)

// RestaurantRepository 定义吃喝地点服务依赖的数据访问能力。
type RestaurantRepository interface {
	Create(context.Context, *gorm.DB, *model.Restaurant) error
	GetByID(context.Context, *gorm.DB, uint64) (*model.Restaurant, error)
	ExistsByName(context.Context, *gorm.DB, string, uint64) (bool, error)
	GetByIDForUpdate(context.Context, *gorm.DB, uint64) (*model.Restaurant, error)
	UpdateProfile(context.Context, *gorm.DB, *model.Restaurant) error
	UpdateSummary(context.Context, *gorm.DB, uint64, dao.RestaurantSummary) error
	SoftDelete(context.Context, *gorm.DB, uint64) error
	List(context.Context, *gorm.DB, dao.RestaurantListQuery) ([]model.Restaurant, int64, error)
	Stats(context.Context, *gorm.DB) (dao.RestaurantStats, error)
}

// RestaurantConsumptionRepository 定义消费记录服务依赖的数据访问能力。
type RestaurantConsumptionRepository interface {
	Create(context.Context, *gorm.DB, *model.RestaurantConsumption) error
	GetByID(context.Context, *gorm.DB, uint64) (*model.RestaurantConsumption, error)
	Update(context.Context, *gorm.DB, *model.RestaurantConsumption) error
	SoftDelete(context.Context, *gorm.DB, uint64) error
	SoftDeleteByRestaurant(context.Context, *gorm.DB, uint64) error
	ListByRestaurant(context.Context, *gorm.DB, uint64, int, int) ([]model.RestaurantConsumption, int64, error)
	ListForSummary(context.Context, *gorm.DB, uint64) ([]model.RestaurantConsumption, error)
}

// RestaurantService 负责吃喝地点、消费记录与汇总数据的业务编排。
type RestaurantService struct {
	db                    *gorm.DB
	repository            RestaurantRepository
	consumptionRepository RestaurantConsumptionRepository
	filter                dao.RestaurantListQuery
	ready                 bool
}

func NewRestaurantService(
	db *gorm.DB,
	repository RestaurantRepository,
	consumptionRepository RestaurantConsumptionRepository,
) *RestaurantService {
	return &RestaurantService{
		db:                    db,
		repository:            repository,
		consumptionRepository: consumptionRepository,
	}
}

// InitAndCheckListParam 清洗并校验吃喝地点列表参数。
func (s *RestaurantService) InitAndCheckListParam(_ context.Context, req *proto.ListRestaurantsRequest) error {
	if req == nil {
		return fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	preference := strings.TrimSpace(req.GetPreference())
	if preference != "" && !isRestaurantPreference(preference) {
		return fmt.Errorf("%w: unsupported preference", hecode.ErrInvalidParam)
	}
	status := strings.TrimSpace(req.GetStatus())
	if status != "" && !isRestaurantStatus(status) {
		return fmt.Errorf("%w: unsupported restaurant status", hecode.ErrInvalidParam)
	}
	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultRestaurantPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxRestaurantPageSize {
		return fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	s.filter = dao.RestaurantListQuery{
		Category:   strings.TrimSpace(req.GetCategory()),
		Preference: preference,
		Keyword:    strings.TrimSpace(req.GetKeyword()),
		CreatedBy:  req.GetCreatedBy(),
		Status:     status,
		Page:       page,
		PageSize:   pageSize,
	}
	s.ready = true
	return nil
}

func (s *RestaurantService) List(ctx context.Context, req *proto.ListRestaurantsRequest) (*proto.ListRestaurantsResponse, error) {
	logger := hlog.FromContext(ctx).Named("RestaurantService.List").
		With(hlog.String("category", req.GetCategory())).
		With(hlog.String("preference", req.GetPreference())).
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
		return nil, fmt.Errorf("%w: list restaurants: %v", hecode.ErrDBQuery, err)
	}
	stats, err := s.repository.Stats(ctx, s.db)
	if err != nil {
		logger.Error("repository Stats failed", hlog.Err(err))
		return nil, fmt.Errorf("%w: summarize restaurants: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.RestaurantItem, 0, len(rows))
	for i := range rows {
		items = append(items, restaurantToProto(&rows[i], nil))
	}
	return &proto.ListRestaurantsResponse{
		Items:           items,
		Page:            int32(s.filter.Page),
		PageSize:        int32(s.filter.PageSize),
		Total:           total,
		HasMore:         int64(s.filter.Page)*int64(s.filter.PageSize) < total,
		OverallTotal:    stats.OverallTotal,
		TotalVisitCount: stats.TotalVisitCount,
		TotalSpent:      stats.TotalSpent,
	}, nil
}

func (s *RestaurantService) Get(ctx context.Context, req *proto.GetRestaurantRequest) (*proto.RestaurantItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseRestaurantID(req.GetId(), "restaurant")
	if err != nil {
		return nil, err
	}
	row, err := s.getRestaurantByID(ctx, id)
	if err != nil {
		return nil, err
	}
	consumptions, _, err := s.consumptionRepository.ListByRestaurant(ctx, s.db, id, 1, maxRestaurantPageSize)
	if err != nil {
		return nil, fmt.Errorf("%w: list restaurant consumptions: %v", hecode.ErrDBQuery, err)
	}
	return restaurantToProto(row, consumptions), nil
}

func (s *RestaurantService) Create(ctx context.Context, req *proto.CreateRestaurantRequest) (*proto.RestaurantItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	row, err := buildRestaurant(
		req.GetName(), req.GetCategory(), req.GetCuisine(), req.GetPreference(),
		req.GetAddress(), req.GetBusinessHours(), req.GetFeatures(), req.GetCoverUrl(), req.GetCoverObjectKey(),
		req.GetLatitude(), req.GetLongitude(),
	)
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	if err := s.checkNameAvailable(ctx, row.Name, 0); err != nil {
		return nil, err
	}
	row.CreatedBy = authctx.UserIDOrDefault(ctx, defaultRestaurantCreator)
	row.TotalSpent = "0.00"
	if err := s.repository.Create(ctx, s.db, row); err != nil {
		return nil, fmt.Errorf("%w: create restaurant: %v", hecode.ErrDBInsert, err)
	}
	return restaurantToProto(row, nil), nil
}

func (s *RestaurantService) Update(ctx context.Context, req *proto.UpdateRestaurantRequest) (*proto.RestaurantItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseRestaurantID(req.GetId(), "restaurant")
	if err != nil {
		return nil, err
	}
	input, err := buildRestaurant(
		req.GetName(), req.GetCategory(), req.GetCuisine(), req.GetPreference(),
		req.GetAddress(), req.GetBusinessHours(), req.GetFeatures(), req.GetCoverUrl(), req.GetCoverObjectKey(),
		req.GetLatitude(), req.GetLongitude(),
	)
	if err != nil {
		return nil, err
	}
	current, err := s.getRestaurantByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireOwnerOrAdmin(ctx, current.CreatedBy); err != nil {
		return nil, err
	}
	if err := s.checkNameAvailable(ctx, input.Name, id); err != nil {
		return nil, err
	}
	current.Name = input.Name
	current.Category = input.Category
	current.Cuisine = input.Cuisine
	current.Preference = input.Preference
	current.Address = input.Address
	current.Latitude = input.Latitude
	current.Longitude = input.Longitude
	current.BusinessHours = input.BusinessHours
	current.Features = input.Features
	current.CoverURL = input.CoverURL
	current.CoverObjectKey = input.CoverObjectKey
	if err := s.repository.UpdateProfile(ctx, s.db, current); err != nil {
		return nil, fmt.Errorf("%w: update restaurant: %v", hecode.ErrDBUpdate, err)
	}
	return s.Get(ctx, &proto.GetRestaurantRequest{Id: req.GetId()})
}

func (s *RestaurantService) Delete(ctx context.Context, req *proto.DeleteRestaurantRequest) (*proto.DeleteRestaurantResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseRestaurantID(req.GetId(), "restaurant")
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, getErr := s.repository.GetByIDForUpdate(ctx, tx, id)
		if getErr != nil {
			return getErr
		}
		if ownerErr := requireOwnerOrAdmin(ctx, current.CreatedBy); ownerErr != nil {
			return ownerErr
		}
		if deleteErr := s.consumptionRepository.SoftDeleteByRestaurant(ctx, tx, id); deleteErr != nil {
			return deleteErr
		}
		return s.repository.SoftDelete(ctx, tx, id)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: restaurant %d", hecode.ErrNotFound, id)
	}
	if errors.Is(err, hecode.ErrPermissionDenied) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("%w: delete restaurant: %v", hecode.ErrDBDelete, err)
	}
	return &proto.DeleteRestaurantResponse{Deleted: true}, nil
}

func (s *RestaurantService) Recommend(ctx context.Context, req *proto.RecommendRestaurantRequest) (*proto.RestaurantItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	preference := strings.TrimSpace(req.GetPreference())
	if preference != "" && !isRestaurantPreference(preference) {
		return nil, fmt.Errorf("%w: unsupported preference", hecode.ErrInvalidParam)
	}
	status := strings.TrimSpace(req.GetStatus())
	if status != "" && !isRestaurantStatus(status) {
		return nil, fmt.Errorf("%w: unsupported restaurant status", hecode.ErrInvalidParam)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	rows, _, err := s.repository.List(ctx, s.db, dao.RestaurantListQuery{
		Category:   strings.TrimSpace(req.GetCategory()),
		Preference: preference,
		Status:     status,
		Page:       1,
		PageSize:   maxRestaurantPageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: recommend restaurant: %v", hecode.ErrDBQuery, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: no restaurant candidate", hecode.ErrNotFound)
	}
	selected, err := chooseWeightedRestaurant(rows, time.Now(), secureRandomInt)
	if err != nil {
		return nil, fmt.Errorf("%w: choose restaurant: %v", hecode.ErrService, err)
	}
	return restaurantToProto(selected, nil), nil
}

// chooseWeightedRestaurant 按喜好、消费次数和距上次消费时间进行加权随机。
// 每家店始终保留正权重，避免固定推荐同一家。
func chooseWeightedRestaurant(
	rows []model.Restaurant,
	now time.Time,
	randomInt func(int64) (int64, error),
) (*model.Restaurant, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("no restaurant candidate")
	}
	weights := make([]int64, len(rows))
	var total int64
	for i := range rows {
		weights[i] = restaurantRecommendationWeight(&rows[i], now)
		total += weights[i]
	}
	ticket, err := randomInt(total)
	if err != nil {
		return nil, err
	}
	for i, weight := range weights {
		if ticket < weight {
			return &rows[i], nil
		}
		ticket -= weight
	}
	return &rows[len(rows)-1], nil
}

func secureRandomInt(max int64) (int64, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return value.Int64(), nil
}

func restaurantRecommendationWeight(row *model.Restaurant, now time.Time) int64 {
	preferenceWeight := map[string]int64{
		model.RestaurantPreferenceOkay: 8,
		model.RestaurantPreferenceGood: 12,
		model.RestaurantPreferenceLove: 16,
		model.RestaurantPreferenceMust: 20,
	}[row.Preference]
	if preferenceWeight == 0 {
		preferenceWeight = 12
	}

	visitCount := int64(row.VisitCount)
	if visitCount > 10 {
		visitCount = 10
	}
	frequencyWeight := int64(24) - visitCount*2

	recencyWeight := int64(30)
	if row.LastVisitedAt != nil {
		days := int64(now.Sub(*row.LastVisitedAt).Hours() / 24)
		if days < 0 {
			days = 0
		}
		recencyWeight = 5 + days/3
		if recencyWeight > 30 {
			recencyWeight = 30
		}
	}
	return preferenceWeight + frequencyWeight + recencyWeight
}

func (s *RestaurantService) ListConsumptions(
	ctx context.Context,
	req *proto.ListRestaurantConsumptionsRequest,
) (*proto.ListRestaurantConsumptionsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	restaurantID, err := parseRestaurantID(req.GetRestaurantId(), "restaurant")
	if err != nil {
		return nil, err
	}
	page, pageSize, err := normalizeRestaurantPagination(req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	if _, err := s.getRestaurantByID(ctx, restaurantID); err != nil {
		return nil, err
	}
	rows, total, err := s.consumptionRepository.ListByRestaurant(ctx, s.db, restaurantID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("%w: list restaurant consumptions: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.RestaurantConsumptionItem, 0, len(rows))
	for i := range rows {
		items = append(items, restaurantConsumptionToProto(&rows[i]))
	}
	return &proto.ListRestaurantConsumptionsResponse{
		Items:    items,
		Page:     int32(page),
		PageSize: int32(pageSize),
		Total:    total,
		HasMore:  int64(page)*int64(pageSize) < total,
	}, nil
}

func (s *RestaurantService) CreateConsumption(
	ctx context.Context,
	req *proto.CreateRestaurantConsumptionRequest,
) (*proto.RestaurantConsumptionItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	restaurantID, err := parseRestaurantID(req.GetRestaurantId(), "restaurant")
	if err != nil {
		return nil, err
	}
	row, err := buildRestaurantConsumption(req.GetConsumedAt(), req.GetAmount(), req.GetMeal(), req.GetNote())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row.RestaurantID = restaurantID
	row.ConsumedBy = authctx.UserIDOrDefault(ctx, defaultRestaurantCreator)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, getErr := s.repository.GetByIDForUpdate(ctx, tx, restaurantID); getErr != nil {
			return getErr
		}
		if createErr := s.consumptionRepository.Create(ctx, tx, row); createErr != nil {
			return createErr
		}
		return s.recalculateSummary(ctx, tx, restaurantID)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: restaurant %d", hecode.ErrNotFound, restaurantID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: create restaurant consumption: %v", hecode.ErrDBInsert, err)
	}
	return restaurantConsumptionToProto(row), nil
}

func (s *RestaurantService) UpdateConsumption(
	ctx context.Context,
	req *proto.UpdateRestaurantConsumptionRequest,
) (*proto.RestaurantConsumptionItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	restaurantID, err := parseRestaurantID(req.GetRestaurantId(), "restaurant")
	if err != nil {
		return nil, err
	}
	consumptionID, err := parseRestaurantID(req.GetId(), "consumption")
	if err != nil {
		return nil, err
	}
	input, err := buildRestaurantConsumption(req.GetConsumedAt(), req.GetAmount(), req.GetMeal(), req.GetNote())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, getErr := s.repository.GetByIDForUpdate(ctx, tx, restaurantID); getErr != nil {
			return getErr
		}
		current, getErr := s.consumptionRepository.GetByID(ctx, tx, consumptionID)
		if getErr != nil {
			return getErr
		}
		if current.RestaurantID != restaurantID {
			return gorm.ErrRecordNotFound
		}
		if ownerErr := requireOwnerOrAdmin(ctx, current.ConsumedBy); ownerErr != nil {
			return ownerErr
		}
		current.ConsumedAt = input.ConsumedAt
		current.Amount = input.Amount
		current.Meal = input.Meal
		current.Note = input.Note
		if updateErr := s.consumptionRepository.Update(ctx, tx, current); updateErr != nil {
			return updateErr
		}
		return s.recalculateSummary(ctx, tx, restaurantID)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: restaurant consumption %d", hecode.ErrNotFound, consumptionID)
	}
	if errors.Is(err, hecode.ErrPermissionDenied) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("%w: update restaurant consumption: %v", hecode.ErrDBUpdate, err)
	}
	updated, err := s.consumptionRepository.GetByID(ctx, s.db, consumptionID)
	if err != nil {
		return nil, fmt.Errorf("%w: get restaurant consumption: %v", hecode.ErrDBQuery, err)
	}
	return restaurantConsumptionToProto(updated), nil
}

func (s *RestaurantService) DeleteConsumption(
	ctx context.Context,
	req *proto.DeleteRestaurantConsumptionRequest,
) (*proto.DeleteRestaurantConsumptionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	restaurantID, err := parseRestaurantID(req.GetRestaurantId(), "restaurant")
	if err != nil {
		return nil, err
	}
	consumptionID, err := parseRestaurantID(req.GetId(), "consumption")
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, getErr := s.repository.GetByIDForUpdate(ctx, tx, restaurantID); getErr != nil {
			return getErr
		}
		current, getErr := s.consumptionRepository.GetByID(ctx, tx, consumptionID)
		if getErr != nil {
			return getErr
		}
		if current.RestaurantID != restaurantID {
			return gorm.ErrRecordNotFound
		}
		if ownerErr := requireOwnerOrAdmin(ctx, current.ConsumedBy); ownerErr != nil {
			return ownerErr
		}
		if deleteErr := s.consumptionRepository.SoftDelete(ctx, tx, consumptionID); deleteErr != nil {
			return deleteErr
		}
		return s.recalculateSummary(ctx, tx, restaurantID)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: restaurant consumption %d", hecode.ErrNotFound, consumptionID)
	}
	if errors.Is(err, hecode.ErrPermissionDenied) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("%w: delete restaurant consumption: %v", hecode.ErrDBDelete, err)
	}
	return &proto.DeleteRestaurantConsumptionResponse{Deleted: true}, nil
}

func (s *RestaurantService) recalculateSummary(ctx context.Context, tx *gorm.DB, restaurantID uint64) error {
	rows, err := s.consumptionRepository.ListForSummary(ctx, tx, restaurantID)
	if err != nil {
		return err
	}
	summary := dao.RestaurantSummary{VisitCount: uint32(len(rows)), TotalSpent: "0.00"}
	var totalCents uint64
	for i := range rows {
		if rows[i].Amount == nil {
			continue
		}
		cents, parseErr := moneyToCents(*rows[i].Amount, 10)
		if parseErr != nil {
			return parseErr
		}
		totalCents += cents
	}
	summary.TotalSpent = formatMoneyCents(totalCents)
	if len(rows) > 0 {
		latest := rows[0]
		summary.LastVisitedAt = &latest.ConsumedAt
		summary.LastAmount = latest.Amount
		summary.LastMeal = latest.Meal
		summary.LastNote = latest.Note
	}
	return s.repository.UpdateSummary(ctx, tx, restaurantID, summary)
}

func (s *RestaurantService) checkDependencies() error {
	if s == nil || s.db == nil || s.repository == nil || s.consumptionRepository == nil {
		return fmt.Errorf("%w: restaurant service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func (s *RestaurantService) checkNameAvailable(ctx context.Context, name string, excludeID uint64) error {
	exists, err := s.repository.ExistsByName(ctx, s.db, name, excludeID)
	if err != nil {
		return fmt.Errorf("%w: check restaurant name: %v", hecode.ErrDBQuery, err)
	}
	if exists {
		return hecode.WithMessage(hecode.ErrInvalidParam, "已存在同名吃喝记录")
	}
	return nil
}

func (s *RestaurantService) getRestaurantByID(ctx context.Context, id uint64) (*model.Restaurant, error) {
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row, err := s.repository.GetByID(ctx, s.db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: restaurant %d", hecode.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get restaurant: %v", hecode.ErrDBQuery, err)
	}
	return row, nil
}

func normalizeRestaurantPagination(pageValue, pageSizeValue int32) (int, int, error) {
	page := int(pageValue)
	if page == 0 {
		page = 1
	}
	pageSize := int(pageSizeValue)
	if pageSize == 0 {
		pageSize = defaultRestaurantPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxRestaurantPageSize {
		return 0, 0, fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	return page, pageSize, nil
}

func parseRestaurantID(value, resource string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid %s id", hecode.ErrInvalidParam, resource)
	}
	return id, nil
}

func buildRestaurant(name, category, cuisine, preference, address, hours, features, coverURL, coverObjectKey, latitude, longitude string) (*model.Restaurant, error) {
	name = strings.TrimSpace(name)
	category = strings.TrimSpace(category)
	cuisine = strings.TrimSpace(cuisine)
	preference = strings.TrimSpace(preference)
	address = strings.TrimSpace(address)
	hours = strings.TrimSpace(hours)
	features = strings.TrimSpace(features)
	coverURL = strings.TrimSpace(coverURL)
	coverObjectKey = strings.TrimSpace(coverObjectKey)
	if coverObjectKey != "" {
		if err := validateObjectKey(coverObjectKey); err != nil || !strings.HasPrefix(coverObjectKey, model.MediaBizTypeRestaurant+"/") {
			return nil, fmt.Errorf("%w: invalid restaurant cover", hecode.ErrInvalidParam)
		}
		// 自建媒体只持久化对象键，签名访问地址在响应阶段动态生成。
		coverURL = ""
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", hecode.ErrInvalidParam)
	}
	if utf8.RuneCountInString(name) > maxRestaurantNameLength {
		return nil, fmt.Errorf("%w: name is too long", hecode.ErrInvalidParam)
	}
	if category == "" {
		category = "其他"
	}
	if utf8.RuneCountInString(category) > maxRestaurantCategoryLen {
		return nil, fmt.Errorf("%w: category is too long", hecode.ErrInvalidParam)
	}
	if preference == "" {
		preference = model.RestaurantPreferenceGood
	}
	if !isRestaurantPreference(preference) {
		return nil, fmt.Errorf("%w: unsupported preference", hecode.ErrInvalidParam)
	}
	if utf8.RuneCountInString(cuisine) > maxRestaurantCuisineLen ||
		utf8.RuneCountInString(address) > maxRestaurantAddressLen ||
		utf8.RuneCountInString(hours) > maxRestaurantHoursLen ||
		utf8.RuneCountInString(features) > maxRestaurantFeaturesLen ||
		len(coverURL) > 512 || len(coverObjectKey) > 255 {
		return nil, fmt.Errorf("%w: restaurant field is too long", hecode.ErrInvalidParam)
	}
	latitudeValue, err := normalizeCoordinate(latitude, -90, 90, "latitude")
	if err != nil {
		return nil, err
	}
	longitudeValue, err := normalizeCoordinate(longitude, -180, 180, "longitude")
	if err != nil {
		return nil, err
	}
	return &model.Restaurant{
		Name:           name,
		Category:       category,
		Cuisine:        optionalString(cuisine),
		Preference:     preference,
		Address:        optionalString(address),
		Latitude:       latitudeValue,
		Longitude:      longitudeValue,
		BusinessHours:  optionalString(hours),
		Features:       optionalString(features),
		CoverURL:       optionalString(coverURL),
		CoverObjectKey: optionalString(coverObjectKey),
	}, nil
}

func buildRestaurantConsumption(consumedAt, amount, meal, note string) (*model.RestaurantConsumption, error) {
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(consumedAt), time.Local)
	if err != nil {
		return nil, fmt.Errorf("%w: consumedAt must be YYYY-MM-DD", hecode.ErrInvalidParam)
	}
	meal = strings.TrimSpace(meal)
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(meal) > maxRestaurantMealLen || utf8.RuneCountInString(note) > maxRestaurantNoteLen {
		return nil, fmt.Errorf("%w: consumption field is too long", hecode.ErrInvalidParam)
	}
	var normalizedAmount *string
	if strings.TrimSpace(amount) != "" {
		cents, parseErr := moneyToCents(amount, 8)
		if parseErr != nil {
			return nil, fmt.Errorf("%w: invalid amount", hecode.ErrInvalidParam)
		}
		value := formatMoneyCents(cents)
		normalizedAmount = &value
	}
	return &model.RestaurantConsumption{
		ConsumedAt: date,
		Amount:     normalizedAmount,
		Meal:       optionalString(meal),
		Note:       optionalString(note),
	}, nil
}

func isRestaurantPreference(value string) bool {
	return value == model.RestaurantPreferenceOkay ||
		value == model.RestaurantPreferenceGood ||
		value == model.RestaurantPreferenceLove ||
		value == model.RestaurantPreferenceMust
}

func isRestaurantStatus(value string) bool {
	return value == "wishlist" || value == "visited" || value == "frequent"
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func moneyToCents(value string, maxIntegerDigits int) (uint64, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts[0]) == 0 || len(parts[0]) > maxIntegerDigits {
		return 0, errors.New("invalid money")
	}
	for _, char := range parts[0] {
		if char < '0' || char > '9' {
			return 0, errors.New("invalid money")
		}
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 0 || len(fraction) > 2 {
			return 0, errors.New("invalid money")
		}
		for _, char := range fraction {
			if char < '0' || char > '9' {
				return 0, errors.New("invalid money")
			}
		}
	}
	if len(fraction) == 0 {
		fraction = "00"
	} else if len(fraction) == 1 {
		fraction += "0"
	}
	integer, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	decimal, err := strconv.ParseUint(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return integer*100 + decimal, nil
}

func formatMoneyCents(cents uint64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func restaurantToProto(row *model.Restaurant, consumptions []model.RestaurantConsumption) *proto.RestaurantItem {
	if row == nil {
		return nil
	}
	item := &proto.RestaurantItem{
		Id:           strconv.FormatUint(row.ID, 10),
		Name:         row.Name,
		Category:     row.Category,
		Preference:   row.Preference,
		VisitCount:   row.VisitCount,
		TotalSpent:   row.TotalSpent,
		CreatedBy:    strconv.FormatUint(row.CreatedBy, 10),
		CreatedAt:    formatTime(row.CreatedAt),
		UpdatedAt:    formatTime(row.UpdatedAt),
		Consumptions: make([]*proto.RestaurantConsumptionItem, 0, len(consumptions)),
	}
	if row.Cuisine != nil {
		item.Cuisine = *row.Cuisine
	}
	if row.Address != nil {
		item.Address = *row.Address
	}
	if row.Latitude != nil {
		item.Latitude = *row.Latitude
	}
	if row.Longitude != nil {
		item.Longitude = *row.Longitude
	}
	if row.BusinessHours != nil {
		item.BusinessHours = *row.BusinessHours
	}
	if row.Features != nil {
		item.Features = *row.Features
	}
	legacyCoverURL := ""
	if row.CoverURL != nil {
		legacyCoverURL = *row.CoverURL
	}
	if row.CoverObjectKey != nil {
		item.CoverObjectKey = *row.CoverObjectKey
	}
	item.CoverUrl = configuredMediaURL(item.CoverObjectKey, legacyCoverURL)
	if row.LastVisitedAt != nil {
		item.LastVisitedAt = row.LastVisitedAt.Format("2006-01-02")
	}
	if row.LastAmount != nil {
		item.LastAmount = *row.LastAmount
	}
	if row.LastMeal != nil {
		item.LastMeal = *row.LastMeal
	}
	if row.LastNote != nil {
		item.LastNote = *row.LastNote
	}
	for i := range consumptions {
		item.Consumptions = append(item.Consumptions, restaurantConsumptionToProto(&consumptions[i]))
	}
	return item
}

func restaurantConsumptionToProto(row *model.RestaurantConsumption) *proto.RestaurantConsumptionItem {
	if row == nil {
		return nil
	}
	item := &proto.RestaurantConsumptionItem{
		Id:           strconv.FormatUint(row.ID, 10),
		RestaurantId: strconv.FormatUint(row.RestaurantID, 10),
		ConsumedBy:   strconv.FormatUint(row.ConsumedBy, 10),
		ConsumedAt:   row.ConsumedAt.Format("2006-01-02"),
		CreatedAt:    formatTime(row.CreatedAt),
		UpdatedAt:    formatTime(row.UpdatedAt),
	}
	if row.Amount != nil {
		item.Amount = *row.Amount
	}
	if row.Meal != nil {
		item.Meal = *row.Meal
	}
	if row.Note != nil {
		item.Note = *row.Note
	}
	return item
}
