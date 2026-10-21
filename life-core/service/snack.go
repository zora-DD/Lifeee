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

type SnackService struct {
	db   *gorm.DB
	repo *dao.SnackDao
}

func NewSnackService(db *gorm.DB, repo *dao.SnackDao) *SnackService {
	return &SnackService{db: db, repo: repo}
}

func (s *SnackService) List(ctx context.Context, req *proto.ListSnacksRequest) (*proto.ListSnacksResponse, error) {
	page, pageSize := int(req.GetPage()), int(req.GetPageSize())
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	status := strings.TrimSpace(req.GetStatus())
	if pageSize > 100 || !validSnackStatus(status, true) {
		return nil, hecode.ErrInvalidParam
	}
	rows, total, err := s.repo.List(ctx, s.db, dao.SnackQuery{
		Category: strings.TrimSpace(req.GetCategory()),
		Status:   status,
		Keyword:  strings.TrimSpace(req.GetKeyword()),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: list snacks: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.SnackItem, 0, len(rows))
	for i := range rows {
		items = append(items, snackToProto(&rows[i], nil))
	}
	return &proto.ListSnacksResponse{Items: items, Page: int32(page), PageSize: int32(pageSize), Total: total, HasMore: int64(page*pageSize) < total}, nil
}

func (s *SnackService) Get(ctx context.Context, req *proto.GetSnackRequest) (*proto.SnackItem, error) {
	id, err := parseSnackID(req.GetId())
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
	return snackToProto(row, records), nil
}

func (s *SnackService) Create(ctx context.Context, req *proto.CreateSnackRequest) (*proto.SnackItem, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	row, err := snackFromCreate(req, user.ID)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.ExistsByName(ctx, s.db, row.Name, row.Brand, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: duplicate snack", hecode.ErrInvalidParam)
	}
	if err = s.repo.Create(ctx, s.db, row); err != nil {
		return nil, err
	}
	return snackToProto(row, nil), nil
}

func (s *SnackService) Update(ctx context.Context, req *proto.UpdateSnackRequest) (*proto.SnackItem, error) {
	id, err := parseSnackID(req.GetId())
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
	row, err := snackFromUpdate(req, current)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.ExistsByName(ctx, s.db, row.Name, row.Brand, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: duplicate snack", hecode.ErrInvalidParam)
	}
	if err = s.repo.Update(ctx, s.db, row); err != nil {
		return nil, err
	}
	return s.Get(ctx, &proto.GetSnackRequest{Id: req.GetId()})
}

func (s *SnackService) Delete(ctx context.Context, req *proto.DeleteSnackRequest) (*proto.DeleteSnackResponse, error) {
	id, err := parseSnackID(req.GetId())
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
	return &proto.DeleteSnackResponse{Deleted: true}, nil
}

func (s *SnackService) Taste(ctx context.Context, req *proto.TasteSnackRequest) (*proto.SnackItem, error) {
	snackID, err := parseSnackID(req.GetSnackId())
	if err != nil {
		return nil, err
	}
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	date := time.Now()
	if strings.TrimSpace(req.GetTastedAt()) != "" {
		date, err = time.ParseInLocation("2006-01-02", req.GetTastedAt(), time.Local)
		if err != nil {
			return nil, hecode.ErrInvalidParam
		}
	}
	record := &model.SnackTastingRecord{SnackID: snackID, TastedBy: userID, TastedAt: date, Note: optionalString(strings.TrimSpace(req.GetNote()))}
	if err = s.repo.RecordTaste(ctx, s.db, snackID, record); err != nil {
		return nil, err
	}
	return s.Get(ctx, &proto.GetSnackRequest{Id: req.GetSnackId()})
}

func (s *SnackService) Recommend(ctx context.Context, req *proto.RecommendSnackRequest) (*proto.SnackItem, error) {
	status := strings.TrimSpace(req.GetStatus())
	if !validSnackStatus(status, true) {
		return nil, hecode.ErrInvalidParam
	}
	rows, err := s.repo.ListCandidates(ctx, s.db, dao.SnackQuery{Category: strings.TrimSpace(req.GetCategory()), Status: status})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, hecode.ErrNotFound
	}
	now := time.Now()
	best, bestScore := -1, -1
	for i := range rows {
		if rows[i].Status == "avoid" {
			continue
		}
		score := minInt(snackDaysSince(rows[i].LastTastedAt, now), 60)*2 + rand.IntN(12)
		switch rows[i].Status {
		case "wishlist":
			score += 28
		case "frequent":
			score += 22
		case "eaten":
			score += 14
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		return nil, hecode.ErrNotFound
	}
	return snackToProto(&rows[best], nil), nil
}

func snackFromCreate(req *proto.CreateSnackRequest, userID uint64) (*model.Snack, error) {
	return buildSnack(req.GetName(), req.GetBrand(), req.GetPurchasePlace(), req.GetCategory(), req.GetStatus(), req.GetCoverUrl(), req.GetCoverObjectKey(), req.GetNote(), req.GetMemoryTag(), userID)
}

func snackFromUpdate(req *proto.UpdateSnackRequest, current *model.Snack) (*model.Snack, error) {
	row, err := buildSnack(req.GetName(), req.GetBrand(), req.GetPurchasePlace(), req.GetCategory(), req.GetStatus(), req.GetCoverUrl(), req.GetCoverObjectKey(), req.GetNote(), req.GetMemoryTag(), current.CreatedBy)
	if row != nil {
		row.ID = current.ID
		row.TastingCount = current.TastingCount
		row.LastTastedAt = current.LastTastedAt
	}
	return row, err
}

func buildSnack(name, brand, purchasePlace, category, status, coverURL, coverKey, note, memoryTag string, userID uint64) (*model.Snack, error) {
	name = strings.TrimSpace(name)
	brand = strings.TrimSpace(brand)
	purchasePlace = strings.TrimSpace(purchasePlace)
	category = strings.TrimSpace(category)
	status = strings.TrimSpace(status)
	note = strings.TrimSpace(note)
	memoryTag = strings.TrimSpace(memoryTag)
	if status == "" {
		status = "wishlist"
	}
	if name == "" || category == "" || userID == 0 || !validSnackStatus(status, false) || utf8.RuneCountInString(name) > 120 || utf8.RuneCountInString(brand) > 120 || utf8.RuneCountInString(purchasePlace) > 255 || utf8.RuneCountInString(note) > 2000 || utf8.RuneCountInString(memoryTag) > 64 {
		return nil, hecode.ErrInvalidParam
	}
	coverURL, coverKey = strings.TrimSpace(coverURL), strings.TrimSpace(coverKey)
	if coverKey != "" {
		coverURL = ""
	}
	return &model.Snack{Name: name, Brand: brand, PurchasePlace: purchasePlace, Category: category, Status: status, CoverURL: optionalString(coverURL), CoverObjectKey: optionalString(coverKey), Note: optionalString(note), MemoryTag: optionalString(memoryTag), CreatedBy: userID}, nil
}

func snackToProto(row *model.Snack, records []model.SnackTastingRecord) *proto.SnackItem {
	item := &proto.SnackItem{Id: strconv.FormatUint(row.ID, 10), Name: row.Name, Brand: row.Brand, PurchasePlace: row.PurchasePlace, Category: row.Category, Status: row.Status, TastingCount: row.TastingCount, CreatedBy: strconv.FormatUint(row.CreatedBy, 10), CreatedAt: row.CreatedAt.Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Format(time.RFC3339)}
	legacyCoverURL := ""
	if row.CoverURL != nil {
		legacyCoverURL = *row.CoverURL
	}
	if row.CoverObjectKey != nil {
		item.CoverObjectKey = *row.CoverObjectKey
	}
	item.CoverUrl = configuredMediaURL(item.CoverObjectKey, legacyCoverURL)
	if row.Note != nil {
		item.Note = *row.Note
	}
	if row.MemoryTag != nil {
		item.MemoryTag = *row.MemoryTag
	}
	if row.LastTastedAt != nil {
		item.LastTastedAt = row.LastTastedAt.Format("2006-01-02")
	}
	for i := range records {
		item.Records = append(item.Records, snackRecordToProto(&records[i]))
	}
	return item
}

func snackRecordToProto(row *model.SnackTastingRecord) *proto.SnackTastingRecordItem {
	item := &proto.SnackTastingRecordItem{Id: strconv.FormatUint(row.ID, 10), SnackId: strconv.FormatUint(row.SnackID, 10), TastedBy: strconv.FormatUint(row.TastedBy, 10), TastedAt: row.TastedAt.Format("2006-01-02"), CreatedAt: row.CreatedAt.Format(time.RFC3339)}
	if row.Note != nil {
		item.Note = *row.Note
	}
	return item
}

func parseSnackID(value string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, hecode.ErrInvalidParam
	}
	return id, nil
}

func validSnackStatus(status string, allowEmpty bool) bool {
	if status == "" {
		return allowEmpty
	}
	return status == "wishlist" || status == "eaten" || status == "frequent" || status == "avoid"
}

func snackDaysSince(last *time.Time, now time.Time) int {
	if last == nil {
		return 365
	}
	days := int(now.Sub(*last).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
