package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestActivityLogJSON(t *testing.T) {
	Convey("列表显式返回空数组和 has_more", t, func() {
		payload, err := json.Marshal(&ListActivityLogsResponse{Page: 1, PageSize: 20})

		So(err, ShouldBeNil)
		So(string(payload), ShouldEqual, `{"items":[],"page":1,"pageSize":20,"total":0,"has_more":false}`)
	})

	Convey("生活记录返回创建人展示资料", t, func() {
		payload, err := json.Marshal(&ActivityLogItem{
			Id: "8", CreatedBy: "4",
			Creator: &ActivityLogCreator{Id: "4", Nickname: "云朵漫游者", AvatarEmoji: "☁️", AvatarColor: "#8AA6B8"},
		})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"creator":{"id":"4","nickname":"云朵漫游者","avatarEmoji":"☁️","avatarColor":"#8AA6B8"}`)
	})

	Convey("统计响应使用前端字段并保留空数组", t, func() {
		payload, err := json.Marshal(&ActivityLogStatsResponse{})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"recordCount":0`)
		So(string(payload), ShouldContainSubstring, `"dailyBreakdown":[]`)
		So(string(payload), ShouldContainSubstring, `"monthlyBreakdown":[]`)
	})

	Convey("语音识别请求兼容 snake_case 字段", t, func() {
		var request RecognizeActivityVoiceRequest
		err := json.Unmarshal([]byte(`{"audio_base64":"YQ==","voice_format":"mp3"}`), &request)

		So(err, ShouldBeNil)
		So(request.GetAudioBase64(), ShouldEqual, "YQ==")
		So(request.GetVoiceFormat(), ShouldEqual, "mp3")
	})
}
