package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTableNames(t *testing.T) {
	Convey("所有模型使用明确的数据库表名", t, func() {
		So((&User{}).TableName(), ShouldEqual, "users")
		So((&Task{}).TableName(), ShouldEqual, "tasks")
		So((&ActivityLog{}).TableName(), ShouldEqual, "activity_logs")
		So((&Bill{}).TableName(), ShouldEqual, "bills")
		So((&Restaurant{}).TableName(), ShouldEqual, "restaurants")
		So((&RestaurantConsumption{}).TableName(), ShouldEqual, "restaurant_consumptions")
		So((&Place{}).TableName(), ShouldEqual, "places")
		So((&Favorite{}).TableName(), ShouldEqual, "favorites")
		So((&MediaAsset{}).TableName(), ShouldEqual, "media_assets")
		So((&Memory{}).TableName(), ShouldEqual, "memories")
		So((&MemoryPhoto{}).TableName(), ShouldEqual, "memory_photos")
		So((&TaskAISummary{}).TableName(), ShouldEqual, "task_ai_summaries")
		So((&UserModulePermission{}).TableName(), ShouldEqual, "user_module_permissions")
	})
}
