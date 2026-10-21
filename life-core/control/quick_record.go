package control

import (
	"context"
	"fmt"

	"github.com/vaynedu/hollow/pkg/hecode"
	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/config"
	"life-core/proto"
	"life-core/service"
	"life-core/third_party/minimax"
	"life-core/third_party/tencentasr"
)

func (c *Control) ParseQuickRecord(ctx context.Context, req *proto.ParseQuickRecordRequest) (*proto.QuickRecordDraftResponse, error) {
	logger := hlog.FromContext(ctx).Named("ParseQuickRecord")
	svc, err := newQuickRecordService(false)
	if err != nil {
		logger.Error("initialize quick record service failed", hlog.Err(err))
		return nil, err
	}
	resp, err := svc.Parse(ctx, req)
	if err != nil {
		logger.Error("parse quick record failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) RecognizeQuickRecordVoice(ctx context.Context, req *proto.RecognizeQuickRecordVoiceRequest) (*proto.QuickRecordDraftResponse, error) {
	logger := hlog.FromContext(ctx).Named("RecognizeQuickRecordVoice").With(hlog.String("voice_format", req.GetVoiceFormat()))
	svc, err := newQuickRecordService(true)
	if err != nil {
		logger.Error("initialize quick record service failed", hlog.Err(err))
		return nil, err
	}
	resp, err := svc.Recognize(ctx, req)
	if err != nil {
		logger.Error("recognize quick record voice failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func newQuickRecordService(withRecognizer bool) (*service.QuickRecordService, error) {
	cfg := config.Get()
	if cfg == nil || !cfg.Voice.Enabled {
		return nil, fmt.Errorf("%w: voice feature is disabled", hecode.ErrService)
	}
	organizer, err := minimax.NewMiniMaxClient(cfg.Voice.MiniMax)
	if err != nil {
		return nil, fmt.Errorf("%w: initialize MiniMax: %v", hecode.ErrConfig, err)
	}
	if !withRecognizer {
		return service.NewQuickRecordService(nil, organizer, cfg.Voice.MaxAudioBytes), nil
	}
	recognizer, err := tencentasr.NewTencentASRClient(cfg.Voice.TencentASR)
	if err != nil {
		return nil, fmt.Errorf("%w: initialize Tencent ASR: %v", hecode.ErrConfig, err)
	}
	return service.NewQuickRecordService(recognizer, organizer, cfg.Voice.MaxAudioBytes), nil
}
