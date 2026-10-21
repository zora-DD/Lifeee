package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vaynedu/hollow/pkg/hecode"

	"life-core/proto"
)

const maxQuickRecordTextLength = 2000

type QuickRecordOrganizer interface {
	OrganizeQuickRecord(context.Context, string, time.Time) (*QuickRecordDraft, error)
}

// QuickRecordService 复用同一套 AI 整理逻辑，处理文字或语音快速记录。
type QuickRecordService struct {
	recognizer    SpeechRecognizer
	organizer     QuickRecordOrganizer
	maxAudioBytes int64
	now           func() time.Time
}

func NewQuickRecordService(recognizer SpeechRecognizer, organizer QuickRecordOrganizer, maxAudioBytes int64) *QuickRecordService {
	return &QuickRecordService{recognizer: recognizer, organizer: organizer, maxAudioBytes: maxAudioBytes, now: time.Now}
}

func (s *QuickRecordService) Parse(ctx context.Context, req *proto.ParseQuickRecordRequest) (*proto.QuickRecordDraftResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if s == nil || s.organizer == nil || s.now == nil {
		return nil, fmt.Errorf("%w: quick record service dependencies are not initialized", hecode.ErrService)
	}
	text := strings.TrimSpace(req.GetText())
	if text == "" || utf8.RuneCountInString(text) > maxQuickRecordTextLength {
		return nil, fmt.Errorf("%w: text is required and must not exceed %d characters", hecode.ErrInvalidParam, maxQuickRecordTextLength)
	}
	return s.organize(ctx, text, 0, "")
}

func (s *QuickRecordService) Recognize(ctx context.Context, req *proto.RecognizeQuickRecordVoiceRequest) (*proto.QuickRecordDraftResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if s == nil || s.recognizer == nil || s.organizer == nil || s.maxAudioBytes <= 0 || s.now == nil {
		return nil, fmt.Errorf("%w: quick record service dependencies are not initialized", hecode.ErrService)
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
	return s.organize(ctx, recognition.Text, recognition.AudioDurationMS, recognition.RequestID)
}

func (s *QuickRecordService) organize(ctx context.Context, rawText string, durationMS int64, requestID string) (*proto.QuickRecordDraftResponse, error) {
	draft, err := s.organizer.OrganizeQuickRecord(ctx, rawText, s.now())
	if err != nil {
		return nil, fmt.Errorf("%w: organize quick record: %v", hecode.ErrService, err)
	}
	if draft == nil {
		return nil, fmt.Errorf("%w: organizer returned an empty draft", hecode.ErrService)
	}
	response := &proto.QuickRecordDraftResponse{
		Intent: draft.Intent, RawText: rawText, AudioDurationMs: durationMS, AsrRequestId: requestID,
		Bills: make([]*proto.BillDraft, 0, len(draft.Bills)),
	}
	if draft.Activity != nil {
		response.Activity = &proto.ActivityLogDraft{
			Content: draft.Activity.Content, OccurredAt: draft.Activity.OccurredAt, DurationMinutes: draft.Activity.DurationMinutes,
		}
	}
	for _, item := range draft.Bills {
		response.Bills = append(response.Bills, &proto.BillDraft{
			Type: item.Type, Title: item.Title, Amount: item.Amount, Category: item.Category,
			OccurredAt: item.OccurredAt, Note: item.Note,
		})
	}
	return response, nil
}
