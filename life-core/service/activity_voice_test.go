package service

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"

	"life-core/proto"
)

type speechRecognizerStub struct {
	audio  []byte
	format string
	result *Recognition
	err    error
}

func (s *speechRecognizerStub) Recognize(_ context.Context, audio []byte, format string) (*Recognition, error) {
	s.audio = append([]byte(nil), audio...)
	s.format = format
	return s.result, s.err
}

type activityOrganizerStub struct {
	rawText string
	now     time.Time
	draft   *ActivityDraft
	err     error
}

func (s *activityOrganizerStub) Organize(_ context.Context, rawText string, now time.Time) (*ActivityDraft, error) {
	s.rawText = rawText
	s.now = now
	return s.draft, s.err
}

func TestActivityVoiceServiceRecognize(t *testing.T) {
	Convey("语音识别后返回待确认草稿", t, func() {
		now := time.Date(2026, 9, 3, 9, 30, 0, 0, time.FixedZone("CST", 8*60*60))
		recognizer := &speechRecognizerStub{result: &Recognition{
			Text: "早上收拾了客厅，花了半小时。", AudioDurationMS: 5200, RequestID: "asr-request-1",
		}}
		organizer := &activityOrganizerStub{draft: &ActivityDraft{
			Content: "收拾客厅", OccurredAt: now.Format(time.RFC3339), DurationMinutes: 30,
		}}
		svc := NewActivityVoiceService(recognizer, organizer, 1024)
		svc.now = func() time.Time { return now }

		resp, err := svc.Recognize(context.Background(), &proto.RecognizeActivityVoiceRequest{
			AudioBase64: base64.StdEncoding.EncodeToString([]byte("fake-mp3")), VoiceFormat: "MP3",
		})

		So(err, ShouldBeNil)
		So(string(recognizer.audio), ShouldEqual, "fake-mp3")
		So(recognizer.format, ShouldEqual, "mp3")
		So(organizer.rawText, ShouldEqual, "早上收拾了客厅，花了半小时。")
		So(organizer.now, ShouldResemble, now)
		So(resp.GetContent(), ShouldEqual, "收拾客厅")
		So(resp.GetDurationMinutes(), ShouldEqual, 30)
		So(resp.GetAudioDurationMs(), ShouldEqual, int64(5200))
		So(resp.GetAsrRequestId(), ShouldEqual, "asr-request-1")
	})

	Convey("拒绝非法音频参数", t, func() {
		svc := NewActivityVoiceService(&speechRecognizerStub{}, &activityOrganizerStub{}, 4)

		_, err := svc.Recognize(context.Background(), &proto.RecognizeActivityVoiceRequest{VoiceFormat: "mp3"})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)

		_, err = svc.Recognize(context.Background(), &proto.RecognizeActivityVoiceRequest{AudioBase64: "%%%", VoiceFormat: "mp3"})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)

		_, err = svc.Recognize(context.Background(), &proto.RecognizeActivityVoiceRequest{
			AudioBase64: base64.StdEncoding.EncodeToString([]byte("12345")), VoiceFormat: "mp3",
		})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)

		_, err = svc.Recognize(context.Background(), &proto.RecognizeActivityVoiceRequest{
			AudioBase64: base64.StdEncoding.EncodeToString([]byte("1")), VoiceFormat: "flac",
		})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})

	Convey("第三方服务失败时返回服务错误", t, func() {
		svc := NewActivityVoiceService(
			&speechRecognizerStub{err: errors.New("asr unavailable")},
			&activityOrganizerStub{},
			1024,
		)
		_, err := svc.Recognize(context.Background(), &proto.RecognizeActivityVoiceRequest{
			AudioBase64: base64.StdEncoding.EncodeToString([]byte("audio")), VoiceFormat: "mp3",
		})
		So(errors.Is(err, hecode.ErrService), ShouldBeTrue)
	})
}
