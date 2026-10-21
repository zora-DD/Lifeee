package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vaynedu/hollow/pkg/hecode"
	"github.com/vaynedu/hollow/pkg/hlog"
	gproto "google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

const maxTaskSummaryFocusCount = 3

type AISummaryRepository interface {
	ListTasksForSummary(context.Context, *gorm.DB, time.Time, time.Time) ([]model.Task, error)
	GetTaskSummaryByDate(context.Context, *gorm.DB, time.Time) (*model.TaskAISummary, error)
	UpsertTaskSummary(context.Context, *gorm.DB, *model.TaskAISummary) error
	GetLifeSnapshot(context.Context, *gorm.DB, uint64, time.Time, time.Time) (*dao.LifeSummarySnapshot, error)
}

type AISummaryOrganizer interface {
	SummarizeTasks(context.Context, TaskSummaryFact) (*SummaryDraft, error)
	SummarizeLife(context.Context, LifeSummaryFact) (*SummaryDraft, error)
}

type taskSummaryCacheEntry struct {
	response  *proto.TaskAISummaryResponse
	expiresAt time.Time
}

type lifeSummaryCacheEntry struct {
	response  *proto.LifeAISummaryResponse
	expiresAt time.Time
}

// AISummaryCache 保存同一天、同一份事实的生成结果，避免重复调用模型。
type AISummaryCache struct {
	mu    sync.Mutex
	tasks map[string]taskSummaryCacheEntry
	life  map[string]lifeSummaryCacheEntry
}

func NewAISummaryCache() *AISummaryCache {
	return &AISummaryCache{
		tasks: make(map[string]taskSummaryCacheEntry),
		life:  make(map[string]lifeSummaryCacheEntry),
	}
}

type AISummaryService struct {
	db         *gorm.DB
	repository AISummaryRepository
	organizer  AISummaryOrganizer
	cache      *AISummaryCache
	now        func() time.Time
	taskMu     sync.Mutex
}

func NewAISummaryService(db *gorm.DB, repository AISummaryRepository, organizer AISummaryOrganizer, cache *AISummaryCache) *AISummaryService {
	if cache == nil {
		cache = NewAISummaryCache()
	}
	return &AISummaryService{db: db, repository: repository, organizer: organizer, cache: cache, now: time.Now}
}

// TaskSummary 汇总公开任务中的逾期、今日截止和临近截止事项。
func (s *AISummaryService) TaskSummary(ctx context.Context, req *proto.GetTaskAISummaryRequest) (*proto.TaskAISummaryResponse, error) {
	_, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	return s.GenerateDailyTaskSummary(ctx, req != nil && req.GetRefresh())
}

// GenerateDailyTaskSummary 供 HTTP 接口和定时任务共同生成当天全局任务总结。
// refresh=false 时会优先复用事实一致的内存或数据库结果。
func (s *AISummaryService) GenerateDailyTaskSummary(ctx context.Context, refresh bool) (*proto.TaskAISummaryResponse, error) {
	if s == nil || s.db == nil || s.repository == nil || s.cache == nil || s.now == nil {
		return nil, fmt.Errorf("%w: ai summary dependencies are not initialized", hecode.ErrService)
	}
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	now := s.now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	rows, err := s.repository.ListTasksForSummary(ctx, s.db, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("%w: list task summary facts: %v", hecode.ErrDBQuery, err)
	}
	response, facts := buildTaskSummary(rows, now)
	factsHash := taskFactsHash(rows, facts)
	key := "task:" + response.Date + ":" + factsHash
	if !refresh {
		if cached := s.cache.getTask(key, now); cached != nil {
			return cached, nil
		}
		persisted, findErr := s.repository.GetTaskSummaryByDate(ctx, s.db, dayStart)
		switch {
		case findErr == nil && persisted != nil && persisted.FactsHash == factsHash:
			var stored proto.TaskAISummaryResponse
			if json.Unmarshal(persisted.PayloadJSON, &stored) == nil {
				s.cache.putTask(key, &stored, dayEnd)
				return cloneTaskSummary(&stored, true), nil
			}
		case findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound):
			return nil, fmt.Errorf("%w: get persisted task summary: %v", hecode.ErrDBQuery, findErr)
		}
	}

	copy := fallbackTaskSummary(response)
	if s.organizer != nil {
		if generated, generateErr := s.organizer.SummarizeTasks(ctx, facts); generateErr == nil && generated != nil {
			copy.Summary = generated.Summary
			copy.Suggestions = append([]string(nil), generated.Suggestions...)
			copy.AiGenerated = true
		} else if generateErr != nil {
			hlog.FromContext(ctx).Named("AISummaryService.GenerateDailyTaskSummary").Error(
				"AI generation failed, using rule fallback", hlog.Err(generateErr))
		}
	}
	copy.GeneratedAt = formatTime(now)
	payload, err := json.Marshal(copy)
	if err != nil {
		return nil, fmt.Errorf("%w: encode task summary: %v", hecode.ErrService, err)
	}
	if err := s.repository.UpsertTaskSummary(ctx, s.db, &model.TaskAISummary{
		SummaryDate: dayStart,
		FactsHash:   factsHash,
		PayloadJSON: payload,
		AIGenerated: copy.AiGenerated,
		GeneratedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("%w: save task summary: %v", hecode.ErrDBQuery, err)
	}
	s.cache.putTask(key, copy, dayEnd)
	return cloneTaskSummary(copy, false), nil
}

// LifeSummary 汇总当前用户指定月份或年份的生活事实。
func (s *AISummaryService) LifeSummary(ctx context.Context, req *proto.GetLifeAISummaryRequest) (*proto.LifeAISummaryResponse, error) {
	userID, ok := authctx.UserIDFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	if s == nil || s.db == nil || s.repository == nil || s.cache == nil || s.now == nil {
		return nil, fmt.Errorf("%w: ai summary dependencies are not initialized", hecode.ErrService)
	}
	scope, year, month, from, to, periodLabel, err := buildLifeSummaryRange(req, s.now())
	if err != nil {
		return nil, err
	}
	snapshot, err := s.repository.GetLifeSnapshot(ctx, s.db, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("%w: get life summary facts: %v", hecode.ErrDBQuery, err)
	}
	expenseTotal := normalizeAggregateMoney(snapshot.ExpenseTotal)
	facts := LifeSummaryFact{
		PeriodLabel: periodLabel, TaskCreatedCount: int(snapshot.TaskCreatedCount), TaskDoneCount: int(snapshot.TaskDoneCount),
		ActivityCount: int(snapshot.ActivityCount), ActivityMinutes: int(snapshot.ActivityMinutes), BillCount: int(snapshot.BillCount),
		ExpenseTotal: expenseTotal, RestaurantVisitCount: int(snapshot.RestaurantVisitCount), VisitedPlaceCount: int(snapshot.VisitedPlaceCount),
	}
	key := summaryCacheKey(fmt.Sprintf("life:%s:%d:%d", scope, year, month), userID, facts)
	now := s.now()
	if req == nil || !req.GetRefresh() {
		if cached := s.cache.getLife(key, now); cached != nil {
			return cached, nil
		}
	}
	response := fallbackLifeSummary(scope, year, month, facts)
	if s.organizer != nil {
		if generated, generateErr := s.organizer.SummarizeLife(ctx, facts); generateErr == nil && generated != nil {
			response.Summary = generated.Summary
			response.Highlights = append([]string(nil), generated.Highlights...)
			response.Suggestions = append([]string(nil), generated.Suggestions...)
			response.AiGenerated = true
		}
	}
	response.GeneratedAt = formatTime(now)
	nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	s.cache.putLife(key, response, nextDay)
	return cloneLifeSummary(response, false), nil
}

func buildTaskSummary(rows []model.Task, now time.Time) (*proto.TaskAISummaryResponse, TaskSummaryFact) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	upcomingEnd := dayEnd.AddDate(0, 0, 3)
	type focusCandidate struct {
		item  *proto.TaskSummaryFocusItem
		score int
		dueAt time.Time
	}
	response := &proto.TaskAISummaryResponse{Date: dayStart.Format("2006-01-02"), FocusTasks: make([]*proto.TaskSummaryFocusItem, 0), Suggestions: make([]string, 0)}
	candidates := make([]focusCandidate, 0)
	for i := range rows {
		row := &rows[i]
		if row.Status == model.TaskStatusDone {
			if !row.UpdatedAt.Before(dayStart) && row.UpdatedAt.Before(dayEnd) {
				response.DoneTodayCount++
			}
			continue
		}
		response.TodoCount++
		if row.Priority == model.TaskPriorityHigh {
			response.HighPriorityCount++
		}
		reason, score := "待处理", 5
		due := time.Time{}
		if row.DueAt != nil {
			due = *row.DueAt
			switch {
			case due.Before(now):
				response.OverdueCount++
				reason, score = "已逾期", 100
			case due.Before(dayEnd):
				response.DueTodayCount++
				reason, score = "今天截止", 80
			case due.Before(upcomingEnd):
				response.UpcomingCount++
				reason, score = "3天内截止", 50
			}
		}
		if row.Priority == model.TaskPriorityHigh && score < 70 {
			reason, score = "高优先级", 70
		}
		item := &proto.TaskSummaryFocusItem{Id: strconv.FormatUint(row.ID, 10), Title: row.Title, Priority: row.Priority, Reason: reason}
		if row.DueAt != nil {
			item.DueAt = formatTime(*row.DueAt)
		}
		candidates = append(candidates, focusCandidate{item: item, score: score, dueAt: due})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].dueAt.IsZero() != candidates[j].dueAt.IsZero() {
			return !candidates[i].dueAt.IsZero()
		}
		return candidates[i].dueAt.Before(candidates[j].dueAt)
	})
	for i := 0; i < len(candidates) && i < maxTaskSummaryFocusCount; i++ {
		response.FocusTasks = append(response.FocusTasks, candidates[i].item)
	}
	facts := TaskSummaryFact{
		Date: response.Date, TodoCount: int(response.TodoCount), DoneTodayCount: int(response.DoneTodayCount),
		OverdueCount: int(response.OverdueCount), DueTodayCount: int(response.DueTodayCount),
		UpcomingCount: int(response.UpcomingCount), HighPriorityCount: int(response.HighPriorityCount),
		FocusTasks: make([]TaskFocusFact, 0, len(response.FocusTasks)),
	}
	for _, item := range response.FocusTasks {
		facts.FocusTasks = append(facts.FocusTasks, TaskFocusFact{Title: item.Title, Priority: item.Priority, DueAt: item.DueAt, Reason: item.Reason})
	}
	return response, facts
}

func fallbackTaskSummary(response *proto.TaskAISummaryResponse) *proto.TaskAISummaryResponse {
	result := cloneTaskSummary(response, false)
	if result.TodoCount == 0 {
		result.Summary = "今天没有待处理任务，可以轻松收尾，也可以提前安排明天。"
		result.Suggestions = []string{"回顾今天完成的事情"}
		return result
	}
	result.Summary = fmt.Sprintf("当前还有 %d 项待处理，其中 %d 项逾期、%d 项今天截止。", result.TodoCount, result.OverdueCount, result.DueTodayCount)
	result.Suggestions = make([]string, 0, len(result.FocusTasks))
	for _, item := range result.FocusTasks {
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("先处理“%s”（%s）", item.Title, item.Reason))
	}
	return result
}

func fallbackLifeSummary(scope string, year, month int, facts LifeSummaryFact) *proto.LifeAISummaryResponse {
	response := &proto.LifeAISummaryResponse{
		Scope: scope, Year: int32(year), Month: int32(month), PeriodLabel: facts.PeriodLabel,
		TaskCreatedCount: int32(facts.TaskCreatedCount), TaskDoneCount: int32(facts.TaskDoneCount),
		ActivityCount: int32(facts.ActivityCount), ActivityMinutes: int32(facts.ActivityMinutes),
		BillCount: int32(facts.BillCount), ExpenseTotal: facts.ExpenseTotal,
		RestaurantVisitCount: int32(facts.RestaurantVisitCount), VisitedPlaceCount: int32(facts.VisitedPlaceCount),
		Highlights: make([]string, 0), Suggestions: make([]string, 0),
	}
	response.Summary = fmt.Sprintf("%s完成了 %d 项任务，记录了 %d 件生活小事，消费 %s 元。", facts.PeriodLabel, facts.TaskDoneCount, facts.ActivityCount, facts.ExpenseTotal)
	if facts.ActivityMinutes > 0 {
		response.Highlights = append(response.Highlights, fmt.Sprintf("为生活事项投入了 %d 分钟", facts.ActivityMinutes))
	}
	if facts.RestaurantVisitCount+facts.VisitedPlaceCount > 0 {
		response.Highlights = append(response.Highlights, fmt.Sprintf("留下了 %d 次吃喝与地点足迹", facts.RestaurantVisitCount+facts.VisitedPlaceCount))
	}
	if facts.ActivityCount == 0 {
		response.Suggestions = append(response.Suggestions, "可以从每天记录一件小事开始")
	}
	if facts.TaskCreatedCount > facts.TaskDoneCount {
		response.Suggestions = append(response.Suggestions, "挑一项最重要的未完成任务优先推进")
	}
	if facts.BillCount > 0 {
		response.Suggestions = append(response.Suggestions, "继续保持记录，月底会更容易看清消费习惯")
	}
	return response
}

func buildLifeSummaryRange(req *proto.GetLifeAISummaryRequest, now time.Time) (string, int, int, time.Time, time.Time, string, error) {
	if req == nil {
		req = new(proto.GetLifeAISummaryRequest)
	}
	scope := strings.TrimSpace(req.GetScope())
	if scope == "" {
		scope = "month"
	}
	if scope != "month" && scope != "year" {
		return "", 0, 0, time.Time{}, time.Time{}, "", fmt.Errorf("%w: unsupported life summary scope", hecode.ErrInvalidParam)
	}
	year, month := int(req.GetYear()), int(req.GetMonth())
	if year == 0 {
		year = now.Year()
	}
	if month == 0 && scope == "month" {
		month = int(now.Month())
	}
	if year < 2000 || year > 2100 || (scope == "month" && (month < 1 || month > 12)) {
		return "", 0, 0, time.Time{}, time.Time{}, "", fmt.Errorf("%w: invalid year or month", hecode.ErrInvalidParam)
	}
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(1, 0, 0)
	label := fmt.Sprintf("%d年", year)
	if scope == "month" {
		from = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 1, 0)
		label = fmt.Sprintf("%d年%d月", year, month)
	} else {
		month = 0
	}
	return scope, year, month, from, to, label, nil
}

func summaryCacheKey(kind string, userID uint64, facts any) string {
	body, _ := json.Marshal(facts)
	digest := sha256.Sum256(body)
	return fmt.Sprintf("%s:%d:%s", kind, userID, hex.EncodeToString(digest[:8]))
}

func taskFactsHash(rows []model.Task, facts TaskSummaryFact) string {
	type taskFactRow struct {
		ID        uint64     `json:"id"`
		Title     string     `json:"title"`
		Status    string     `json:"status"`
		Priority  string     `json:"priority"`
		DueAt     *time.Time `json:"due_at,omitempty"`
		UpdatedAt time.Time  `json:"updated_at"`
	}
	normalized := make([]taskFactRow, 0, len(rows))
	for i := range rows {
		normalized = append(normalized, taskFactRow{
			ID: rows[i].ID, Title: rows[i].Title, Status: rows[i].Status,
			Priority: rows[i].Priority, DueAt: rows[i].DueAt, UpdatedAt: rows[i].UpdatedAt,
		})
	}
	body, _ := json.Marshal(struct {
		Rows  []taskFactRow   `json:"rows"`
		Facts TaskSummaryFact `json:"facts"`
	}{Rows: normalized, Facts: facts})
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:8])
}

func (c *AISummaryCache) getTask(key string, now time.Time) *proto.TaskAISummaryResponse {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.tasks[key]
	if !ok || !now.Before(entry.expiresAt) {
		delete(c.tasks, key)
		return nil
	}
	return cloneTaskSummary(entry.response, true)
}

func (c *AISummaryCache) putTask(key string, response *proto.TaskAISummaryResponse, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tasks[key] = taskSummaryCacheEntry{response: cloneTaskSummary(response, false), expiresAt: expiresAt}
}

func (c *AISummaryCache) getLife(key string, now time.Time) *proto.LifeAISummaryResponse {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.life[key]
	if !ok || !now.Before(entry.expiresAt) {
		delete(c.life, key)
		return nil
	}
	return cloneLifeSummary(entry.response, true)
}

func (c *AISummaryCache) putLife(key string, response *proto.LifeAISummaryResponse, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.life[key] = lifeSummaryCacheEntry{response: cloneLifeSummary(response, false), expiresAt: expiresAt}
}

func cloneTaskSummary(source *proto.TaskAISummaryResponse, cached bool) *proto.TaskAISummaryResponse {
	if source == nil {
		return nil
	}
	result := gproto.Clone(source).(*proto.TaskAISummaryResponse)
	result.Cached = cached
	return result
}

func cloneLifeSummary(source *proto.LifeAISummaryResponse, cached bool) *proto.LifeAISummaryResponse {
	if source == nil {
		return nil
	}
	result := gproto.Clone(source).(*proto.LifeAISummaryResponse)
	result.Cached = cached
	return result
}
