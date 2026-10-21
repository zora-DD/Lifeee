package router

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaynedu/hollow/pkg/hecode"
	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/auth"
	"life-core/config"
)

var aiFeaturePaths = map[string]struct{}{
	"/v1/life-core/activity_log/voice/recognize": {},
	"/v1/life-core/task/voice/recognize":         {},
	"/v1/life-core/task/ai_summary":              {},
	"/v1/life-core/record/parse":                 {},
	"/v1/life-core/record/voice/recognize":       {},
	"/v1/life-core/report/life_summary":          {},
}

type aiUsage struct {
	minuteStart time.Time
	minuteCount int
	dayKey      string
	dayCount    int
}

type aiRateLimiter struct {
	mu    sync.Mutex
	users map[string]*aiUsage
	now   func() time.Time
}

func newAIRateLimiter() *aiRateLimiter {
	return &aiRateLimiter{users: make(map[string]*aiUsage), now: time.Now}
}

// allow 使用固定分钟窗口和中国自然日窗口限制单用户调用次数。
func (l *aiRateLimiter) allow(key string, perMinute, perDay int) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	minuteStart := now.Truncate(time.Minute)
	chinaTime := now.In(time.FixedZone("CST", 8*60*60))
	dayKey := chinaTime.Format("2006-01-02")
	usage := l.users[key]
	if usage == nil {
		usage = &aiUsage{minuteStart: minuteStart, dayKey: dayKey}
		l.users[key] = usage
	}
	if usage.minuteStart != minuteStart {
		usage.minuteStart = minuteStart
		usage.minuteCount = 0
	}
	if usage.dayKey != dayKey {
		usage.dayKey = dayKey
		usage.dayCount = 0
	}
	if usage.dayCount >= perDay {
		nextDay := time.Date(chinaTime.Year(), chinaTime.Month(), chinaTime.Day()+1, 0, 0, 0, 0, chinaTime.Location())
		return false, nextDay.Sub(chinaTime)
	}
	if usage.minuteCount >= perMinute {
		return false, minuteStart.Add(time.Minute).Sub(now)
	}
	usage.minuteCount++
	usage.dayCount++
	return true, 0
}

func configuredAIRateLimitMiddleware() gin.HandlerFunc {
	limiter := newAIRateLimiter()
	return func(c *gin.Context) {
		if _, ok := aiFeaturePaths[c.Request.URL.Path]; !ok {
			c.Next()
			return
		}

		cfg := config.Get()
		if cfg == nil || !cfg.Voice.Enabled {
			c.Next()
			return
		}
		userID, ok := auth.UserIDFromContext(c.Request.Context())
		key := c.ClientIP()
		if ok {
			key = strconv.FormatUint(userID, 10)
		}
		allowed, retryAfter := limiter.allow(key, cfg.Voice.RateLimit.PerMinute, cfg.Voice.RateLimit.PerDay)
		logger := hlog.FromContext(c.Request.Context()).Named("AIRequest").With(
			hlog.String("path", c.Request.URL.Path),
			hlog.Uint64("user_id", userID),
		)
		if !allowed {
			retrySeconds := int(retryAfter.Seconds())
			if retrySeconds < 1 {
				retrySeconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(retrySeconds))
			logger.Warn("AI request rate limited",
				hlog.Bool("success", false),
				hlog.Int("status", http.StatusTooManyRequests),
				hlog.Int64("duration_ms", 0),
			)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "AI 请求过于频繁，请稍后再试",
				"data": nil,
			})
			return
		}

		startedAt := time.Now()
		c.Next()
		status := c.Writer.Status()
		success := len(c.Errors) == 0 && status >= http.StatusOK && status < http.StatusMultipleChoices
		if ginErr := c.Errors.Last(); ginErr != nil && status < http.StatusBadRequest {
			status, _ = hecode.Failure("", ginErr.Err)
		}
		logger.Info("AI request completed",
			hlog.Bool("success", success),
			hlog.Int("status", status),
			hlog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
	}
}
