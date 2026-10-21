package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/vaynedu/hollow/pkg/hecode"

	"life-core/proto"
)

const maxTencentASRBase64Bytes = 3 * 1024 * 1024

var supportedVoiceFormats = map[string]struct{}{
	"wav": {}, "pcm": {}, "ogg-opus": {}, "speex": {}, "silk": {},
	"mp3": {}, "m4a": {}, "aac": {}, "amr": {},
}

// SpeechRecognizer 定义语音转文字能力。
type SpeechRecognizer interface {
	Recognize(context.Context, []byte, string) (*Recognition, error)
}

// ActivityTextOrganizer 定义生活记录文本整理能力。
type ActivityTextOrganizer interface {
	Organize(context.Context, string, time.Time) (*ActivityDraft, error)
}

// ActivityVoiceService 编排语音识别和生活记录草稿整理。
type ActivityVoiceService struct {
	recognizer    SpeechRecognizer
	organizer     ActivityTextOrganizer
	maxAudioBytes int64
	now           func() time.Time
}

// NewActivityVoiceService 创建语音生活记录服务。
func NewActivityVoiceService(recognizer SpeechRecognizer, organizer ActivityTextOrganizer, maxAudioBytes int64) *ActivityVoiceService {
	return &ActivityVoiceService{
		recognizer: recognizer, organizer: organizer, maxAudioBytes: maxAudioBytes, now: time.Now,
	}
}

// Recognize 识别录音并返回待用户确认的草稿，不直接写入数据库。
func (s *ActivityVoiceService) Recognize(ctx context.Context, req *proto.RecognizeActivityVoiceRequest) (*proto.RecognizeActivityVoiceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if s == nil || s.recognizer == nil || s.organizer == nil || s.maxAudioBytes <= 0 || s.now == nil {
		return nil, fmt.Errorf("%w: activity voice service dependencies are not initialized", hecode.ErrService)
	}
	format := strings.ToLower(strings.TrimSpace(req.GetVoiceFormat()))
	if _, ok := supportedVoiceFormats[format]; !ok {
		return nil, fmt.Errorf("%w: unsupported voiceFormat", hecode.ErrInvalidParam)
	}
	encoded := normalizeAudioBase64(req.GetAudioBase64())
	if encoded == "" {
		return nil, fmt.Errorf("%w: audioBase64 is required", hecode.ErrInvalidParam)
	}
	if len(encoded) > maxTencentASRBase64Bytes {
		return nil, fmt.Errorf("%w: encoded audio exceeds Tencent ASR 3MB limit", hecode.ErrInvalidParam)
	}
	audio, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: audioBase64 is invalid", hecode.ErrInvalidParam)
	}
	if len(audio) == 0 || int64(len(audio)) > s.maxAudioBytes {
		return nil, fmt.Errorf("%w: audio size must be between 1 and %d bytes", hecode.ErrInvalidParam, s.maxAudioBytes)
	}

	recognition, err := s.recognizer.Recognize(ctx, audio, format)
	if err != nil {
		return nil, fmt.Errorf("%w: recognize voice: %v", hecode.ErrService, err)
	}
	if recognition == nil || strings.TrimSpace(recognition.Text) == "" {
		return nil, fmt.Errorf("%w: voice recognizer returned empty text", hecode.ErrService)
	}
	draft, err := s.organizer.Organize(ctx, recognition.Text, s.now())
	if err != nil {
		return nil, fmt.Errorf("%w: organize activity text: %v", hecode.ErrService, err)
	}
	if draft == nil {
		return nil, fmt.Errorf("%w: activity organizer returned an empty draft", hecode.ErrService)
	}
	return &proto.RecognizeActivityVoiceResponse{
		RawText:         recognition.Text,
		Content:         draft.Content,
		OccurredAt:      draft.OccurredAt,
		DurationMinutes: draft.DurationMinutes,
		AudioDurationMs: recognition.AudioDurationMS,
		AsrRequestId:    recognition.RequestID,
	}, nil
}

func normalizeAudioBase64(value string) string {
	value = strings.TrimSpace(value)
	if comma := strings.IndexByte(value, ','); strings.HasPrefix(value, "data:audio/") && comma >= 0 {
		return strings.TrimSpace(value[comma+1:])
	}
	return value
}
