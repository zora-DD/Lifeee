package service

import (
	"context"
	"errors"
	"fmt"
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
	defaultActivityLogPageSize = 20
	maxActivityLogPageSize     = 100
	defaultActivityLogCreator  = uint64(1)
	maxActivityLogContentLen   = 2000
	maxActivityLogDuration     = 1440
)

// ActivityLogRepository 定义生活记录服务依赖的数据访问能力。
type ActivityLogRepository interface {
	Create(context.Context, *gorm.DB, *model.ActivityLog) error
	GetByID(context.Context, *gorm.DB, uint64) (*dao.ActivityLogWithCreator, error)
	Update(context.Context, *gorm.DB, *model.ActivityLog) error
	SoftDelete(context.Context, *gorm.DB, uint64) error
	List(context.Context, *gorm.DB, dao.ActivityLogListQuery) ([]dao.ActivityLogWithCreator, int64, error)
	ListForStats(context.Context, *gorm.DB, uint64, time.Time, time.Time) ([]model.ActivityLog, error)
}

// ActivityLogService 负责生活记录校验、读写和统计。
type ActivityLogService struct {
	db         *gorm.DB
	repository ActivityLogRepository
	filter     dao.ActivityLogListQuery
	ready      bool
}

func NewActivityLogService(db *gorm.DB, repository ActivityLogRepository) *ActivityLogService {
	return &ActivityLogService{db: db, repository: repository}
}

// InitAndCheckListParam 清洗并校验生活记录列表参数。
func (s *ActivityLogService) InitAndCheckListParam(_ context.Context, req *proto.ListActivityLogsRequest) error {
	if req == nil {
		return fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	inputType := strings.TrimSpace(req.GetInputType())
	if inputType != "" && inputType != model.ActivityInputTypeText && inputType != model.ActivityInputTypeVoice {
		return fmt.Errorf("%w: unsupported input type", hecode.ErrInvalidParam)
	}
	from, err := parseOptionalRFC3339(req.GetFrom(), "from")
	if err != nil {
		return err
	}
	to, err := parseOptionalRFC3339(req.GetTo(), "to")
	if err != nil {
		return err
	}
	if from != nil && to != nil && !from.Before(*to) {
		return fmt.Errorf("%w: from must be earlier than to", hecode.ErrInvalidParam)
	}
	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultActivityLogPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxActivityLogPageSize {
		return fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	s.filter = dao.ActivityLogListQuery{
		CreatedBy: req.GetCreatedBy(),
		From:      from,
		To:        to,
		InputType: inputType,
		Page:      page,
		PageSize:  pageSize,
	}
	s.ready = true
	return nil
}

func (s *ActivityLogService) List(ctx context.Context, req *proto.ListActivityLogsRequest) (*proto.ListActivityLogsResponse, error) {
	logger := hlog.FromContext(ctx).Named("ActivityLogService.List").
		With(hlog.Uint64("created_by", req.GetCreatedBy())).
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
		return nil, fmt.Errorf("%w: list activity logs: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.ActivityLogItem, 0, len(rows))
	for i := range rows {
		items = append(items, activityLogToProto(&rows[i]))
	}
	return &proto.ListActivityLogsResponse{
		Items:    items,
		Page:     int32(s.filter.Page),
		PageSize: int32(s.filter.PageSize),
		Total:    total,
		HasMore:  int64(s.filter.Page)*int64(s.filter.PageSize) < total,
	}, nil
}

func (s *ActivityLogService) Get(ctx context.Context, req *proto.GetActivityLogRequest) (*proto.ActivityLogItem, error) {
	id, err := parseActivityLogID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return activityLogToProto(row), nil
}

func (s *ActivityLogService) Create(ctx context.Context, req *proto.CreateActivityLogRequest) (*proto.ActivityLogItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	row, err := buildActivityLog(req.GetContent(), req.GetOccurredAt(), req.GetDurationMinutes(), req.GetInputType(), req.GetAudioUrl())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row.CreatedBy = authctx.UserIDOrDefault(ctx, defaultActivityLogCreator)
	if err := s.repository.Create(ctx, s.db, row); err != nil {
		return nil, fmt.Errorf("%w: create activity log: %v", hecode.ErrDBInsert, err)
	}
	created, err := s.getByID(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return activityLogToProto(created), nil
}

func (s *ActivityLogService) Update(ctx context.Context, req *proto.UpdateActivityLogRequest) (*proto.ActivityLogItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseActivityLogID(req.GetId())
	if err != nil {
		return nil, err
	}
	input, err := buildActivityLog(req.GetContent(), req.GetOccurredAt(), req.GetDurationMinutes(), req.GetInputType(), req.GetAudioUrl())
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
	current.Content = input.Content
	current.OccurredAt = input.OccurredAt
	current.DurationMinutes = input.DurationMinutes
	current.InputType = input.InputType
	current.AudioURL = input.AudioURL
	if err := s.repository.Update(ctx, s.db, &current.ActivityLog); err != nil {
		return nil, fmt.Errorf("%w: update activity log: %v", hecode.ErrDBUpdate, err)
	}
	updated, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return activityLogToProto(updated), nil
}

func (s *ActivityLogService) Delete(ctx context.Context, req *proto.DeleteActivityLogRequest) (*proto.DeleteActivityLogResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseActivityLogID(req.GetId())
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
		return nil, fmt.Errorf("%w: delete activity log: %v", hecode.ErrDBDelete, err)
	}
	return &proto.DeleteActivityLogResponse{Deleted: true}, nil
}

func (s *ActivityLogService) Stats(ctx context.Context, req *proto.ActivityLogStatsRequest) (*proto.ActivityLogStatsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	scope, year, month, from, to, err := buildActivityStatsRange(req)
	if err != nil {
		return nil, err
	}
	rows, err := s.repository.ListForStats(ctx, s.db, req.GetCreatedBy(), from, to)
	if err != nil {
		return nil, fmt.Errorf("%w: stat activity logs: %v", hecode.ErrDBQuery, err)
	}
	return buildActivityLogStats(rows, scope, year, month), nil
}

func (s *ActivityLogService) checkDependencies() error {
	if s == nil || s.db == nil || s.repository == nil {
		return fmt.Errorf("%w: activity log service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func (s *ActivityLogService) getByID(ctx context.Context, id uint64) (*dao.ActivityLogWithCreator, error) {
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row, err := s.repository.GetByID(ctx, s.db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: activity log %d", hecode.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get activity log: %v", hecode.ErrDBQuery, err)
	}
	return row, nil
}

func parseActivityLogID(value string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid activity log id", hecode.ErrInvalidParam)
	}
	return id, nil
}

func parseOptionalRFC3339(value, field string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s must be RFC3339", hecode.ErrInvalidParam, field)
	}
	return &parsed, nil
}

func buildActivityLog(content, occurredAt string, durationMinutes int32, inputType, audioURL string) (*model.ActivityLog, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: content is required", hecode.ErrInvalidParam)
	}
	if utf8.RuneCountInString(content) > maxActivityLogContentLen {
		return nil, fmt.Errorf("%w: content is too long", hecode.ErrInvalidParam)
	}
	occurred, err := time.Parse(time.RFC3339, strings.TrimSpace(occurredAt))
	if err != nil {
		return nil, fmt.Errorf("%w: occurredAt must be RFC3339", hecode.ErrInvalidParam)
	}
	if durationMinutes < 1 || durationMinutes > maxActivityLogDuration {
		return nil, fmt.Errorf("%w: durationMinutes must be between 1 and %d", hecode.ErrInvalidParam, maxActivityLogDuration)
	}
	inputType = strings.TrimSpace(inputType)
	if inputType == "" {
		inputType = model.ActivityInputTypeText
	}
	if inputType != model.ActivityInputTypeText && inputType != model.ActivityInputTypeVoice {
		return nil, fmt.Errorf("%w: unsupported input type", hecode.ErrInvalidParam)
	}
	audioURL = strings.TrimSpace(audioURL)
	if len(audioURL) > 512 {
		return nil, fmt.Errorf("%w: audioUrl is too long", hecode.ErrInvalidParam)
	}
	row := &model.ActivityLog{Content: content, OccurredAt: occurred, InputType: inputType}
	duration := uint16(durationMinutes)
	row.DurationMinutes = &duration
	if audioURL != "" {
		row.AudioURL = &audioURL
	}
	return row, nil
}

func buildActivityStatsRange(req *proto.ActivityLogStatsRequest) (string, int, int, time.Time, time.Time, error) {
	now := time.Now()
	scope := strings.TrimSpace(req.GetScope())
	if scope == "" {
		scope = "month"
	}
	if scope != "month" && scope != "year" {
		return "", 0, 0, time.Time{}, time.Time{}, fmt.Errorf("%w: unsupported scope", hecode.ErrInvalidParam)
	}
	year := int(req.GetYear())
	if year == 0 {
		year = now.Year()
	}
	if year < 2000 || year > 2100 {
		return "", 0, 0, time.Time{}, time.Time{}, fmt.Errorf("%w: invalid year", hecode.ErrInvalidParam)
	}
	month := int(req.GetMonth())
	if scope == "month" {
		if month == 0 {
			month = int(now.Month())
		}
		if month < 1 || month > 12 {
			return "", 0, 0, time.Time{}, time.Time{}, fmt.Errorf("%w: invalid month", hecode.ErrInvalidParam)
		}
		from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		return scope, year, month, from, from.AddDate(0, 1, 0), nil
	}
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	return scope, year, 0, from, from.AddDate(1, 0, 0), nil
}

func buildActivityLogStats(rows []model.ActivityLog, scope string, year, month int) *proto.ActivityLogStatsResponse {
	response := &proto.ActivityLogStatsResponse{
		RecordCount:      int32(len(rows)),
		DailyBreakdown:   make([]*proto.ActivityLogDailyBreakdown, 0),
		MonthlyBreakdown: make([]*proto.ActivityLogMonthlyBreakdown, 12),
	}
	activeDates := make(map[string]struct{})
	daily := make(map[string]*proto.ActivityLogDailyBreakdown)
	for index := 1; index <= 12; index++ {
		response.MonthlyBreakdown[index-1] = &proto.ActivityLogMonthlyBreakdown{
			Month: int32(index), Label: fmt.Sprintf("%d月", index),
		}
	}
	for i := range rows {
		occurred := rows[i].OccurredAt.In(time.Local)
		dateKey := occurred.Format("2006-01-02")
		activeDates[dateKey] = struct{}{}
		minutes := int32(0)
		if rows[i].DurationMinutes != nil {
			minutes = int32(*rows[i].DurationMinutes)
			response.TotalMinutes += minutes
			response.DurationRecordCount++
			if minutes > response.LongestMinutes {
				response.LongestMinutes = minutes
			}
		}
		monthItem := response.MonthlyBreakdown[int(occurred.Month())-1]
		monthItem.RecordCount++
		monthItem.Minutes += minutes
		if scope == "month" && occurred.Year() == year && int(occurred.Month()) == month {
			item := daily[dateKey]
			if item == nil {
				item = &proto.ActivityLogDailyBreakdown{
					Key: dateKey, Label: fmt.Sprintf("%d/%d", occurred.Month(), occurred.Day()),
				}
				daily[dateKey] = item
				response.DailyBreakdown = append(response.DailyBreakdown, item)
			}
			item.RecordCount++
			item.Minutes += minutes
		}
	}
	response.ActiveDays = int32(len(activeDates))
	if response.DurationRecordCount > 0 {
		response.AverageMinutes = int32((int64(response.TotalMinutes) + int64(response.DurationRecordCount)/2) / int64(response.DurationRecordCount))
	}
	return response
}

func activityLogToProto(row *dao.ActivityLogWithCreator) *proto.ActivityLogItem {
	if row == nil {
		return nil
	}
	item := &proto.ActivityLogItem{
		Id:         strconv.FormatUint(row.ID, 10),
		Content:    row.Content,
		OccurredAt: formatTime(row.OccurredAt),
		InputType:  row.InputType,
		CreatedBy:  strconv.FormatUint(row.CreatedBy, 10),
		CreatedAt:  formatTime(row.CreatedAt),
		UpdatedAt:  formatTime(row.UpdatedAt),
		Creator: &proto.ActivityLogCreator{
			Id: strconv.FormatUint(row.CreatedBy, 10), Nickname: row.CreatorNickname,
			AvatarEmoji: row.CreatorEmoji, AvatarColor: row.CreatorColor,
		},
	}
	if row.DurationMinutes != nil {
		item.DurationMinutes = int32(*row.DurationMinutes)
	}
	if row.AudioURL != nil {
		item.AudioUrl = *row.AudioURL
	}
	return item
}
