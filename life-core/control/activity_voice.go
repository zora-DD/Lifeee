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

// RecognizeActivityVoice 识别语音并返回待确认的生活记录草稿。
func (c *Control) RecognizeActivityVoice(ctx context.Context, req *proto.RecognizeActivityVoiceRequest) (*proto.RecognizeActivityVoiceResponse, error) {
	logger := hlog.FromContext(ctx).Named("RecognizeActivityVoice").With(hlog.String("voice_format", req.GetVoiceFormat()))
	cfg := config.Get()
	if cfg == nil || !cfg.Voice.Enabled {
		return nil, fmt.Errorf("%w: voice recognition is disabled", hecode.ErrConfig)
	}
	recognizer, err := tencentasr.NewTencentASRClient(cfg.Voice.TencentASR)
	if err != nil {
		logger.Error("initialize Tencent ASR failed", hlog.Err(err))
		return nil, fmt.Errorf("%w: initialize Tencent ASR: %v", hecode.ErrConfig, err)
	}
	organizer, err := minimax.NewMiniMaxClient(cfg.Voice.MiniMax)
	if err != nil {
		logger.Error("initialize MiniMax failed", hlog.Err(err))
		return nil, fmt.Errorf("%w: initialize MiniMax: %v", hecode.ErrConfig, err)
	}
	resp, err := service.NewActivityVoiceService(recognizer, organizer, cfg.Voice.MaxAudioBytes).Recognize(ctx, req)
	if err != nil {
		logger.Error("recognize activity voice failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}
