package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoad(t *testing.T) {
	Convey("加载项目配置", t, func() {
		dir := t.TempDir()

		Convey("数据库默认关闭", func() {
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte("database: {}\n"), 0o600), ShouldBeNil)

			cfg, err := load(dir, "conf")

			So(err, ShouldBeNil)
			So(cfg.Database.MySQL.Enabled, ShouldBeFalse)
			So(cfg.Database.Redis.Enabled, ShouldBeFalse)
			So(cfg.Auth.TokenTTL, ShouldEqual, defaultAccessTokenTTL)
			So(cfg.Auth.RequiredPaths, ShouldResemble, defaultAuthRequiredPaths())
			So(cfg.Voice.MaxAudioBytes, ShouldEqual, defaultVoiceMaxAudioBytes)
			So(cfg.Voice.RateLimit.PerMinute, ShouldEqual, defaultVoicePerMinute)
			So(cfg.Voice.RateLimit.PerDay, ShouldEqual, defaultVoicePerDay)
			So(cfg.Voice.TencentASR.Region, ShouldEqual, "ap-guangzhou")
			So(cfg.Voice.TencentASR.EngineModelType, ShouldEqual, "16k_zh")
			So(cfg.Voice.MiniMax.Timeout, ShouldEqual, defaultVoiceTimeout)
			So(cfg.Media.StoragePath, ShouldEqual, "data/media")
			So(cfg.Media.MaxUploadBytes, ShouldEqual, defaultMediaMaxUploadBytes)
			So(cfg.Media.MaxImageEdge, ShouldEqual, defaultMediaMaxImageEdge)
			So(cfg.Media.SignedURLTTL, ShouldEqual, defaultMediaSignedURLTTL)
			So(cfg.Scheduler.Enabled, ShouldBeFalse)
			So(cfg.Scheduler.Timezone, ShouldEqual, defaultSchedulerTimezone)
			So(cfg.Scheduler.DailyTaskSummary.Spec, ShouldEqual, defaultTaskSummarySpec)
			So(cfg.Scheduler.DailyTaskSummary.Timeout, ShouldEqual, defaultTaskSummaryTimeout)
		})

		Convey("读取 MySQL 和 Redis 配置", func() {
			content := "database:\n" +
				"  mysql:\n    enabled: true\n    dsn: mysql-dsn\n" +
				"  redis:\n    enabled: true\n    addr: 127.0.0.1:6379\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			cfg, err := load(dir, "conf")

			So(err, ShouldBeNil)
			So(cfg.Database.MySQL.DSN, ShouldEqual, "mysql-dsn")
			So(cfg.Database.Redis.Addr, ShouldEqual, "127.0.0.1:6379")
		})

		Convey("启用 MySQL 时必须配置 DSN", func() {
			content := "database:\n  mysql:\n    enabled: true\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			_, err := load(dir, "conf")

			So(errors.Is(err, ErrMySQLDSNRequired), ShouldBeTrue)
		})

		Convey("启用 Redis 时必须配置地址", func() {
			content := "database:\n  redis:\n    enabled: true\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			_, err := load(dir, "conf")

			So(errors.Is(err, ErrRedisAddrRequired), ShouldBeTrue)
		})

		Convey("启用鉴权时读取 Token 和微信配置", func() {
			content := "auth:\n" +
				"  enabled: true\n  jwt_secret: unit-test-jwt-secret-with-at-least-32-characters\n  token_ttl: 24h\n" +
				"  required_paths:\n    - /v1/life-core/task/*\n    - /v1/life-core/place/detail\n" +
				"wechat:\n  mock_enabled: true\n  mock_open_id: local-open-id\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			cfg, err := load(dir, "conf")

			So(err, ShouldBeNil)
			So(cfg.Auth.Enabled, ShouldBeTrue)
			So(cfg.Auth.TokenTTL.String(), ShouldEqual, "24h0m0s")
			So(cfg.Auth.RequiredPaths, ShouldResemble, []string{"/v1/life-core/task/*", "/v1/life-core/place/detail"})
			So(cfg.WeChat.MockOpenID, ShouldEqual, "local-open-id")
		})

		Convey("真实微信登录必须配置 AppID 和 AppSecret", func() {
			content := "auth:\n" +
				"  enabled: true\n  jwt_secret: unit-test-jwt-secret-with-at-least-32-characters\n" +
				"wechat:\n  mock_enabled: false\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			_, err := load(dir, "conf")
			So(errors.Is(err, ErrWeChatConfigRequired), ShouldBeTrue)
		})

		Convey("启用语音识别时必须配置腾讯云和 MiniMax", func() {
			content := "voice:\n  enabled: true\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			_, err := load(dir, "conf")
			So(errors.Is(err, ErrVoiceConfigRequired), ShouldBeTrue)
		})

		Convey("读取完整语音识别配置", func() {
			content := "voice:\n" +
				"  enabled: true\n  max_audio_bytes: 1024\n  rate_limit:\n    per_minute: 4\n    per_day: 40\n" +
				"  tencent_asr:\n    secret_id: sid\n    secret_key: skey\n    region: ap-shanghai\n    engine_model_type: 16k_zh\n" +
				"  minimax:\n    endpoint: https://example.com/v1/chat/completions\n    api_key: key\n    model: test-model\n    timeout: 3s\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			cfg, err := load(dir, "conf")
			So(err, ShouldBeNil)
			So(cfg.Voice.MaxAudioBytes, ShouldEqual, int64(1024))
			So(cfg.Voice.RateLimit.PerMinute, ShouldEqual, 4)
			So(cfg.Voice.RateLimit.PerDay, ShouldEqual, 40)
			So(cfg.Voice.TencentASR.Region, ShouldEqual, "ap-shanghai")
			So(cfg.Voice.MiniMax.Model, ShouldEqual, "test-model")
			So(cfg.Voice.MiniMax.Timeout, ShouldEqual, 3*time.Second)
		})

		Convey("通过环境变量注入语音服务密钥", func() {
			t.Setenv("LIFE_CORE_VOICE_ENABLED", "true")
			t.Setenv("TENCENTCLOUD_SECRET_ID", "env-secret-id")
			t.Setenv("TENCENTCLOUD_SECRET_KEY", "env-secret-key")
			t.Setenv("LIFE_CORE_MINIMAX_API_KEY", "env-minimax-key")
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte("database: {}\n"), 0o600), ShouldBeNil)

			cfg, err := load(dir, "conf")
			So(err, ShouldBeNil)
			So(cfg.Voice.Enabled, ShouldBeTrue)
			So(cfg.Voice.TencentASR.SecretID, ShouldEqual, "env-secret-id")
			So(cfg.Voice.MiniMax.APIKey, ShouldEqual, "env-minimax-key")
			So(cfg.Voice.MiniMax.Endpoint, ShouldEqual, "https://api.minimaxi.com/anthropic/v1/messages")
			So(cfg.Voice.MiniMax.Model, ShouldEqual, "MiniMax-M2.7")
		})

		Convey("启用图片服务时读取上传、压缩和签名配置", func() {
			content := "auth:\n  jwt_secret: unit-test-jwt-secret-with-at-least-32-characters\n" +
				"media:\n  enabled: true\n  storage_path: /data/life-core/media\n" +
				"  max_upload_bytes: 2048\n  max_image_edge: 900\n  jpeg_quality: 80\n" +
				"  signed_url_ttl: 30m\n  upload_rate_limit:\n    per_minute: 3\n    per_day: 20\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			cfg, err := load(dir, "conf")

			So(err, ShouldBeNil)
			So(cfg.Media.Enabled, ShouldBeTrue)
			So(cfg.Media.StoragePath, ShouldEqual, "/data/life-core/media")
			So(cfg.MediaSigningSecret(), ShouldEqual, "unit-test-jwt-secret-with-at-least-32-characters")
			So(cfg.Media.SignedURLTTL, ShouldEqual, 30*time.Minute)
			So(cfg.Media.UploadRateLimit.PerMinute, ShouldEqual, 3)
		})

		Convey("启用图片服务时必须存在足够长度的签名密钥", func() {
			content := "media:\n  enabled: true\n  signing_secret: short\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			_, err := load(dir, "conf")
			So(errors.Is(err, ErrMediaConfigRequired), ShouldBeTrue)
		})

		Convey("读取定时任务配置", func() {
			content := "scheduler:\n  enabled: true\n  timezone: Asia/Shanghai\n" +
				"  daily_task_summary:\n    enabled: true\n    spec: '30 6 * * *'\n    timeout: 90s\n    run_on_start: true\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)

			cfg, err := load(dir, "conf")
			So(err, ShouldBeNil)
			So(cfg.Scheduler.Enabled, ShouldBeTrue)
			So(cfg.Scheduler.DailyTaskSummary.Enabled, ShouldBeTrue)
			So(cfg.Scheduler.DailyTaskSummary.Spec, ShouldEqual, "30 6 * * *")
			So(cfg.Scheduler.DailyTaskSummary.Timeout, ShouldEqual, 90*time.Second)
			So(cfg.Scheduler.DailyTaskSummary.RunOnStart, ShouldBeTrue)
		})

		Convey("启用调度器时拒绝非法时区和 cron 表达式", func() {
			content := "scheduler:\n  enabled: true\n  timezone: Invalid/Timezone\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)
			_, err := load(dir, "conf")
			So(errors.Is(err, ErrSchedulerConfigInvalid), ShouldBeTrue)

			content = "scheduler:\n  enabled: true\n  timezone: Asia/Shanghai\n" +
				"  daily_task_summary:\n    enabled: true\n    spec: invalid\n"
			So(os.WriteFile(filepath.Join(dir, "conf.yaml"), []byte(content), 0o600), ShouldBeNil)
			_, err = load(dir, "conf")
			So(errors.Is(err, ErrSchedulerConfigInvalid), ShouldBeTrue)
		})
	})
}
