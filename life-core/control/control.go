package control

import (
	"sync"

	"life-core/proto"
	"life-core/service"
)

// Control 实现项目的 Proto Service，负责接口校验、业务编排和响应组装。
type Control struct {
	proto.UnimplementedLifeCoreServiceService
	aiSummaryOnce sync.Once
	aiSummary     *service.AISummaryService
}

var instance = new(Control)

// Get 返回项目唯一的 Control 实例。
func Get() *Control {
	return instance
}
