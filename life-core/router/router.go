package router

import (
	"github.com/vaynedu/hollow"

	"life-core/control"
	"life-core/proto"
)

// Register 注册项目的 HTTP 路由。
func Register(app *hollow.App) {
	app.Engine.Use(configuredAuthMiddleware())
	app.Engine.Use(configuredAIRateLimitMiddleware())
	app.Engine.Use(configuredMediaUploadRateLimitMiddleware())
	app.Engine.POST("/v1/life-core/media/upload", control.UploadMedia)
	app.Engine.GET("/v1/life-core/media/authorize", control.AuthorizeMedia)
	proto.RegisterLifeCoreServiceGinRouter(
		app.Engine,
		control.Get(),
	)
}
