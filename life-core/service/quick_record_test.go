package service

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"life-core/proto"
)

type quickRecordOrganizerStub struct {
	rawText string
	draft   *QuickRecordDraft
}

func (s *quickRecordOrganizerStub) OrganizeQuickRecord(_ context.Context, rawText string, _ time.Time) (*QuickRecordDraft, error) {
	s.rawText = rawText
	return s.draft, nil
}

func TestQuickRecordService(t *testing.T) {
	Convey("文字输入可以拆分为多笔账单草稿", t, func() {
		organizer := &quickRecordOrganizerStub{draft: &QuickRecordDraft{Intent: "bill", Bills: []BillDraft{
			{Type: "expense", Title: "午饭", Amount: "28.00", Category: "餐饮", OccurredAt: "2026-09-04T12:00:00+08:00"},
			{Type: "expense", Title: "打车", Amount: "16.00", Category: "交通", OccurredAt: "2026-09-04T13:00:00+08:00"},
		}}}
		response, err := NewQuickRecordService(nil, organizer, 1024).Parse(context.Background(), &proto.ParseQuickRecordRequest{Text: "昨天午饭28，打车16"})

		So(err, ShouldBeNil)
		So(response.Intent, ShouldEqual, "bill")
		So(response.Bills, ShouldHaveLength, 2)
		So(organizer.rawText, ShouldEqual, "昨天午饭28，打车16")
	})

	Convey("语音输入复用识别结果并返回生活记录草稿", t, func() {
		recognizer := &speechRecognizerStub{result: &Recognition{Text: "收拾客厅半小时", AudioDurationMS: 2200, RequestID: "asr-2"}}
		organizer := &quickRecordOrganizerStub{draft: &QuickRecordDraft{Intent: "activity_log", Activity: &ActivityDraft{Content: "收拾客厅", OccurredAt: "2026-09-05T10:00:00+08:00", DurationMinutes: 30}}}
		svc := NewQuickRecordService(recognizer, organizer, 1024)

		response, err := svc.Recognize(context.Background(), &proto.RecognizeQuickRecordVoiceRequest{AudioBase64: base64.StdEncoding.EncodeToString([]byte("audio")), VoiceFormat: "mp3"})

		So(err, ShouldBeNil)
		So(response.Intent, ShouldEqual, "activity_log")
		So(response.Activity.DurationMinutes, ShouldEqual, int32(30))
		So(response.AsrRequestId, ShouldEqual, "asr-2")
	})
}
