package auth

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTokenManager(t *testing.T) {
	Convey("JWT 可以签发并解析用户 ID", t, func() {
		manager, err := NewTokenManager("unit-test-jwt-secret-with-at-least-32-characters", time.Hour)
		So(err, ShouldBeNil)

		token, expiresAt, err := manager.Issue(42)
		So(err, ShouldBeNil)
		So(token, ShouldNotBeEmpty)
		So(expiresAt.After(time.Now()), ShouldBeTrue)

		userID, err := manager.Parse(token)
		So(err, ShouldBeNil)
		So(userID, ShouldEqual, uint64(42))
	})

	Convey("拒绝非法 Token 和不完整配置", t, func() {
		_, err := NewTokenManager("short", time.Hour)
		So(err, ShouldEqual, ErrInvalidTokenConfig)

		manager, err := NewTokenManager("unit-test-jwt-secret-with-at-least-32-characters", time.Hour)
		So(err, ShouldBeNil)
		_, err = manager.Parse("not-a-token")
		So(err, ShouldNotBeNil)
	})

	Convey("拒绝已经过期的 Token", t, func() {
		manager, err := NewTokenManager("unit-test-jwt-secret-with-at-least-32-characters", time.Hour)
		So(err, ShouldBeNil)
		now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
		manager.now = func() time.Time { return now }
		token, _, err := manager.Issue(42)
		So(err, ShouldBeNil)

		manager.now = func() time.Time { return now.Add(2 * time.Hour) }
		_, err = manager.Parse(token)
		So(err, ShouldNotBeNil)
	})
}
