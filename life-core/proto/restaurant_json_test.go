package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRestaurantJSON(t *testing.T) {
	Convey("吃喝列表保留空数组、分页和 has_more", t, func() {
		payload, err := json.Marshal(&ListRestaurantsResponse{Page: 1, PageSize: 20})

		So(err, ShouldBeNil)
		So(string(payload), ShouldEqual, `{"items":[],"page":1,"pageSize":20,"total":0,"has_more":false,"overallTotal":0,"totalVisitCount":0,"totalSpent":""}`)
	})

	Convey("吃喝详情使用前端字段名并保留消费记录数组", t, func() {
		payload, err := json.Marshal(&RestaurantItem{Id: "8", BusinessHours: "09:00-20:00", Features: "鱼鲜 · 菜味很好"})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"id":"8"`)
		So(string(payload), ShouldContainSubstring, `"businessHours":"09:00-20:00"`)
		So(string(payload), ShouldContainSubstring, `"features":"鱼鲜 · 菜味很好"`)
		So(string(payload), ShouldContainSubstring, `"consumptions":[]`)
	})
}
