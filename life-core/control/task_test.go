package control

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"

	"life-core/proto"
)

func TestListTasksInvalidParam(t *testing.T) {
	controller := new(Control)
	ctx := context.Background()

	Convey("nil 请求返回参数错误", t, func() {
		response, err := controller.ListTasks(ctx, nil)
		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("非法优先级返回参数错误", t, func() {
		response, err := controller.ListTasks(ctx, &proto.ListTasksRequest{Priority: "urgent"})
		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})
}

func TestTaskMutationInvalidParam(t *testing.T) {
	controller := new(Control)
	ctx := context.Background()

	Convey("创建任务缺少标题时返回参数错误", t, func() {
		response, err := controller.CreateTask(ctx, &proto.CreateTaskRequest{})
		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("详情任务 ID 非法时返回参数错误", t, func() {
		response, err := controller.GetTask(ctx, &proto.GetTaskRequest{Id: "abc"})
		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("任务状态非法时返回参数错误", t, func() {
		response, err := controller.UpdateTaskStatus(ctx, &proto.UpdateTaskStatusRequest{Id: "1", Status: "doing"})
		So(response, ShouldBeNil)
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})
}
