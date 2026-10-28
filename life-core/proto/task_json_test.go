package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestListTasksResponseMarshalJSON(t *testing.T) {
	Convey("空任务列表保留 items 并显式返回 has_more=false", t, func() {
		payload, err := json.Marshal(&ListTasksResponse{Page: 1, PageSize: 20})

		So(err, ShouldBeNil)
		So(string(payload), ShouldEqual, `{"items":[],"page":1,"pageSize":20,"total":0,"has_more":false,"overallTotal":0,"todoCount":0,"doneCount":0}`)
	})

	Convey("任务条目 ID 按字符串返回", t, func() {
		payload, err := json.Marshal(&ListTasksResponse{
			Items: []*TaskItem{{
				Id: "9007199254740993", CreatedBy: "8",
				Creator: &TaskCreator{Id: "8", Nickname: "晚风收藏家", AvatarEmoji: "🌿", AvatarColor: "#759B89"},
			}},
			Page:  1,
			Total: 1,
		})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"id":"9007199254740993"`)
		So(string(payload), ShouldContainSubstring, `"createdBy":"8"`)
		So(string(payload), ShouldContainSubstring, `"creator":{"id":"8","nickname":"晚风收藏家","avatarEmoji":"🌿","avatarColor":"#759B89"}`)
	})
}

func TestRecognizeTaskVoiceResponseMarshalJSON(t *testing.T) {
	Convey("语音任务草稿使用前端约定的字段名", t, func() {
		payload, err := json.Marshal(&RecognizeTaskVoiceResponse{
			RawText: "买零食", Title: "购买零食", Category: "生活", AudioDurationMs: 1800,
		})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"rawText":"买零食"`)
		So(string(payload), ShouldContainSubstring, `"audioDurationMs":1800`)
	})
}
