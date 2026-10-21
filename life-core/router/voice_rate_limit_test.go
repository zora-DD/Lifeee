package router

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAIRateLimiter(t *testing.T) {
	Convey("同一用户每分钟最多调用五次", t, func() {
		now := time.Date(2026, 9, 3, 10, 20, 20, 0, time.Local)
		limiter := newAIRateLimiter()
		limiter.now = func() time.Time { return now }

		for i := 0; i < 5; i++ {
			allowed, _ := limiter.allow("user-1", 5, 50)
			So(allowed, ShouldBeTrue)
		}
		allowed, retryAfter := limiter.allow("user-1", 5, 50)
		So(allowed, ShouldBeFalse)
		So(retryAfter, ShouldEqual, 40*time.Second)

		now = now.Add(time.Minute)
		allowed, _ = limiter.allow("user-1", 5, 50)
		So(allowed, ShouldBeTrue)
	})

	Convey("同一用户每天最多调用五十次且用户之间互不影响", t, func() {
		now := time.Date(2026, 9, 3, 10, 20, 0, 0, time.Local)
		limiter := newAIRateLimiter()
		limiter.now = func() time.Time { return now }

		for i := 0; i < 50; i++ {
			allowed, _ := limiter.allow("user-1", 100, 50)
			So(allowed, ShouldBeTrue)
		}
		allowed, _ := limiter.allow("user-1", 100, 50)
		So(allowed, ShouldBeFalse)
		allowed, _ = limiter.allow("user-2", 100, 50)
		So(allowed, ShouldBeTrue)

		now = now.Add(24 * time.Hour)
		allowed, _ = limiter.allow("user-1", 100, 50)
		So(allowed, ShouldBeTrue)
	})

	Convey("并发计数不会超过限制", t, func() {
		limiter := newAIRateLimiter()
		allowedResults := make(chan bool, 20)
		for i := 0; i < 20; i++ {
			go func(index int) {
				allowed, _ := limiter.allow(fmt.Sprintf("user-%d", index%2), 5, 50)
				allowedResults <- allowed
			}(i)
		}
		allowedCount := 0
		for i := 0; i < 20; i++ {
			if <-allowedResults {
				allowedCount++
			}
		}
		So(allowedCount, ShouldEqual, 10)
	})
}
