package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRequestJSON(t *testing.T) {
	Convey("请求体支持 lowerCamelCase 字段", t, func() {
		var task CreateTaskRequest
		So(json.Unmarshal([]byte(`{"title":"测试","dueAt":"2026-09-04T18:00:00+08:00"}`), &task), ShouldBeNil)
		So(task.DueAt, ShouldEqual, "2026-09-04T18:00:00+08:00")

		var list ListTasksRequest
		So(json.Unmarshal([]byte(`{"page":2,"pageSize":10}`), &list), ShouldBeNil)
		So(list.Page, ShouldEqual, 2)
		So(list.PageSize, ShouldEqual, 10)

		var consumption CreateRestaurantConsumptionRequest
		So(json.Unmarshal([]byte(`{"restaurantId":"8","consumedAt":"2026-09-02"}`), &consumption), ShouldBeNil)
		So(consumption.RestaurantId, ShouldEqual, "8")
		So(consumption.ConsumedAt, ShouldEqual, "2026-09-02")

		var review ModifyAccessRequestRequest
		So(json.Unmarshal([]byte(`{"userId":"8","status":"approved"}`), &review), ShouldBeNil)
		So(review.UserId, ShouldEqual, "8")
		So(review.Status, ShouldEqual, "approved")

		var voice RecognizeTaskVoiceRequest
		So(json.Unmarshal([]byte(`{"audioBase64":"YWJj","voiceFormat":"mp3"}`), &voice), ShouldBeNil)
		So(voice.AudioBase64, ShouldEqual, "YWJj")
		So(voice.VoiceFormat, ShouldEqual, "mp3")
	})

	Convey("请求体继续兼容 snake_case 字段", t, func() {
		var place CreatePlaceRequest
		So(json.Unmarshal([]byte(`{"visit_count":2,"last_visited_at":"2026-08-17"}`), &place), ShouldBeNil)
		So(place.VisitCount, ShouldEqual, 2)
		So(place.LastVisitedAt, ShouldEqual, "2026-08-17")
	})
}
