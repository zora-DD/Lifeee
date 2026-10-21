package main

import (
	"log"

	"github.com/vaynedu/hollow"

	"life-core/config"
	"life-core/infrastructure/database"
	"life-core/infrastructure/scheduler"
	"life-core/router"
)

func run() error {
	app, err := hollow.NewApp(hollow.AppOption{
		ConfigPath: ".",
		ConfigName: "conf",
	})
	if err != nil {
		return err
	}

	app.Startup(
		config.Startup,
		database.InitMySQL,
		database.InitRedis,
		scheduler.Startup,
	)
	// Hollow 按注册逆序关闭：先停止调度器，避免数据库关闭后任务仍在运行。
	app.Shutdown(database.Shutdown, scheduler.Shutdown)
	router.Register(app)
	return app.Run()
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
