package service

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"life-core/proto"
)

type taskOrganizerStub struct {
	rawText string
	now     time.Time
	draft   *TaskDraft
	err     error
}

func (s *taskOrganizerStub) OrganizeTask(_ context.Context, rawText string, now time.Time) (*TaskDraft, error) {
	s.rawText = rawText
	s.now = now
	return s.draft, s.err
}

func TestTaskVoiceServiceRecognize(t *testing.T) {
	Convey("语音识别后返回待确认的任务草稿", t, func() {
		now := time.Date(2026, 9, 3, 9, 30, 0, 0, time.FixedZone("CST", 8*60*60))
		recognizer := &speechRecognizerStub{result: &Recognition{
			Text: "记得买猫粮，顺便买两个罐头。", AudioDurationMS: 3600, RequestID: "asr-task-1",
		}}
		organizer := &taskOrganizerStub{draft: &TaskDraft{
			Title: "购买猫粮", Description: "顺便买两个罐头", Category: "生活",
		}}
		svc := NewTaskVoiceService(recognizer, organizer, 1024)
		svc.now = func() time.Time { return now }

		resp, err := svc.Recognize(context.Background(), &proto.RecognizeTaskVoiceRequest{
			AudioBase64: base64.StdEncoding.EncodeToString([]byte("fake-mp3")), VoiceFormat: "MP3",
		})

		So(err, ShouldBeNil)
		So(string(recognizer.audio), ShouldEqual, "fake-mp3")
		So(recognizer.format, ShouldEqual, "mp3")
		So(organizer.rawText, ShouldEqual, "记得买猫粮，顺便买两个罐头。")
		So(organizer.now, ShouldResemble, now)
		So(resp.GetTitle(), ShouldEqual, "购买猫粮")
		So(resp.GetDescription(), ShouldEqual, "顺便买两个罐头")
		So(resp.GetCategory(), ShouldEqual, "生活")
		So(resp.GetAudioDurationMs(), ShouldEqual, int64(3600))
		So(resp.GetAsrRequestId(), ShouldEqual, "asr-task-1")
	})
}
