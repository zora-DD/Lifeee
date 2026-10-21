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
	defaultTaskPageSize = 20
	maxTaskPageSize     = 100
	defaultTaskCreator  = uint64(1)
	maxTaskTitleLength  = 120
	maxTaskDescLength   = 2000
	maxTaskCategoryLen  = 32
)

// TaskRepository 定义任务服务依赖的数据访问能力。
type TaskRepository interface {
	Create(ctx context.Context, db *gorm.DB, row *model.Task) error
	GetByID(ctx context.Context, db *gorm.DB, id uint64) (*dao.TaskWithCreator, error)
	Update(ctx context.Context, db *gorm.DB, row *model.Task) error
	UpdateStatus(ctx context.Context, db *gorm.DB, id uint64, status string) error
	SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error
	List(ctx context.Context, db *gorm.DB, in dao.TaskListQuery) ([]dao.TaskWithCreator, int64, error)
	Stats(ctx context.Context, db *gorm.DB, createdBy uint64) (dao.TaskStats, error)
}

// TaskService 负责任务参数校验和业务编排。
type TaskService struct {
	db         *gorm.DB
	repository TaskRepository
	filter     dao.TaskListQuery
	ready      bool
}

// NewTaskService 显式注入数据库和 DAO，依赖组装由 Control 层完成。
func NewTaskService(db *gorm.DB, repository TaskRepository) *TaskService {
	return &TaskService{db: db, repository: repository}
}

// InitAndCheckParam 清洗并校验任务列表参数。
func (s *TaskService) InitAndCheckParam(_ context.Context, req *proto.ListTasksRequest) error {
	if req == nil {
		return fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	status := strings.TrimSpace(req.GetStatus())
	if status != "" && status != model.TaskStatusTodo && status != model.TaskStatusDone {
		return fmt.Errorf("%w: unsupported status", hecode.ErrInvalidParam)
	}
	priority := strings.TrimSpace(req.GetPriority())
	if priority != "" &&
		priority != model.TaskPriorityHigh &&
		priority != model.TaskPriorityMedium &&
		priority != model.TaskPriorityLow {
		return fmt.Errorf("%w: unsupported priority", hecode.ErrInvalidParam)
	}
	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}
	if page < 1 {
		return fmt.Errorf("%w: page must be greater than zero", hecode.ErrInvalidParam)
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultTaskPageSize
	}
	if pageSize < 1 || pageSize > maxTaskPageSize {
		return fmt.Errorf("%w: pageSize must be between 1 and %d", hecode.ErrInvalidParam, maxTaskPageSize)
	}
	dueFrom, err := parseOptionalTaskTime(req.GetDueFrom())
	if err != nil {
		return fmt.Errorf("%w: invalid dueFrom", hecode.ErrInvalidParam)
	}
	dueTo, err := parseOptionalTaskTime(req.GetDueTo())
	if err != nil {
		return fmt.Errorf("%w: invalid dueTo", hecode.ErrInvalidParam)
	}
	if dueFrom != nil && dueTo != nil && dueFrom.After(*dueTo) {
		return fmt.Errorf("%w: dueFrom must not be later than dueTo", hecode.ErrInvalidParam)
	}
	s.filter = dao.TaskListQuery{
		Status:    status,
		Priority:  priority,
		Category:  strings.TrimSpace(req.GetCategory()),
		Keyword:   strings.TrimSpace(req.GetKeyword()),
		CreatedBy: req.GetCreatedBy(),
		DueFrom:   dueFrom,
		DueTo:     dueTo,
		Page:      page,
		PageSize:  pageSize,
	}
	s.ready = true
	return nil
}

func parseOptionalTaskTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// List 查询任务并组装接口响应。
func (s *TaskService) List(ctx context.Context, req *proto.ListTasksRequest) (*proto.ListTasksResponse, error) {
	logger := hlog.FromContext(ctx).Named("TaskService.List").
		With(hlog.String("status", req.GetStatus())).
		With(hlog.String("priority", req.GetPriority())).
		With(hlog.Uint64("created_by", req.GetCreatedBy())).
		With(hlog.Int32("page", req.GetPage())).
		With(hlog.Int32("page_size", req.GetPageSize()))
	if s == nil || !s.ready {
		return nil, fmt.Errorf("%w: InitAndCheckParam must be called first", hecode.ErrService)
	}
	if s.db == nil || s.repository == nil {
		return nil, fmt.Errorf("%w: task service dependencies are not initialized", hecode.ErrService)
	}
	rows, total, err := s.repository.List(ctx, s.db, s.filter)
	if err != nil {
		logger.Error("repository List failed", hlog.Err(err))
		return nil, fmt.Errorf("%w: list tasks: %v", hecode.ErrDBQuery, err)
	}
	stats, err := s.repository.Stats(ctx, s.db, s.filter.CreatedBy)
	if err != nil {
		logger.Error("repository Stats failed", hlog.Err(err))
		return nil, fmt.Errorf("%w: summarize tasks: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.TaskItem, 0, len(rows))
	for i := range rows {
		items = append(items, taskToProto(&rows[i]))
	}
	return &proto.ListTasksResponse{
		Items:        items,
		Page:         int32(s.filter.Page),
		PageSize:     int32(s.filter.PageSize),
		Total:        total,
		HasMore:      int64(s.filter.Page)*int64(s.filter.PageSize) < total,
		OverallTotal: stats.OverallTotal,
		TodoCount:    stats.TodoCount,
		DoneCount:    stats.DoneCount,
	}, nil
}

// Get 查询单个任务详情。
func (s *TaskService) Get(ctx context.Context, req *proto.GetTaskRequest) (*proto.TaskItem, error) {
	id, err := parseTaskID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return taskToProto(row), nil
}

// Create 创建任务。登录模块接入前，创建人暂时使用默认用户 ID。
func (s *TaskService) Create(ctx context.Context, req *proto.CreateTaskRequest) (*proto.TaskItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	row, err := buildTask(req.GetTitle(), req.GetDescription(), req.GetPriority(), req.GetCategory(), req.GetDueAt())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row.Status = model.TaskStatusTodo
	row.CreatedBy = authctx.UserIDOrDefault(ctx, defaultTaskCreator)
	if err := s.repository.Create(ctx, s.db, row); err != nil {
		return nil, fmt.Errorf("%w: create task: %v", hecode.ErrDBInsert, err)
	}
	created, err := s.getByID(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return taskToProto(created), nil
}

// Update 更新任务的可编辑字段，任务状态由独立接口维护。
func (s *TaskService) Update(ctx context.Context, req *proto.UpdateTaskRequest) (*proto.TaskItem, error) {
	id, err := parseTaskID(req.GetId())
	if err != nil {
		return nil, err
	}
	input, err := buildTask(req.GetTitle(), req.GetDescription(), req.GetPriority(), req.GetCategory(), req.GetDueAt())
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
	current.Title = input.Title
	current.Description = input.Description
	current.Priority = input.Priority
	current.Category = input.Category
	current.DueAt = input.DueAt
	if err := s.repository.Update(ctx, s.db, &current.Task); err != nil {
		return nil, fmt.Errorf("%w: update task: %v", hecode.ErrDBUpdate, err)
	}
	updated, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return taskToProto(updated), nil
}

// UpdateStatus 更新任务完成状态。
func (s *TaskService) UpdateStatus(ctx context.Context, req *proto.UpdateTaskStatusRequest) (*proto.TaskItem, error) {
	id, err := parseTaskID(req.GetId())
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(req.GetStatus())
	if status != model.TaskStatusTodo && status != model.TaskStatusDone {
		return nil, fmt.Errorf("%w: unsupported status", hecode.ErrInvalidParam)
	}
	current, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireOwnerOrAdmin(ctx, current.CreatedBy); err != nil {
		return nil, err
	}
	if err := s.repository.UpdateStatus(ctx, s.db, id, status); err != nil {
		return nil, fmt.Errorf("%w: update task status: %v", hecode.ErrDBUpdate, err)
	}
	updated, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return taskToProto(updated), nil
}

// Delete 软删除任务。
func (s *TaskService) Delete(ctx context.Context, req *proto.DeleteTaskRequest) (*proto.DeleteTaskResponse, error) {
	id, err := parseTaskID(req.GetId())
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
		return nil, fmt.Errorf("%w: delete task: %v", hecode.ErrDBDelete, err)
	}
	return &proto.DeleteTaskResponse{Deleted: true}, nil
}

func (s *TaskService) checkDependencies() error {
	if s == nil || s.db == nil || s.repository == nil {
		return fmt.Errorf("%w: task service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func (s *TaskService) getByID(ctx context.Context, id uint64) (*dao.TaskWithCreator, error) {
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row, err := s.repository.GetByID(ctx, s.db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: task %d", hecode.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get task: %v", hecode.ErrDBQuery, err)
	}
	return row, nil
}

func parseTaskID(value string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid task id", hecode.ErrInvalidParam)
	}
	return id, nil
}

func buildTask(title, description, priority, category, dueAt string) (*model.Task, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	priority = strings.TrimSpace(priority)
	category = strings.TrimSpace(category)
	dueAt = strings.TrimSpace(dueAt)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", hecode.ErrInvalidParam)
	}
	if utf8.RuneCountInString(title) > maxTaskTitleLength {
		return nil, fmt.Errorf("%w: title is too long", hecode.ErrInvalidParam)
	}
	if utf8.RuneCountInString(description) > maxTaskDescLength {
		return nil, fmt.Errorf("%w: description is too long", hecode.ErrInvalidParam)
	}
	if priority == "" {
		priority = model.TaskPriorityMedium
	}
	if priority != model.TaskPriorityHigh && priority != model.TaskPriorityMedium && priority != model.TaskPriorityLow {
		return nil, fmt.Errorf("%w: unsupported priority", hecode.ErrInvalidParam)
	}
	if category == "" {
		category = "其他"
	}
	if utf8.RuneCountInString(category) > maxTaskCategoryLen {
		return nil, fmt.Errorf("%w: category is too long", hecode.ErrInvalidParam)
	}
	row := &model.Task{Title: title, Priority: priority, Category: category}
	if description != "" {
		row.Description = &description
	}
	if dueAt != "" {
		value, err := time.Parse(time.RFC3339, dueAt)
		if err != nil {
			return nil, fmt.Errorf("%w: dueAt must be RFC3339", hecode.ErrInvalidParam)
		}
		row.DueAt = &value
	}
	return row, nil
}

func taskToProto(row *dao.TaskWithCreator) *proto.TaskItem {
	if row == nil {
		return nil
	}
	item := taskModelToProto(&row.Task)
	item.Creator = &proto.TaskCreator{
		Id: strconv.FormatUint(row.CreatedBy, 10), Nickname: row.CreatorNickname,
		AvatarEmoji: row.CreatorEmoji, AvatarColor: row.CreatorColor,
	}
	return item
}

func taskModelToProto(row *model.Task) *proto.TaskItem {
	if row == nil {
		return nil
	}
	item := &proto.TaskItem{
		Id:        strconv.FormatUint(row.ID, 10),
		Title:     row.Title,
		Status:    row.Status,
		Priority:  row.Priority,
		Category:  row.Category,
		CreatedBy: strconv.FormatUint(row.CreatedBy, 10),
		CreatedAt: formatTime(row.CreatedAt),
		UpdatedAt: formatTime(row.UpdatedAt),
	}
	if row.Description != nil {
		item.Description = *row.Description
	}
	if row.DueAt != nil {
		item.DueAt = formatTime(*row.DueAt)
	}
	return item
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
