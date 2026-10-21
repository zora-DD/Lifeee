package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestContentResponseMarshalJSON(t *testing.T) {
	Convey("贡献和收藏空列表固定返回空数组", t, func() {
		contributions, err := json.Marshal(&ListContributionsResponse{Page: 1, PageSize: 20})
		So(err, ShouldBeNil)
		So(string(contributions), ShouldEqual, `{"items":[],"page":1,"pageSize":20,"total":0,"has_more":false}`)

		favorites, err := json.Marshal(&ListFavoritesResponse{Page: 1, PageSize: 20})
		So(err, ShouldBeNil)
		So(string(favorites), ShouldEqual, `{"items":[],"page":1,"pageSize":20,"total":0,"has_more":false}`)
	})

	Convey("统一内容摘要使用小程序字段命名", t, func() {
		payload, err := json.Marshal(&ContentSummary{TargetId: "9", TargetType: "task", DueAt: "2026-09-03"})
		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"targetId":"9"`)
		So(string(payload), ShouldContainSubstring, `"targetType":"task"`)
		So(string(payload), ShouldContainSubstring, `"dueAt":"2026-09-03"`)
	})
}
