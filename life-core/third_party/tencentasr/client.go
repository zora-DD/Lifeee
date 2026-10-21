package tencentasr

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	asr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/asr/v20190614"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	"life-core/config"
	"life-core/service"
)

// TencentASRClient 调用腾讯云一句话识别接口。
type TencentASRClient struct {
	client          *asr.Client
	engineModelType string
}

// NewTencentASRClient 创建腾讯云语音识别客户端。
func NewTencentASRClient(cfg config.TencentASRConfig) (*TencentASRClient, error) {
	if strings.TrimSpace(cfg.SecretID) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, errors.New("voice: tencent asr credential is required")
	}
	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)
	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.Endpoint = "asr.tencentcloudapi.com"
	client, err := asr.NewClient(credential, cfg.Region, clientProfile)
	if err != nil {
		return nil, fmt.Errorf("voice: create tencent asr client: %w", err)
	}
	return &TencentASRClient{client: client, engineModelType: cfg.EngineModelType}, nil
}

// Recognize 识别不超过 60 秒的短音频。
func (c *TencentASRClient) Recognize(ctx context.Context, audio []byte, format string) (*service.Recognition, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("voice: tencent asr client is not initialized")
	}
	request := asr.NewSentenceRecognitionRequest()
	request.EngSerViceType = common.StringPtr(c.engineModelType)
	request.SourceType = common.Uint64Ptr(1)
	request.VoiceFormat = common.StringPtr(format)
	request.Data = common.StringPtr(base64.StdEncoding.EncodeToString(audio))
	request.DataLen = common.Int64Ptr(int64(len(audio)))
	request.ConvertNumMode = common.Int64Ptr(1)
	request.FilterModal = common.Int64Ptr(1)

	response, err := c.client.SentenceRecognitionWithContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("voice: tencent asr request failed: %w", err)
	}
	if response == nil || response.Response == nil || response.Response.Result == nil {
		return nil, errors.New("voice: tencent asr returned an empty response")
	}
	result := &service.Recognition{Text: strings.TrimSpace(*response.Response.Result)}
	if response.Response.AudioDuration != nil {
		result.AudioDurationMS = *response.Response.AudioDuration
	}
	if response.Response.RequestId != nil {
		result.RequestID = *response.Response.RequestId
	}
	if result.Text == "" {
		return nil, errors.New("voice: tencent asr returned empty text")
	}
	return result, nil
}
