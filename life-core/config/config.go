package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vaynedu/hollow/pkg/hconfig"
	"github.com/vaynedu/hollow/pkg/hgorm"
	"github.com/vaynedu/hollow/pkg/hredis"
)

var (
	// ErrMySQLDSNRequired 表示启用 MySQL 后未配置 DSN。
	ErrMySQLDSNRequired = errors.New("config: database.mysql.dsn is required when enabled")
	// ErrRedisAddrRequired 表示启用 Redis 后未配置地址。
	ErrRedisAddrRequired = errors.New("config: database.redis.addr is required when enabled")
	// ErrJWTSecretRequired 表示启用鉴权后未配置足够长度的 JWT 密钥。
	ErrJWTSecretRequired = errors.New("config: auth.jwt_secret must contain at least 32 characters when enabled")
	// ErrWeChatConfigRequired 表示真实微信登录缺少 AppID 或 AppSecret。
	ErrWeChatConfigRequired = errors.New("config: wechat.app_id and wechat.app_secret are required when mock login is disabled")
	// ErrVoiceConfigRequired 表示启用语音识别后缺少第三方服务配置。
	ErrVoiceConfigRequired = errors.New("config: voice tencent_asr and minimax configuration is required when enabled")
	// ErrMediaConfigRequired 表示启用图片服务后缺少存储或签名配置。
	ErrMediaConfigRequired = errors.New("config: media storage_path and signing secret are required when enabled")
	// ErrSchedulerConfigInvalid 表示定时任务时区或 cron 表达式无效。
	ErrSchedulerConfigInvalid = errors.New("config: scheduler timezone or cron spec is invalid")
)

// Config 定义项目业务配置。
type Config struct {
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	WeChat    WeChatConfig    `mapstructure:"wechat"`
	Voice     VoiceConfig     `mapstructure:"voice"`
	Media     MediaConfig     `mapstructure:"media"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
}

// SchedulerConfig 定义进程内定时任务调度配置。
type SchedulerConfig struct {
	Enabled          bool               `mapstructure:"enabled"`
	Timezone         string             `mapstructure:"timezone"`
	DailyTaskSummary ScheduledJobConfig `mapstructure:"daily_task_summary"`
}

// ScheduledJobConfig 定义单个定时任务的开关和运行约束。
type ScheduledJobConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Spec       string        `mapstructure:"spec"`
	Timeout    time.Duration `mapstructure:"timeout"`
	RunOnStart bool          `mapstructure:"run_on_start"`
}

// MediaConfig 定义图片上传、压缩和短期签名地址配置。
type MediaConfig struct {
	Enabled         bool           `mapstructure:"enabled"`
	StoragePath     string         `mapstructure:"storage_path"`
	SigningSecret   string         `mapstructure:"signing_secret"`
	MaxUploadBytes  int64          `mapstructure:"max_upload_bytes"`
	MaxImageEdge    int            `mapstructure:"max_image_edge"`
	JPEGQuality     int            `mapstructure:"jpeg_quality"`
	SignedURLTTL    time.Duration  `mapstructure:"signed_url_ttl"`
	UploadRateLimit VoiceRateLimit `mapstructure:"upload_rate_limit"`
}

// VoiceConfig 定义语音生活记录 Demo 配置。
type VoiceConfig struct {
	Enabled       bool             `mapstructure:"enabled"`
	MaxAudioBytes int64            `mapstructure:"max_audio_bytes"`
	RateLimit     VoiceRateLimit   `mapstructure:"rate_limit"`
	TencentASR    TencentASRConfig `mapstructure:"tencent_asr"`
	MiniMax       MiniMaxConfig    `mapstructure:"minimax"`
}

// VoiceRateLimit 定义单实例下每个用户的语音接口调用上限。
type VoiceRateLimit struct {
	PerMinute int `mapstructure:"per_minute"`
	PerDay    int `mapstructure:"per_day"`
}

// TencentASRConfig 定义腾讯云一句话识别配置。
type TencentASRConfig struct {
	SecretID        string `mapstructure:"secret_id"`
	SecretKey       string `mapstructure:"secret_key"`
	Region          string `mapstructure:"region"`
	EngineModelType string `mapstructure:"engine_model_type"`
}

// MiniMaxConfig 定义 MiniMax Anthropic Messages 兼容接口配置。
type MiniMaxConfig struct {
	Endpoint string        `mapstructure:"endpoint"`
	APIKey   string        `mapstructure:"api_key"`
	Model    string        `mapstructure:"model"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// AuthConfig 定义 JWT 鉴权配置。
type AuthConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	JWTSecret          string        `mapstructure:"jwt_secret"`
	TokenTTL           time.Duration `mapstructure:"token_ttl"`
	BootstrapFirstUser bool          `mapstructure:"bootstrap_first_user"`
	RequiredPaths      []string      `mapstructure:"required_paths"`
}

// WeChatConfig 定义微信小程序登录配置。
type WeChatConfig struct {
	AppID       string `mapstructure:"app_id"`
	AppSecret   string `mapstructure:"app_secret"`
	MockEnabled bool   `mapstructure:"mock_enabled"`
	MockOpenID  string `mapstructure:"mock_open_id"`
}

// DatabaseConfig 定义项目使用的数据库组件。
type DatabaseConfig struct {
	MySQL MySQLConfig `mapstructure:"mysql"`
	Redis RedisConfig `mapstructure:"redis"`
}

// MySQLConfig 使用 enabled 控制是否初始化 MySQL。
type MySQLConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	hgorm.Config `mapstructure:",squash"`
}

// RedisConfig 使用 enabled 控制是否初始化 Redis。
type RedisConfig struct {
	Enabled       bool `mapstructure:"enabled"`
	hredis.Config `mapstructure:",squash"`
}

var current *Config

const defaultAccessTokenTTL = 180 * 24 * time.Hour

const (
	defaultVoiceMaxAudioBytes  = int64(2 * 1024 * 1024)
	defaultVoiceTimeout        = 45 * time.Second
	defaultVoicePerMinute      = 5
	defaultVoicePerDay         = 50
	defaultMediaMaxUploadBytes = int64(5 * 1024 * 1024)
	defaultMediaMaxImageEdge   = 1200
	defaultMediaJPEGQuality    = 82
	defaultMediaSignedURLTTL   = 2 * time.Hour
	defaultMediaPerMinute      = 10
	defaultMediaPerDay         = 100
	defaultSchedulerTimezone   = "Asia/Shanghai"
	defaultTaskSummarySpec     = "0 6 * * *"
	defaultTaskSummaryTimeout  = 2 * time.Minute
)

func defaultAuthRequiredPaths() []string {
	return []string{
		"/v1/life-core/user/*",
		"/v1/life-core/task/*",
		"/v1/life-core/activity_log/*",
		"/v1/life-core/bill/*",
		"/v1/life-core/record/*",
		"/v1/life-core/report/*",
		"/v1/life-core/restaurant/*",
		"/v1/life-core/place/*",
		"/v1/life-core/memory/*",
		"/v1/life-core/favorite/*",
		"/v1/life-core/access/*",
		"/v1/life-core/admin/*",
		"/v1/life-core/media/upload",
	}
}

// Startup 初始化业务配置。
func Startup() error {
	cfg, err := load(".", "conf")
	if err != nil {
		return err
	}
	current = cfg
	return nil
}

// Get 返回启动时加载的只读配置。
func Get() *Config {
	return current
}

func load(path, name string) (*Config, error) {
	cfg := &Config{
		Auth: AuthConfig{
			TokenTTL:      defaultAccessTokenTTL,
			RequiredPaths: defaultAuthRequiredPaths(),
		},
		Voice: VoiceConfig{
			MaxAudioBytes: defaultVoiceMaxAudioBytes,
			RateLimit: VoiceRateLimit{
				PerMinute: defaultVoicePerMinute,
				PerDay:    defaultVoicePerDay,
			},
			TencentASR: TencentASRConfig{
				Region:          "ap-guangzhou",
				EngineModelType: "16k_zh",
			},
			MiniMax: MiniMaxConfig{
				Endpoint: "https://api.minimaxi.com/anthropic/v1/messages",
				Model:    "MiniMax-M2.7",
				Timeout:  defaultVoiceTimeout,
			},
		},
		Media: MediaConfig{
			StoragePath:    "data/media",
			MaxUploadBytes: defaultMediaMaxUploadBytes,
			MaxImageEdge:   defaultMediaMaxImageEdge,
			JPEGQuality:    defaultMediaJPEGQuality,
			SignedURLTTL:   defaultMediaSignedURLTTL,
			UploadRateLimit: VoiceRateLimit{
				PerMinute: defaultMediaPerMinute,
				PerDay:    defaultMediaPerDay,
			},
		},
		Scheduler: SchedulerConfig{
			Timezone: defaultSchedulerTimezone,
			DailyTaskSummary: ScheduledJobConfig{
				Spec:    defaultTaskSummarySpec,
				Timeout: defaultTaskSummaryTimeout,
			},
		},
	}
	if err := hconfig.Load(path, name, cfg); err != nil {
		return nil, err
	}
	if err := applyEnvironmentOverrides(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyEnvironmentOverrides 仅覆盖敏感配置，方便本地和服务器通过进程环境注入密钥。
func applyEnvironmentOverrides(cfg *Config) error {
	if cfg == nil {
		return errors.New("config: nil config")
	}
	if value := strings.TrimSpace(os.Getenv("LIFE_CORE_VOICE_ENABLED")); value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return errors.New("config: LIFE_CORE_VOICE_ENABLED must be true or false")
		}
		cfg.Voice.Enabled = enabled
	}
	if value := strings.TrimSpace(os.Getenv("TENCENTCLOUD_SECRET_ID")); value != "" {
		cfg.Voice.TencentASR.SecretID = value
	}
	if value := strings.TrimSpace(os.Getenv("TENCENTCLOUD_SECRET_KEY")); value != "" {
		cfg.Voice.TencentASR.SecretKey = value
	}
	if value := strings.TrimSpace(os.Getenv("LIFE_CORE_MINIMAX_API_KEY")); value != "" {
		cfg.Voice.MiniMax.APIKey = value
	}
	if value := strings.TrimSpace(os.Getenv("LIFE_CORE_MINIMAX_ENDPOINT")); value != "" {
		cfg.Voice.MiniMax.Endpoint = value
	}
	if value := strings.TrimSpace(os.Getenv("LIFE_CORE_MINIMAX_MODEL")); value != "" {
		cfg.Voice.MiniMax.Model = value
	}
	if value := strings.TrimSpace(os.Getenv("LIFE_CORE_MEDIA_STORAGE_PATH")); value != "" {
		cfg.Media.StoragePath = value
	}
	if value := strings.TrimSpace(os.Getenv("LIFE_CORE_MEDIA_SIGNING_SECRET")); value != "" {
		cfg.Media.SigningSecret = value
	}
	return nil
}

// Validate 校验已启用组件的必填配置。
func (c *Config) Validate() error {
	if c.Database.MySQL.Enabled && strings.TrimSpace(c.Database.MySQL.DSN) == "" {
		return ErrMySQLDSNRequired
	}
	if c.Database.Redis.Enabled && strings.TrimSpace(c.Database.Redis.Addr) == "" {
		return ErrRedisAddrRequired
	}
	if c.Auth.Enabled {
		if len(strings.TrimSpace(c.Auth.JWTSecret)) < 32 {
			return ErrJWTSecretRequired
		}
		if c.Auth.TokenTTL <= 0 {
			c.Auth.TokenTTL = defaultAccessTokenTTL
		}
		if !c.WeChat.MockEnabled &&
			(strings.TrimSpace(c.WeChat.AppID) == "" || strings.TrimSpace(c.WeChat.AppSecret) == "") {
			return ErrWeChatConfigRequired
		}
	}
	if c.Voice.Enabled {
		if c.Voice.MaxAudioBytes <= 0 {
			c.Voice.MaxAudioBytes = defaultVoiceMaxAudioBytes
		}
		if c.Voice.MiniMax.Timeout <= 0 {
			c.Voice.MiniMax.Timeout = defaultVoiceTimeout
		}
		if c.Voice.RateLimit.PerMinute <= 0 {
			c.Voice.RateLimit.PerMinute = defaultVoicePerMinute
		}
		if c.Voice.RateLimit.PerDay <= 0 {
			c.Voice.RateLimit.PerDay = defaultVoicePerDay
		}
		asr := c.Voice.TencentASR
		miniMax := c.Voice.MiniMax
		if strings.TrimSpace(asr.SecretID) == "" || strings.TrimSpace(asr.SecretKey) == "" ||
			strings.TrimSpace(asr.Region) == "" || strings.TrimSpace(asr.EngineModelType) == "" ||
			strings.TrimSpace(miniMax.Endpoint) == "" || strings.TrimSpace(miniMax.APIKey) == "" ||
			strings.TrimSpace(miniMax.Model) == "" {
			return ErrVoiceConfigRequired
		}
	}
	if c.Media.Enabled {
		if strings.TrimSpace(c.Media.StoragePath) == "" {
			c.Media.StoragePath = "data/media"
		}
		if c.Media.MaxUploadBytes <= 0 {
			c.Media.MaxUploadBytes = defaultMediaMaxUploadBytes
		}
		if c.Media.MaxImageEdge <= 0 {
			c.Media.MaxImageEdge = defaultMediaMaxImageEdge
		}
		if c.Media.JPEGQuality < 1 || c.Media.JPEGQuality > 100 {
			c.Media.JPEGQuality = defaultMediaJPEGQuality
		}
		if c.Media.SignedURLTTL <= 0 {
			c.Media.SignedURLTTL = defaultMediaSignedURLTTL
		}
		if c.Media.UploadRateLimit.PerMinute <= 0 {
			c.Media.UploadRateLimit.PerMinute = defaultMediaPerMinute
		}
		if c.Media.UploadRateLimit.PerDay <= 0 {
			c.Media.UploadRateLimit.PerDay = defaultMediaPerDay
		}
		if len(c.MediaSigningSecret()) < 32 {
			return ErrMediaConfigRequired
		}
	}
	if strings.TrimSpace(c.Scheduler.Timezone) == "" {
		c.Scheduler.Timezone = defaultSchedulerTimezone
	}
	if strings.TrimSpace(c.Scheduler.DailyTaskSummary.Spec) == "" {
		c.Scheduler.DailyTaskSummary.Spec = defaultTaskSummarySpec
	}
	if c.Scheduler.DailyTaskSummary.Timeout <= 0 {
		c.Scheduler.DailyTaskSummary.Timeout = defaultTaskSummaryTimeout
	}
	if c.Scheduler.Enabled {
		if _, err := time.LoadLocation(c.Scheduler.Timezone); err != nil {
			return fmt.Errorf("%w: timezone %q", ErrSchedulerConfigInvalid, c.Scheduler.Timezone)
		}
		if c.Scheduler.DailyTaskSummary.Enabled {
			if _, err := cron.ParseStandard(c.Scheduler.DailyTaskSummary.Spec); err != nil {
				return fmt.Errorf("%w: daily_task_summary spec: %v", ErrSchedulerConfigInvalid, err)
			}
		}
	}
	return nil
}

// MediaSigningSecret 返回图片URL签名密钥；未单独配置时复用JWT密钥。
func (c *Config) MediaSigningSecret() string {
	if c == nil {
		return ""
	}
	if secret := strings.TrimSpace(c.Media.SigningSecret); secret != "" {
		return secret
	}
	return strings.TrimSpace(c.Auth.JWTSecret)
}
