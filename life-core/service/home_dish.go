package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	"life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type HomeDishService struct {
	db   *gorm.DB
	repo *dao.HomeDishDao
}

func NewHomeDishService(db *gorm.DB, repo *dao.HomeDishDao) *HomeDishService {
	return &HomeDishService{db: db, repo: repo}
}

func (s *HomeDishService) List(ctx context.Context, req *proto.ListHomeDishesRequest) (*proto.ListHomeDishesResponse, error) {
	page, size := int(req.GetPage()), int(req.GetPageSize())
	status := strings.TrimSpace(req.GetStatus())
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 || req.GetMaxMinutes() < 0 || !validHomeDishStatus(status) {
		return nil, hecode.ErrInvalidParam
	}
	rows, total, err := s.repo.List(ctx, s.db, dao.HomeDishQuery{Category: strings.TrimSpace(req.GetCategory()), Difficulty: strings.TrimSpace(req.GetDifficulty()), Keyword: strings.TrimSpace(req.GetKeyword()), Status: status, MaxMinutes: int(req.GetMaxMinutes()), Page: page, PageSize: size})
	if err != nil {
		return nil, fmt.Errorf("%w: list home dishes: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.HomeDishItem, 0, len(rows))
	for i := range rows {
		items = append(items, homeDishToProto(&rows[i], nil))
	}
	return &proto.ListHomeDishesResponse{Items: items, Page: int32(page), PageSize: int32(size), Total: total, HasMore: int64(page*size) < total}, nil
}
func (s *HomeDishService) Get(ctx context.Context, req *proto.GetHomeDishRequest) (*proto.HomeDishItem, error) {
	id, err := parseHomeDishID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.repo.Get(ctx, s.db, id)
	if err == gorm.ErrRecordNotFound {
		return nil, hecode.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	records, err := s.repo.ListRecords(ctx, s.db, id, 20)
	if err != nil {
		return nil, err
	}
	return homeDishToProto(row, records), nil
}
func (s *HomeDishService) Create(ctx context.Context, req *proto.CreateHomeDishRequest) (*proto.HomeDishItem, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	row, err := homeDishFromCreate(req, user.ID)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.ExistsByName(ctx, s.db, row.Name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: duplicate dish name", hecode.ErrInvalidParam)
	}
	if err = s.repo.Create(ctx, s.db, row); err != nil {
		return nil, err
	}
	return homeDishToProto(row, nil), nil
}
func (s *HomeDishService) Update(ctx context.Context, req *proto.UpdateHomeDishRequest) (*proto.HomeDishItem, error) {
	id, err := parseHomeDishID(req.GetId())
	if err != nil {
		return nil, err
	}
	current, err := s.repo.Get(ctx, s.db, id)
	if err != nil {
		return nil, hecode.ErrNotFound
	}
	user, _ := auth.UserFromContext(ctx)
	if user == nil || (user.Role != model.UserRoleAdmin && user.ID != current.CreatedBy) {
		return nil, hecode.ErrPermissionDenied
	}
	row, err := homeDishFromUpdate(req, current)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.ExistsByName(ctx, s.db, row.Name, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: duplicate dish name", hecode.ErrInvalidParam)
	}
	if err = s.repo.Update(ctx, s.db, row); err != nil {
		return nil, err
	}
	return s.Get(ctx, &proto.GetHomeDishRequest{Id: req.GetId()})
}
func (s *HomeDishService) Delete(ctx context.Context, req *proto.DeleteHomeDishRequest) (*proto.DeleteHomeDishResponse, error) {
	id, err := parseHomeDishID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.repo.Get(ctx, s.db, id)
	if err != nil {
		return nil, hecode.ErrNotFound
	}
	user, _ := auth.UserFromContext(ctx)
	if user == nil || (user.Role != model.UserRoleAdmin && user.ID != row.CreatedBy) {
		return nil, hecode.ErrPermissionDenied
	}
	if err = s.repo.Delete(ctx, s.db, id); err != nil {
		return nil, err
	}
	return &proto.DeleteHomeDishResponse{Deleted: true}, nil
}
func (s *HomeDishService) Cook(ctx context.Context, req *proto.CookHomeDishRequest) (*proto.HomeDishItem, error) {
	dishID, err := parseHomeDishID(req.GetDishId())
	if err != nil {
		return nil, err
	}
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	date := time.Now()
	if strings.TrimSpace(req.GetCookedAt()) != "" {
		date, err = time.ParseInLocation("2006-01-02", req.GetCookedAt(), time.Local)
		if err != nil {
			return nil, hecode.ErrInvalidParam
		}
	}
	if req.GetDurationMinutes() > 1440 || req.GetRating() > 5 {
		return nil, hecode.ErrInvalidParam
	}
	record := &model.HomeCookingRecord{DishID: dishID, CookedBy: userID, CookedAt: date, DurationMinutes: req.GetDurationMinutes(), Rating: req.GetRating(), Note: optionalString(strings.TrimSpace(req.GetNote())), PhotoURL: optionalString(strings.TrimSpace(req.GetPhotoUrl())), PhotoObjectKey: optionalString(strings.TrimSpace(req.GetPhotoObjectKey()))}
	if err = s.repo.RecordCook(ctx, s.db, dishID, userID, record); err != nil {
		return nil, err
	}
	return s.Get(ctx, &proto.GetHomeDishRequest{Id: req.GetDishId()})
}
func (s *HomeDishService) Recommend(ctx context.Context, req *proto.RecommendHomeDishRequest) (*proto.HomeDishItem, error) {
	status := strings.TrimSpace(req.GetStatus())
	if !validHomeDishStatus(status) {
		return nil, hecode.ErrInvalidParam
	}
	rows, err := s.repo.ListCandidates(ctx, s.db, dao.HomeDishQuery{Category: strings.TrimSpace(req.GetCategory()), Difficulty: strings.TrimSpace(req.GetDifficulty()), Status: status, MaxMinutes: int(req.GetMaxMinutes())})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, hecode.ErrNotFound
	}
	wanted := splitIngredients(req.GetIngredients())
	now := time.Now()
	best := -1
	bestScore := -1
	for i := range rows {
		score := int(rows[i].Rating)*20 + minInt(homeDishDaysSince(rows[i].LastCookedAt, now), 60)*2 + minInt(int(rows[i].CookCount)*3, 15) + rand.IntN(10)
		if len(wanted) > 0 {
			dishText := strings.ToLower(rows[i].Ingredients)
			for _, v := range wanted {
				if strings.Contains(dishText, strings.ToLower(v)) {
					score += 12
				}
			}
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	return homeDishToProto(&rows[best], nil), nil
}

func homeDishFromCreate(req *proto.CreateHomeDishRequest, userID uint64) (*model.HomeDish, error) {
	return buildHomeDish(req.GetName(), req.GetCategory(), req.GetIngredients(), req.GetSteps(), req.GetDifficulty(), req.GetEstimatedMinutes(), req.GetCoverUrl(), req.GetCoverObjectKey(), req.GetExperience(), req.GetRating(), userID)
}
func homeDishFromUpdate(req *proto.UpdateHomeDishRequest, current *model.HomeDish) (*model.HomeDish, error) {
	row, err := buildHomeDish(req.GetName(), req.GetCategory(), req.GetIngredients(), req.GetSteps(), req.GetDifficulty(), req.GetEstimatedMinutes(), req.GetCoverUrl(), req.GetCoverObjectKey(), req.GetExperience(), req.GetRating(), current.CreatedBy)
	if row != nil {
		row.ID = current.ID
		row.CookCount = current.CookCount
		row.LastCookedAt = current.LastCookedAt
	}
	return row, err
}
func buildHomeDish(name, category, ingredients, steps, difficulty string, minutes uint32, coverURL, coverKey, experience string, rating uint32, userID uint64) (*model.HomeDish, error) {
	name, category, ingredients, steps, difficulty, experience = strings.TrimSpace(name), strings.TrimSpace(category), strings.TrimSpace(ingredients), strings.TrimSpace(steps), strings.TrimSpace(difficulty), strings.TrimSpace(experience)
	if name == "" || category == "" || userID == 0 || utf8.RuneCountInString(name) > 120 || utf8.RuneCountInString(ingredients) > 1000 || utf8.RuneCountInString(steps) > 10000 || utf8.RuneCountInString(experience) > 2000 || minutes > 1440 || rating > 5 {
		return nil, hecode.ErrInvalidParam
	}
	if difficulty == "" {
		difficulty = "normal"
	}
	if difficulty != "easy" && difficulty != "normal" && difficulty != "hard" {
		return nil, hecode.ErrInvalidParam
	}
	coverURL, coverKey = strings.TrimSpace(coverURL), strings.TrimSpace(coverKey)
	if coverKey != "" {
		coverURL = ""
	}
	return &model.HomeDish{Name: name, Category: category, Ingredients: ingredients, Steps: steps, Difficulty: difficulty, EstimatedMinutes: minutes, CoverURL: optionalString(strings.TrimSpace(coverURL)), CoverObjectKey: optionalString(strings.TrimSpace(coverKey)), Experience: optionalString(experience), Rating: rating, CreatedBy: userID}, nil
}
func homeDishToProto(row *model.HomeDish, records []model.HomeCookingRecord) *proto.HomeDishItem {
	item := &proto.HomeDishItem{Id: strconv.FormatUint(row.ID, 10), Name: row.Name, Category: row.Category, Ingredients: row.Ingredients, Steps: row.Steps, Difficulty: row.Difficulty, EstimatedMinutes: row.EstimatedMinutes, Rating: row.Rating, CookCount: row.CookCount, Status: homeDishStatus(row.CookCount), CreatedBy: strconv.FormatUint(row.CreatedBy, 10), CreatedAt: row.CreatedAt.Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Format(time.RFC3339)}
	legacyCoverURL := ""
	if row.CoverURL != nil {
		legacyCoverURL = *row.CoverURL
	}
	if row.CoverObjectKey != nil {
		item.CoverObjectKey = *row.CoverObjectKey
	}
	item.CoverUrl = configuredMediaURL(item.CoverObjectKey, legacyCoverURL)
	if row.Experience != nil {
		item.Experience = *row.Experience
	}
	if row.LastCookedAt != nil {
		item.LastCookedAt = row.LastCookedAt.Format("2006-01-02")
	}
	for i := range records {
		item.Records = append(item.Records, homeRecordToProto(&records[i]))
	}
	return item
}
func homeRecordToProto(row *model.HomeCookingRecord) *proto.HomeCookingRecordItem {
	item := &proto.HomeCookingRecordItem{Id: strconv.FormatUint(row.ID, 10), DishId: strconv.FormatUint(row.DishID, 10), CookedBy: strconv.FormatUint(row.CookedBy, 10), CookedAt: row.CookedAt.Format("2006-01-02"), DurationMinutes: row.DurationMinutes, Rating: row.Rating, CreatedAt: row.CreatedAt.Format(time.RFC3339)}
	if row.Note != nil {
		item.Note = *row.Note
	}
	legacyPhotoURL := ""
	if row.PhotoURL != nil {
		legacyPhotoURL = *row.PhotoURL
	}
	if row.PhotoObjectKey != nil {
		item.PhotoObjectKey = *row.PhotoObjectKey
	}
	item.PhotoUrl = configuredMediaURL(item.PhotoObjectKey, legacyPhotoURL)
	return item
}
func parseHomeDishID(text string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
	if err != nil || id == 0 {
		return 0, hecode.ErrInvalidParam
	}
	return id, nil
}
func splitIngredients(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '，' || r == '、' || r == ' ' })
}

func validHomeDishStatus(status string) bool {
	return status == "" || status == "wishlist" || status == "eaten" || status == "frequent"
}

func homeDishStatus(cookCount uint32) string {
	if cookCount == 0 {
		return "wishlist"
	}
	if cookCount >= 3 {
		return "frequent"
	}
	return "eaten"
}
func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func homeDishDaysSince(last *time.Time, now time.Time) int {
	if last == nil {
		return 365
	}
	days := int(now.Sub(*last).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
