package control

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/smartystreets/goconvey/convey"
)

func TestParseMediaAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	Convey("解析Nginx传递的原始签名图片地址", t, func() {
		request := httptest.NewRequest("GET", "/v1/life-core/media/authorize", nil)
		request.Header.Set("X-Original-URI", "/media/restaurant/2026/09/a.jpg?expires=123&signature=abc")
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = request

		objectKey, expires, signature, err := parseMediaAuthorization(ctx)

		So(err, ShouldBeNil)
		So(objectKey, ShouldEqual, "restaurant/2026/09/a.jpg")
		So(expires, ShouldEqual, "123")
		So(signature, ShouldEqual, "abc")
	})

	Convey("拒绝非媒体目录的原始地址", t, func() {
		request := httptest.NewRequest("GET", "/v1/life-core/media/authorize", nil)
		request.Header.Set("X-Original-URI", "/v1/life-core/health")
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = request

		_, _, _, err := parseMediaAuthorization(ctx)

		So(err, ShouldNotBeNil)
	})
}
