//go:build integration

package tencentasr

import (
	"context"
	"os"
	"testing"
	"time"

	"life-core/config"
)

// TestTencentASRIntegration 使用真实腾讯云账号验证一句话识别。
// 凭据和音频路径只从环境变量读取，避免进入代码仓库。
func TestTencentASRIntegration(t *testing.T) {
	secretID := os.Getenv("TENCENTCLOUD_SECRET_ID")
	secretKey := os.Getenv("TENCENTCLOUD_SECRET_KEY")
	audioPath := os.Getenv("LIFE_CORE_TEST_AUDIO")
	if secretID == "" || secretKey == "" || audioPath == "" {
		t.Skip("未配置腾讯云集成测试环境变量")
	}
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewTencentASRClient(config.TencentASRConfig{
		SecretID: secretID, SecretKey: secretKey, Region: "ap-guangzhou", EngineModelType: "16k_zh",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := client.Recognize(ctx, audio, "wav")
	if err != nil {
		t.Fatal(err)
	}
	if result.Text == "" {
		t.Fatal("腾讯云返回空识别文本")
	}
	t.Logf("识别结果：%s；音频时长：%dms；请求ID：%s", result.Text, result.AudioDurationMS, result.RequestID)
}
