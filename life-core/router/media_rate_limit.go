package router

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"life-core/auth"
	"life-core/config"
)

// configuredMediaUploadRateLimitMiddleware 使用现有内存窗口限制单用户图片上传频率。
func configuredMediaUploadRateLimitMiddleware() gin.HandlerFunc {
	limiter := newAIRateLimiter()
	return func(c *gin.Context) {
		if c.Request.URL.Path != "/v1/life-core/media/upload" {
			c.Next()
			return
		}
		cfg := config.Get()
		if cfg == nil || !cfg.Media.Enabled {
			c.Next()
			return
		}
		userID, ok := auth.UserIDFromContext(c.Request.Context())
		if !ok {
			c.Next()
			return
		}
		allowed, retryAfter := limiter.allow(
			strconv.FormatUint(userID, 10),
			cfg.Media.UploadRateLimit.PerMinute,
			cfg.Media.UploadRateLimit.PerDay,
		)
		if allowed {
			c.Next()
			return
		}
		retrySeconds := max(1, int(retryAfter.Seconds()))
		c.Header("Retry-After", strconv.Itoa(retrySeconds))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"code": 429,
			"msg":  "图片上传过于频繁，请稍后再试",
			"data": nil,
		})
	}
}
