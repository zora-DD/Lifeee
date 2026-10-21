package service

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type taskRepositoryStub struct {
	items          []dao.TaskWithCreator
	item           *dao.TaskWithCreator
	total          int64
	err            error
	getErr         error
	query          dao.TaskListQuery
	listInvoked    bool
	created        *model.Task
	updated        *model.Task
	status         string
	deletedID      uint64
	stats          dao.TaskStats
	statsCreatedBy uint64
}

func (s *taskRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.Task) error {
	if s.err != nil {
		return s.err
	}
	row.ID = 9
	s.created = row
	s.item = &dao.TaskWithCreator{
		Task:            *row,
		CreatorNickname: "晚风收藏家",
		CreatorEmoji:    "🌿",
		CreatorColor:    "#759B89",
	}
	return nil
}

func (s *taskRepositoryStub) GetByID(_ context.Context, _ *gorm.DB, _ uint64) (*dao.TaskWithCreator, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.item == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return s.item, nil
}

func (s *taskRepositoryStub) Update(_ context.Context, _ *gorm.DB, row *model.Task) error {
	if s.err != nil {
		return s.err
	}
	s.updated = row
	if s.item == nil {
		s.item = &dao.TaskWithCreator{}
	}
	s.item.Task = *row
	return nil
}

func (s *taskRepositoryStub) UpdateStatus(_ context.Context, _ *gorm.DB, _ uint64, status string) error {
	if s.err != nil {
		return s.err
	}
	s.status = status
	if s.item != nil {
		s.item.Status = status
	}
	return nil
}

func (s *taskRepositoryStub) SoftDelete(_ context.Context, _ *gorm.DB, id uint64) error {
	if s.err != nil {
		return s.err
	}
	s.deletedID = id
	return nil
}

func (s *taskRepositoryStub) List(
	_ context.Context,
	_ *gorm.DB,
	in dao.TaskListQuery,
) ([]dao.TaskWithCreator, int64, error) {
	s.listInvoked = true
	s.query = in
	return s.items, s.total, s.err
}

func (s *taskRepositoryStub) Stats(_ context.Context, _ *gorm.DB, createdBy uint64) (dao.TaskStats, error) {
	s.statsCreatedBy = createdBy
	return s.stats, s.err
}

func TestTaskServiceInitAndCheckParam(t *testing.T) {
	Convey("nil 请求返回参数错误", t, func() {
		err := NewTaskService(nil, nil).InitAndCheckParam(context.Background(), nil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("默认分页和文本清洗正确", t, func() {
		svc := NewTaskService(nil, nil)
		err := svc.InitAndCheckParam(context.Background(), &proto.ListTasksRequest{
			Status:   " todo ",
			Priority: " high ",
			Category: " 生活 ",
			Keyword:  " 水果 ",
		})

		So(err, ShouldBeNil)
		So(svc.filter.Status, ShouldEqual, model.TaskStatusTodo)
		So(svc.filter.Priority, ShouldEqual, model.TaskPriorityHigh)
		So(svc.filter.Category, ShouldEqual, "生活")
		So(svc.filter.Keyword, ShouldEqual, "水果")
		So(svc.filter.Page, ShouldEqual, 1)
		So(svc.filter.PageSize, ShouldEqual, 20)
		So(svc.ready, ShouldBeTrue)
	})

	Convey("非法枚举和分页参数返回错误", t, func() {
		requests := []*proto.ListTasksRequest{
			{Status: "doing"},
			{Priority: "urgent"},
			{Page: -1},
			{PageSize: 101},
		}
		for _, request := range requests {
			err := NewTaskService(nil, nil).InitAndCheckParam(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})

	Convey("截止时间范围会解析并传给 DAO", t, func() {
		svc := NewTaskService(nil, nil)
		err := svc.InitAndCheckParam(context.Background(), &proto.ListTasksRequest{
			DueFrom: "2026-09-14T00:00:00+08:00",
			DueTo:   "2026-09-21T00:00:00+08:00",
		})

		So(err, ShouldBeNil)
		So(svc.filter.DueFrom, ShouldNotBeNil)
		So(svc.filter.DueTo, ShouldNotBeNil)
		So(svc.filter.DueFrom.Before(*svc.filter.DueTo), ShouldBeTrue)
	})

	Convey("非法或倒置的截止时间范围返回参数错误", t, func() {
		requests := []*proto.ListTasksRequest{
			{DueFrom: "not-a-time"},
			{DueFrom: "2026-09-21T00:00:00+08:00", DueTo: "2026-09-14T00:00:00+08:00"},
		}
		for _, request := range requests {
			err := NewTaskService(nil, nil).InitAndCheckParam(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})
}

func TestTaskServiceList(t *testing.T) {
	Convey("未执行参数初始化时拒绝查询", t, func() {
		response, err := NewTaskService(new(gorm.DB), new(taskRepositoryStub)).
			List(context.Background(), &proto.ListTasksRequest{})

		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrService), ShouldBeTrue)
	})

	Convey("依赖未注入时返回服务错误", t, func() {
		svc := NewTaskService(nil, nil)
		So(svc.InitAndCheckParam(context.Background(), &proto.ListTasksRequest{}), ShouldBeNil)

		response, err := svc.List(context.Background(), &proto.ListTasksRequest{})

		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrService), ShouldBeTrue)
	})

	Convey("查询成功时传递筛选条件并组装响应", t, func() {
		now := time.Date(2026, 9, 3, 18, 0, 0, 0, time.Local)
		description := "完成接口联调"
		repository := &taskRepositoryStub{
			items: []dao.TaskWithCreator{{
				Task: model.Task{
					ID:          9,
					Title:       "买水果",
					Description: &description,
					Status:      model.TaskStatusTodo,
					Priority:    model.TaskPriorityHigh,
					Category:    "生活",
					DueAt:       &now,
					CreatedBy:   3,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
				CreatorNickname: "晚风收藏家",
				CreatorEmoji:    "🌿",
				CreatorColor:    "#759B89",
			}},
			total: 3,
			stats: dao.TaskStats{OverallTotal: 7, TodoCount: 5, DoneCount: 2},
		}
		svc := NewTaskService(new(gorm.DB), repository)
		req := &proto.ListTasksRequest{
			Status:    model.TaskStatusTodo,
			Priority:  model.TaskPriorityHigh,
			Category:  "生活",
			Keyword:   "水果",
			CreatedBy: 3,
			Page:      1,
			PageSize:  2,
		}
		So(svc.InitAndCheckParam(context.Background(), req), ShouldBeNil)

		response, err := svc.List(context.Background(), req)

		So(err, ShouldBeNil)
		So(repository.listInvoked, ShouldBeTrue)
		So(repository.query.Keyword, ShouldEqual, "水果")
		So(repository.statsCreatedBy, ShouldEqual, uint64(3))
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].Id, ShouldEqual, "9")
		So(response.Items[0].CreatedBy, ShouldEqual, "3")
		So(response.Items[0].Creator.Nickname, ShouldEqual, "晚风收藏家")
		So(response.Items[0].Creator.AvatarEmoji, ShouldEqual, "🌿")
		So(response.Items[0].Description, ShouldEqual, description)
		So(response.Items[0].DueAt, ShouldNotBeEmpty)
		So(response.Total, ShouldEqual, int64(3))
		So(response.HasMore, ShouldBeTrue)
		So(response.OverallTotal, ShouldEqual, int64(7))
		So(response.TodoCount, ShouldEqual, int64(5))
		So(response.DoneCount, ShouldEqual, int64(2))
	})

	Convey("DAO 错误转换为数据库查询错误", t, func() {
		repository := &taskRepositoryStub{err: errors.New("query failed")}
		svc := NewTaskService(new(gorm.DB), repository)
		req := &proto.ListTasksRequest{}
		So(svc.InitAndCheckParam(context.Background(), req), ShouldBeNil)

		response, err := svc.List(context.Background(), req)

		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrDBQuery), ShouldBeTrue)
	})
}

func TestTaskToProto(t *testing.T) {
	Convey("nil 模型返回 nil", t, func() {
		So(taskToProto(nil), ShouldBeNil)
	})

	Convey("零值时间返回空字符串", t, func() {
		item := taskToProto(&dao.TaskWithCreator{})
		So(item.CreatedAt, ShouldBeEmpty)
		So(item.UpdatedAt, ShouldBeEmpty)
	})
}

func TestTaskServiceCRUD(t *testing.T) {
	Convey("创建任务时清洗字段并设置默认值", t, func() {
		repository := new(taskRepositoryStub)
		svc := NewTaskService(new(gorm.DB), repository)

		item, err := svc.Create(context.Background(), &proto.CreateTaskRequest{
			Title:       " 买水果 ",
			Description: " 新鲜水果 ",
			DueAt:       "2026-09-03T18:00:00+08:00",
		})

		So(err, ShouldBeNil)
		So(item.Id, ShouldEqual, "9")
		So(repository.created.Title, ShouldEqual, "买水果")
		So(repository.created.Priority, ShouldEqual, model.TaskPriorityMedium)
		So(repository.created.Category, ShouldEqual, "其他")
		So(repository.created.Status, ShouldEqual, model.TaskStatusTodo)
		So(repository.created.CreatedBy, ShouldEqual, defaultTaskCreator)
		So(item.Creator.Nickname, ShouldEqual, "晚风收藏家")
	})

	Convey("详情不存在时返回资源不存在", t, func() {
		svc := NewTaskService(new(gorm.DB), new(taskRepositoryStub))

		item, err := svc.Get(context.Background(), &proto.GetTaskRequest{Id: "7"})

		So(item, ShouldBeNil)
		So(errors.Is(err, hecode.ErrNotFound), ShouldBeTrue)
	})

	Convey("编辑任务保留状态并更新可编辑字段", t, func() {
		repository := &taskRepositoryStub{item: &dao.TaskWithCreator{Task: model.Task{ID: 7, Status: model.TaskStatusDone, CreatedBy: 3}}}
		svc := NewTaskService(new(gorm.DB), repository)

		item, err := svc.Update(adminContext(), &proto.UpdateTaskRequest{
			Id:       "7",
			Title:    "整理房间",
			Priority: model.TaskPriorityHigh,
			Category: "生活",
		})

		So(err, ShouldBeNil)
		So(item.Status, ShouldEqual, model.TaskStatusDone)
		So(repository.updated.Title, ShouldEqual, "整理房间")
	})

	Convey("状态接口只接受 todo 或 done", t, func() {
		repository := &taskRepositoryStub{item: &dao.TaskWithCreator{Task: model.Task{ID: 7, Status: model.TaskStatusTodo}}}
		svc := NewTaskService(new(gorm.DB), repository)

		item, err := svc.UpdateStatus(context.Background(), &proto.UpdateTaskStatusRequest{Id: "7", Status: "doing"})
		So(item, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)

		item, err = svc.UpdateStatus(adminContext(), &proto.UpdateTaskStatusRequest{Id: "7", Status: model.TaskStatusDone})
		So(err, ShouldBeNil)
		So(item.Status, ShouldEqual, model.TaskStatusDone)
		So(repository.status, ShouldEqual, model.TaskStatusDone)
	})

	Convey("删除任务调用软删除", t, func() {
		repository := &taskRepositoryStub{item: &dao.TaskWithCreator{Task: model.Task{ID: 7}}}
		svc := NewTaskService(new(gorm.DB), repository)

		response, err := svc.Delete(adminContext(), &proto.DeleteTaskRequest{Id: "7"})

		So(err, ShouldBeNil)
		So(response.Deleted, ShouldBeTrue)
		So(repository.deletedID, ShouldEqual, uint64(7))
	})
}

func TestBuildTaskValidation(t *testing.T) {
	Convey("非法任务字段返回参数错误", t, func() {
		inputs := [][5]string{
			{"", "", "medium", "其他", ""},
			{"标题", "", "urgent", "其他", ""},
			{"标题", "", "medium", "其他", "明天"},
		}
		for _, input := range inputs {
			_, err := buildTask(input[0], input[1], input[2], input[3], input[4])
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})
}
