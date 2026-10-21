package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPlaceJSON(t *testing.T) {
	Convey("地点列表保留空数组和分页字段", t, func() {
		payload, err := json.Marshal(&ListPlacesResponse{Page: 1, PageSize: 20})

		So(err, ShouldBeNil)
		So(string(payload), ShouldEqual, `{"items":[],"page":1,"pageSize":20,"total":0,"has_more":false,"overallTotal":0,"visitedCount":0,"wishlistCount":0}`)
	})

	Convey("地点条目使用前端字段名", t, func() {
		payload, err := json.Marshal(&PlaceItem{
			Id: "6", VisitCount: 3, LastVisitedAt: "2026-08-17",
			GuideUrl: "https://mp.weixin.qq.com/s?mid=1", DistanceKm: 320, DriveDurationMinutes: 270,
		})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"id":"6"`)
		So(string(payload), ShouldContainSubstring, `"visitCount":3`)
		So(string(payload), ShouldContainSubstring, `"lastVisitedAt":"2026-08-17"`)
		So(string(payload), ShouldContainSubstring, `"guideUrl":"https://mp.weixin.qq.com/s?mid=1"`)
		So(string(payload), ShouldContainSubstring, `"distanceKm":320`)
		So(string(payload), ShouldContainSubstring, `"driveDurationMinutes":270`)
	})
}
