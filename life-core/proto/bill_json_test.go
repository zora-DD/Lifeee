package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBillJSONContract(t *testing.T) {
	Convey("账单列表保持前端 camelCase 契约并输出空数组", t, func() {
		payload, err := json.Marshal(&ListBillsResponse{Page: 1, PageSize: 20})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"items":[]`)
		So(string(payload), ShouldContainSubstring, `"creators":[]`)
		So(string(payload), ShouldContainSubstring, `"pageSize":20`)
		So(string(payload), ShouldContainSubstring, `"has_more":false`)
	})

	Convey("快速记录响应支持多笔账单", t, func() {
		payload, err := json.Marshal(&QuickRecordDraftResponse{
			Intent:          "bill",
			Bills:           []*BillDraft{{Title: "午饭", Amount: "28.00", OccurredAt: "2026-09-05T12:00:00+08:00"}},
			AudioDurationMs: 1200,
		})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"occurredAt":"2026-09-05T12:00:00+08:00"`)
		So(string(payload), ShouldContainSubstring, `"audioDurationMs":1200`)
	})

	Convey("账单请求兼容 lowerCamelCase", t, func() {
		var request CreateBillRequest
		err := json.Unmarshal([]byte(`{"title":"午饭","amount":"28","occurredAt":"2026-09-05T12:00:00+08:00","inputType":"text","rawText":"午饭28"}`), &request)

		So(err, ShouldBeNil)
		So(request.OccurredAt, ShouldEqual, "2026-09-05T12:00:00+08:00")
		So(request.InputType, ShouldEqual, "text")
		So(request.RawText, ShouldEqual, "午饭28")
	})
}
