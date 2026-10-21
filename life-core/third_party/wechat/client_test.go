package wechat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClientExchange(t *testing.T) {
	Convey("正确解析微信 code2session 响应", t, func() {
		var receivedAppID string
		var receivedCode string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAppID = r.URL.Query().Get("appid")
			receivedCode = r.URL.Query().Get("js_code")
			_, _ = w.Write([]byte(`{"openid":"open-id","unionid":"union-id"}`))
		}))
		defer server.Close()
		client := NewClient("app-id", "app-secret")
		client.endpoint = server.URL

		session, err := client.Exchange(context.Background(), "wx-code")

		So(err, ShouldBeNil)
		So(receivedAppID, ShouldEqual, "app-id")
		So(receivedCode, ShouldEqual, "wx-code")
		So(session.OpenID, ShouldEqual, "open-id")
		So(session.UnionID, ShouldEqual, "union-id")
	})

	Convey("微信业务错误不会被当作登录成功", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code"}`))
		}))
		defer server.Close()
		client := NewClient("app-id", "app-secret")
		client.endpoint = server.URL

		_, err := client.Exchange(context.Background(), "bad-code")
		So(err, ShouldNotBeNil)
	})
}
