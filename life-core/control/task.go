package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

// ListTasks 返回任务分页列表。
func (c *Control) ListTasks(ctx context.Context, req *proto.ListTasksRequest) (*proto.ListTasksResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListTasks").
		With(hlog.String("status", req.GetStatus())).
		With(hlog.String("priority", req.GetPriority())).
		With(hlog.String("category", req.GetCategory())).
		With(hlog.String("keyword", req.GetKeyword())).
		With(hlog.Uint64("created_by", req.GetCreatedBy())).
		With(hlog.String("due_from", req.GetDueFrom())).
		With(hlog.String("due_to", req.GetDueTo())).
		With(hlog.Int32("page", req.GetPage())).
		With(hlog.Int32("page_size", req.GetPageSize()))

	svc := newTaskService()
	if err := svc.InitAndCheckParam(ctx, req); err != nil {
		logger.Error("InitAndCheckParam failed", hlog.Err(err))
		return nil, err
	}
	resp, err := svc.List(ctx, req)
	if err != nil {
		logger.Error("List failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

// GetTask 返回任务详情。
func (c *Control) GetTask(ctx context.Context, req *proto.GetTaskRequest) (*proto.TaskItem, error) {
	logger := hlog.FromContext(ctx).Named("GetTask").With(hlog.String("id", req.GetId()))
	resp, err := newTaskService().Get(ctx, req)
	if err != nil {
		logger.Error("Get failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

// CreateTask 创建任务。
func (c *Control) CreateTask(ctx context.Context, req *proto.CreateTaskRequest) (*proto.TaskItem, error) {
	logger := hlog.FromContext(ctx).Named("CreateTask").With(hlog.String("title", req.GetTitle()))
	resp, err := newTaskService().Create(ctx, req)
	if err != nil {
		logger.Error("Create failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

// UpdateTask 编辑任务。
func (c *Control) UpdateTask(ctx context.Context, req *proto.UpdateTaskRequest) (*proto.TaskItem, error) {
	logger := hlog.FromContext(ctx).Named("UpdateTask").With(hlog.String("id", req.GetId()))
	resp, err := newTaskService().Update(ctx, req)
	if err != nil {
		logger.Error("Update failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

// UpdateTaskStatus 更新任务完成状态。
func (c *Control) UpdateTaskStatus(ctx context.Context, req *proto.UpdateTaskStatusRequest) (*proto.TaskItem, error) {
	logger := hlog.FromContext(ctx).Named("UpdateTaskStatus").
		With(hlog.String("id", req.GetId())).
		With(hlog.String("status", req.GetStatus()))
	resp, err := newTaskService().UpdateStatus(ctx, req)
	if err != nil {
		logger.Error("UpdateStatus failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

// DeleteTask 软删除任务。
func (c *Control) DeleteTask(ctx context.Context, req *proto.DeleteTaskRequest) (*proto.DeleteTaskResponse, error) {
	logger := hlog.FromContext(ctx).Named("DeleteTask").With(hlog.String("id", req.GetId()))
	resp, err := newTaskService().Delete(ctx, req)
	if err != nil {
		logger.Error("Delete failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func newTaskService() *service.TaskService {
	return service.NewTaskService(database.MySQL(), dao.NewTaskDao())
}
