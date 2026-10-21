//go:build integration

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow"

	"life-core/auth"
	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
)

func TestAuthHTTP(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })
	if err := config.Startup(); err != nil {
		t.Fatal(err)
	}
	if err := database.InitMySQL(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Shutdown(context.Background()) })
	app, err := hollow.NewApp(hollow.AppOption{ConfigPath: ".", ConfigName: "conf"})
	if err != nil {
		t.Fatal(err)
	}
	Register(app)
	server := httptest.NewServer(app.Engine)
	defer server.Close()

	Convey("微信登录后可以携带 Token 获取当前用户", t, func() {
		loginRequest, _ := http.NewRequest(
			http.MethodPost, server.URL+"/v1/life-core/auth/login",
			bytes.NewBufferString(`{"code":"integration-code"}`),
		)
		loginRequest.Header.Set("Content-Type", "application/json")
		loginResponse, err := http.DefaultClient.Do(loginRequest)
		So(err, ShouldBeNil)
		defer loginResponse.Body.Close()
		So(loginResponse.StatusCode, ShouldEqual, http.StatusOK)
		var loginBody struct {
			Code int `json:"code"`
			Data struct {
				AccessToken string `json:"access_token"`
				User        struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"data"`
		}
		So(json.NewDecoder(loginResponse.Body).Decode(&loginBody), ShouldBeNil)
		So(loginBody.Code, ShouldEqual, 0)
		So(loginBody.Data.AccessToken, ShouldNotBeEmpty)

		tokens, err := auth.NewTokenManager(config.Get().Auth.JWTSecret, config.Get().Auth.TokenTTL)
		So(err, ShouldBeNil)
		userID, err := tokens.Parse(loginBody.Data.AccessToken)
		So(err, ShouldBeNil)
		user, err := dao.NewUserDao().GetByID(context.Background(), database.MySQL(), userID)
		So(err, ShouldBeNil)
		So(user.ID, ShouldEqual, userID)
		So(user.WxOpenID, ShouldNotBeEmpty)

		var uniqueOpenIDIndexCount int64
		err = database.MySQL().Raw(`
			SELECT COUNT(*)
			FROM information_schema.statistics
			WHERE table_schema = DATABASE()
			  AND table_name = 'users'
			  AND index_name = 'uk_users_wx_open_id'
			  AND non_unique = 0
		`).Scan(&uniqueOpenIDIndexCount).Error
		So(err, ShouldBeNil)
		So(uniqueOpenIDIndexCount, ShouldEqual, int64(1))

		meRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/life-core/user/current", bytes.NewBufferString(`{}`))
		meRequest.Header.Set("Content-Type", "application/json")
		meRequest.Header.Set("Authorization", "Bearer "+loginBody.Data.AccessToken)
		meResponse, err := http.DefaultClient.Do(meRequest)
		So(err, ShouldBeNil)
		defer meResponse.Body.Close()
		So(meResponse.StatusCode, ShouldEqual, http.StatusOK)

		invalidRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/life-core/user/current", bytes.NewBufferString(`{}`))
		invalidRequest.Header.Set("Content-Type", "application/json")
		invalidRequest.Header.Set("Authorization", "Bearer invalid-access-token")
		invalidResponse, err := http.DefaultClient.Do(invalidRequest)
		So(err, ShouldBeNil)
		defer invalidResponse.Body.Close()
		So(invalidResponse.StatusCode, ShouldEqual, http.StatusUnauthorized)
	})
}
