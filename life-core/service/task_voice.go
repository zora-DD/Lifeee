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

// TaskTextOrganizer 定义任务文本整理能力。
type TaskTextOrganizer interface {
	OrganizeTask(context.Context, string, time.Time) (*TaskDraft, error)
}

// TaskVoiceService 编排语音识别和任务草稿整理。
type TaskVoiceService struct {
	recognizer    SpeechRecognizer
	organizer     TaskTextOrganizer
	maxAudioBytes int64
	now           func() time.Time
}

// NewTaskVoiceService 创建语音任务服务。
func NewTaskVoiceService(recognizer SpeechRecognizer, organizer TaskTextOrganizer, maxAudioBytes int64) *TaskVoiceService {
	return &TaskVoiceService{
		recognizer: recognizer, organizer: organizer, maxAudioBytes: maxAudioBytes, now: time.Now,
	}
}

// Recognize 识别录音并返回待用户确认的任务草稿，不直接创建任务。
func (s *TaskVoiceService) Recognize(ctx context.Context, req *proto.RecognizeTaskVoiceRequest) (*proto.RecognizeTaskVoiceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if s == nil || s.recognizer == nil || s.organizer == nil || s.maxAudioBytes <= 0 || s.now == nil {
		return nil, fmt.Errorf("%w: task voice service dependencies are not initialized", hecode.ErrService)
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
	draft, err := s.organizer.OrganizeTask(ctx, recognition.Text, s.now())
	if err != nil {
		return nil, fmt.Errorf("%w: organize task text: %v", hecode.ErrService, err)
	}
	if draft == nil {
		return nil, fmt.Errorf("%w: task organizer returned an empty draft", hecode.ErrService)
	}
	return &proto.RecognizeTaskVoiceResponse{
		RawText:         recognition.Text,
		Title:           draft.Title,
		Description:     draft.Description,
		Category:        draft.Category,
		AudioDurationMs: recognition.AudioDurationMS,
		AsrRequestId:    recognition.RequestID,
	}, nil
}
