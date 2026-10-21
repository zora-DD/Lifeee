//go:build integration

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow"

	"life-core/auth"
	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/model"
)

func TestAccessApprovalHTTP(t *testing.T) {
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
	t.Cleanup(server.Close)

	openID := fmt.Sprintf("integration-pending-%d", time.Now().UnixNano())
	target := &model.User{
		WxOpenID: openID, Nickname: "待审批体验官", Status: model.UserStatusPending, Role: model.UserRoleUser,
	}
	if err := dao.NewUserDao().Create(context.Background(), database.MySQL(), target); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = database.MySQL().Where("wx_open_id = ?", openID).Delete(&model.User{}).Error
	})
	tokenManager, err := auth.NewTokenManager(config.Get().Auth.JWTSecret, config.Get().Auth.TokenTTL)
	if err != nil {
		t.Fatal(err)
	}
	targetToken, _, err := tokenManager.Issue(target.ID)
	if err != nil {
		t.Fatal(err)
	}
	adminToken := integrationTokenForServer(t, server.URL)

	Convey("待审批用户可查状态但不能读取业务数据", t, func() {
		status, body := requestAccessJSON(t, server.URL+"/v1/life-core/access/status", []byte(`{}`), targetToken)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"accessStatus":"pending"`)

		status, _ = requestAccessJSON(t, server.URL+"/v1/life-core/task/list", []byte(`{}`), targetToken)
		So(status, ShouldEqual, http.StatusForbidden)
	})

	Convey("管理员查询申请、通过、拒绝形成完整审批链路", t, func() {
		status, body := requestAccessJSON(t, server.URL+"/v1/life-core/admin/access_request/list", []byte(`{"status":"pending","page":1,"pageSize":100}`), adminToken)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"nickname":"待审批体验官"`)
		So(string(body), ShouldContainSubstring, `"has_more":false`)

		approveBody, _ := json.Marshal(map[string]string{"userId": fmt.Sprint(target.ID), "status": model.UserStatusApproved})
		status, body = requestAccessJSON(t, server.URL+"/v1/life-core/admin/access_request/modify", approveBody, adminToken)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"accessStatus":"approved"`)

		status, _ = requestAccessJSON(t, server.URL+"/v1/life-core/task/list", []byte(`{"page":1,"pageSize":1}`), targetToken)
		So(status, ShouldEqual, http.StatusOK)
		status, _ = requestAccessJSON(t, server.URL+"/v1/life-core/admin/access_request/list", []byte(`{"status":"pending"}`), targetToken)
		So(status, ShouldEqual, http.StatusForbidden)

		rejectBody, _ := json.Marshal(map[string]string{"userId": fmt.Sprint(target.ID), "status": model.UserStatusRejected})
		status, body = requestAccessJSON(t, server.URL+"/v1/life-core/admin/access_request/modify", rejectBody, adminToken)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"accessStatus":"rejected"`)

		status, _ = requestAccessJSON(t, server.URL+"/v1/life-core/task/list", []byte(`{}`), targetToken)
		So(status, ShouldEqual, http.StatusForbidden)
	})
}

func requestAccessJSON(t *testing.T, requestURL string, body []byte, token string) (int, []byte) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, payload
}
