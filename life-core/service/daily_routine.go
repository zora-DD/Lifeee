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
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/model"
	"life-core/proto"
)

const maxDailyRoutineTitleLength = 120

type DailyRoutineRepository interface {
	Create(context.Context, *gorm.DB, *model.DailyRoutine) error
	GetByIDAndUser(context.Context, *gorm.DB, uint64, uint64) (*model.DailyRoutine, error)
	ListByUser(context.Context, *gorm.DB, uint64, bool) ([]model.DailyRoutine, error)
	Update(context.Context, *gorm.DB, *model.DailyRoutine) error
	SoftDelete(context.Context, *gorm.DB, uint64, uint64) error
	ListCheckins(context.Context, *gorm.DB, uint64, []uint64, time.Time, time.Time) ([]model.DailyCheckin, error)
	UpsertCheckin(context.Context, *gorm.DB, *model.DailyCheckin) error
}

// DailyRoutineService 管理个人每日事项及最近七天打卡记录。
type DailyRoutineService struct {
	db         *gorm.DB
	repository DailyRoutineRepository
	now        func() time.Time
}

func NewDailyRoutineService(db *gorm.DB, repository DailyRoutineRepository) *DailyRoutineService {
	return &DailyRoutineService{db: db, repository: repository, now: time.Now}
}

func (s *DailyRoutineService) List(ctx context.Context, req *proto.ListDailyRoutinesRequest) (*proto.ListDailyRoutinesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	date, err := s.parseDateOrToday(req.GetDate())
	if err != nil {
		return nil, err
	}
	userID := authctx.UserIDOrDefault(ctx, defaultTaskCreator)
	rows, err := s.repository.ListByUser(ctx, s.db, userID, req.GetIncludeDisabled())
	if err != nil {
		return nil, fmt.Errorf("%w: list daily routines: %v", hecode.ErrDBQuery, err)
	}
	ids := make([]uint64, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	// 习惯数量很少，读取历史打卡用于累计次数、连续天数和月历统计。
	checkins, err := s.repository.ListCheckins(ctx, s.db, userID, ids, time.Time{}, date)
	if err != nil {
		return nil, fmt.Errorf("%w: list daily checkins: %v", hecode.ErrDBQuery, err)
	}
	byRoutineDate := make(map[string]model.DailyCheckin, len(checkins))
	for i := range checkins {
		key := fmt.Sprintf("%d:%s", checkins[i].RoutineID, checkins[i].CheckinDate.Format("2006-01-02"))
		byRoutineDate[key] = checkins[i]
	}
	items := make([]*proto.DailyRoutineItem, 0, len(rows))
	for i := range rows {
		items = append(items, dailyRoutineToProto(&rows[i], date, byRoutineDate))
	}
	return &proto.ListDailyRoutinesResponse{Items: items, Date: date.Format("2006-01-02")}, nil
}

func (s *DailyRoutineService) Create(ctx context.Context, req *proto.CreateDailyRoutineRequest) (*proto.DailyRoutineItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	title, timeOfDay, err := normalizeDailyRoutine(req.GetTitle(), req.GetTimeOfDay())
	if err != nil {
		return nil, err
	}
	repeatUnit, repeatInterval, startDate, err := normalizeRoutineSchedule(
		req.GetRepeatUnit(), req.GetRepeatInterval(), req.GetStartDate(), s.now(),
	)
	if err != nil {
		return nil, err
	}
	row := &model.DailyRoutine{
		Title: title, TimeOfDay: timeOfDay, RepeatUnit: repeatUnit, RepeatInterval: repeatInterval,
		StartDate: &startDate, Enabled: true,
		CreatedBy: authctx.UserIDOrDefault(ctx, defaultTaskCreator),
	}
	if err := s.repository.Create(ctx, s.db, row); err != nil {
		return nil, fmt.Errorf("%w: create daily routine: %v", hecode.ErrDBInsert, err)
	}
	return dailyRoutineToProto(row, s.now(), nil), nil
}

func (s *DailyRoutineService) Update(ctx context.Context, req *proto.UpdateDailyRoutineRequest) (*proto.DailyRoutineItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseDailyRoutineID(req.GetId())
	if err != nil {
		return nil, err
	}
	title, timeOfDay, err := normalizeDailyRoutine(req.GetTitle(), req.GetTimeOfDay())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	userID := authctx.UserIDOrDefault(ctx, defaultTaskCreator)
	existing, err := s.repository.GetByIDAndUser(ctx, s.db, id, userID)
	if err != nil {
		return nil, s.mapNotFound("get daily routine", id, err, hecode.ErrDBQuery)
	}
	repeatUnit := req.GetRepeatUnit()
	if repeatUnit == "" {
		repeatUnit = existing.RepeatUnit
	}
	repeatInterval := req.GetRepeatInterval()
	if repeatInterval == 0 {
		repeatInterval = existing.RepeatInterval
	}
	fallbackStart := s.now()
	if existing.StartDate != nil {
		fallbackStart = *existing.StartDate
	}
	repeatUnit, repeatInterval, startDate, err := normalizeRoutineSchedule(
		repeatUnit, repeatInterval, req.GetStartDate(), fallbackStart,
	)
	if err != nil {
		return nil, err
	}
	row := &model.DailyRoutine{
		ID: id, Title: title, TimeOfDay: timeOfDay, RepeatUnit: repeatUnit, RepeatInterval: repeatInterval,
		StartDate: &startDate, Enabled: req.GetEnabled(), CreatedBy: userID,
	}
	if err := s.repository.Update(ctx, s.db, row); err != nil {
		return nil, s.mapNotFound("update daily routine", id, err, hecode.ErrDBUpdate)
	}
	loaded, err := s.repository.GetByIDAndUser(ctx, s.db, id, userID)
	if err != nil {
		return nil, s.mapNotFound("get daily routine", id, err, hecode.ErrDBQuery)
	}
	return dailyRoutineToProto(loaded, s.now(), nil), nil
}

func (s *DailyRoutineService) Delete(ctx context.Context, req *proto.DeleteDailyRoutineRequest) (*proto.DeleteDailyRoutineResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseDailyRoutineID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	userID := authctx.UserIDOrDefault(ctx, defaultTaskCreator)
	if err := s.repository.SoftDelete(ctx, s.db, id, userID); err != nil {
		return nil, s.mapNotFound("delete daily routine", id, err, hecode.ErrDBDelete)
	}
	return &proto.DeleteDailyRoutineResponse{Deleted: true}, nil
}

func (s *DailyRoutineService) Checkin(ctx context.Context, req *proto.CheckinDailyRoutineRequest) (*proto.DailyRoutineItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseDailyRoutineID(req.GetId())
	if err != nil {
		return nil, err
	}
	date, err := s.parseDateOrToday(req.GetDate())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	userID := authctx.UserIDOrDefault(ctx, defaultTaskCreator)
	routine, err := s.repository.GetByIDAndUser(ctx, s.db, id, userID)
	if err != nil {
		return nil, s.mapNotFound("get daily routine", id, err, hecode.ErrDBQuery)
	}
	if !isRoutineScheduled(routine, date) {
		return nil, fmt.Errorf("%w: habit is not scheduled on %s", hecode.ErrInvalidParam, date.Format("2006-01-02"))
	}
	var checkedAt *time.Time
	if req.GetCompleted() {
		now := s.now()
		checkedAt = &now
	}
	row := &model.DailyCheckin{
		RoutineID: id, CheckinDate: date, Completed: req.GetCompleted(), CheckedAt: checkedAt, CreatedBy: userID,
	}
	if err := s.repository.UpsertCheckin(ctx, s.db, row); err != nil {
		return nil, fmt.Errorf("%w: save daily checkin: %v", hecode.ErrDBUpdate, err)
	}
	checkins := map[string]model.DailyCheckin{
		fmt.Sprintf("%d:%s", id, date.Format("2006-01-02")): *row,
	}
	return dailyRoutineToProto(routine, date, checkins), nil
}

func (s *DailyRoutineService) checkDependencies() error {
	if s == nil || s.db == nil || s.repository == nil || s.now == nil {
		return fmt.Errorf("%w: daily routine service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func (s *DailyRoutineService) parseDateOrToday(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := s.now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local), nil
	}
	date, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date must be YYYY-MM-DD", hecode.ErrInvalidParam)
	}
	return date, nil
}

func (s *DailyRoutineService) mapNotFound(action string, id uint64, err, code error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: daily routine %d", hecode.ErrNotFound, id)
	}
	return fmt.Errorf("%w: %s: %v", code, action, err)
}

func normalizeDailyRoutine(title, timeOfDay string) (string, string, error) {
	title = strings.TrimSpace(title)
	timeOfDay = strings.TrimSpace(timeOfDay)
	if title == "" {
		return "", "", fmt.Errorf("%w: title is required", hecode.ErrInvalidParam)
	}
	if utf8.RuneCountInString(title) > maxDailyRoutineTitleLength {
		return "", "", fmt.Errorf("%w: title is too long", hecode.ErrInvalidParam)
	}
	parsed, err := time.Parse("15:04", timeOfDay)
	if err != nil {
		return "", "", fmt.Errorf("%w: timeOfDay must be HH:mm", hecode.ErrInvalidParam)
	}
	return title, parsed.Format("15:04"), nil
}

func normalizeRoutineSchedule(unit string, interval int32, startDate string, fallback time.Time) (string, int32, time.Time, error) {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		unit = "day"
	}
	if unit != "day" && unit != "week" && unit != "month" {
		return "", 0, time.Time{}, fmt.Errorf("%w: repeatUnit must be day, week or month", hecode.ErrInvalidParam)
	}
	if interval == 0 {
		interval = 1
	}
	if interval < 1 || interval > 365 {
		return "", 0, time.Time{}, fmt.Errorf("%w: repeatInterval must be between 1 and 365", hecode.ErrInvalidParam)
	}
	startDate = strings.TrimSpace(startDate)
	if startDate == "" {
		fallback = normalizeRoutineDate(fallback)
		return unit, interval, fallback, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", startDate, time.Local)
	if err != nil {
		return "", 0, time.Time{}, fmt.Errorf("%w: startDate must be YYYY-MM-DD", hecode.ErrInvalidParam)
	}
	return unit, interval, parsed, nil
}

func parseDailyRoutineID(value string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid daily routine id", hecode.ErrInvalidParam)
	}
	return id, nil
}

func dailyRoutineToProto(row *model.DailyRoutine, date time.Time, checkins map[string]model.DailyCheckin) *proto.DailyRoutineItem {
	if row == nil {
		return nil
	}
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	item := &proto.DailyRoutineItem{
		Id: strconv.FormatUint(row.ID, 10), Title: row.Title, TimeOfDay: row.TimeOfDay, Enabled: row.Enabled,
		RepeatUnit: normalizedRepeatUnit(row.RepeatUnit), RepeatInterval: normalizedRepeatInterval(row.RepeatInterval),
		CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt),
		History: make([]*proto.DailyRoutineDay, 0, 7), MonthHistory: make([]*proto.DailyRoutineDay, 0, 31),
	}
	startDate := routineStartDate(row, date)
	item.StartDate = startDate.Format("2006-01-02")
	item.ScheduledToday = isRoutineScheduled(row, date)
	completedLast7 := 0
	scheduledLast7 := 0
	for offset := -6; offset <= 0; offset++ {
		day := date.AddDate(0, 0, offset)
		dayText := day.Format("2006-01-02")
		checkin, exists := checkins[fmt.Sprintf("%d:%s", row.ID, dayText)]
		history := &proto.DailyRoutineDay{
			Date: dayText, Completed: exists && checkin.Completed, Scheduled: isRoutineScheduled(row, day),
		}
		if history.Completed && checkin.CheckedAt != nil {
			history.CheckedAt = formatTime(*checkin.CheckedAt)
		}
		item.History = append(item.History, history)
		if history.Scheduled {
			scheduledLast7++
		}
		if history.Scheduled && history.Completed {
			completedLast7++
		}
		if offset == 0 {
			item.CheckedToday = history.Completed
			item.CheckedAt = history.CheckedAt
		}
	}
	if scheduledLast7 > 0 {
		item.LastSevenDayRate = int32(completedLast7 * 100 / scheduledLast7)
	}

	for _, checkin := range checkins {
		if checkin.RoutineID == row.ID && checkin.Completed && !checkin.CheckinDate.After(date) {
			item.TotalCheckins++
		}
	}

	streakDate := date
	if isRoutineScheduled(row, streakDate) && !routineCompletedOn(row.ID, streakDate, checkins) {
		streakDate = streakDate.AddDate(0, 0, -1)
	}
	for !streakDate.Before(startDate) {
		if !isRoutineScheduled(row, streakDate) {
			streakDate = streakDate.AddDate(0, 0, -1)
			continue
		}
		if !routineCompletedOn(row.ID, streakDate, checkins) {
			break
		}
		item.CurrentStreak++
		streakDate = streakDate.AddDate(0, 0, -1)
	}

	monthStart := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, -1)
	for day := monthStart; !day.After(monthEnd); day = day.AddDate(0, 0, 1) {
		dayText := day.Format("2006-01-02")
		checkin, exists := checkins[fmt.Sprintf("%d:%s", row.ID, dayText)]
		monthDay := &proto.DailyRoutineDay{
			Date: dayText, Completed: exists && checkin.Completed, Scheduled: isRoutineScheduled(row, day),
		}
		if monthDay.Completed && checkin.CheckedAt != nil {
			monthDay.CheckedAt = formatTime(*checkin.CheckedAt)
		}
		item.MonthHistory = append(item.MonthHistory, monthDay)
	}
	return item
}

func normalizedRepeatUnit(value string) string {
	if value == "week" || value == "month" {
		return value
	}
	return "day"
}

func normalizedRepeatInterval(value int32) int32 {
	if value > 0 {
		return value
	}
	return 1
}

func routineStartDate(row *model.DailyRoutine, fallback time.Time) time.Time {
	if row != nil && row.StartDate != nil && !row.StartDate.IsZero() {
		return normalizeRoutineDate(*row.StartDate)
	}
	if row != nil && !row.CreatedAt.IsZero() {
		return normalizeRoutineDate(row.CreatedAt)
	}
	return normalizeRoutineDate(fallback)
}

func isRoutineScheduled(row *model.DailyRoutine, date time.Time) bool {
	if row == nil {
		return false
	}
	date = normalizeRoutineDate(date)
	startDate := routineStartDate(row, date)
	if date.Before(startDate) {
		return false
	}
	interval := int(normalizedRepeatInterval(row.RepeatInterval))
	switch normalizedRepeatUnit(row.RepeatUnit) {
	case "week":
		return int(date.Sub(startDate).Hours()/24)%(7*interval) == 0
	case "month":
		monthDiff := (date.Year()-startDate.Year())*12 + int(date.Month()-startDate.Month())
		if monthDiff%interval != 0 {
			return false
		}
		return date.Day() == min(startDate.Day(), daysInMonth(date.Year(), date.Month()))
	default:
		return int(date.Sub(startDate).Hours()/24)%interval == 0
	}
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func normalizeRoutineDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.Local)
}

func routineCompletedOn(routineID uint64, date time.Time, checkins map[string]model.DailyCheckin) bool {
	checkin, exists := checkins[fmt.Sprintf("%d:%s", routineID, date.Format("2006-01-02"))]
	return exists && checkin.Completed
}
