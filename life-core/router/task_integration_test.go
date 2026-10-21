//go:build integration

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow"

	"life-core/config"
	"life-core/infrastructure/database"
)

func TestTaskListHTTP(t *testing.T) {
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
	t.Cleanup(func() {
		_ = database.Shutdown(context.Background())
	})
	app, err := hollow.NewApp(hollow.AppOption{ConfigPath: ".", ConfigName: "conf"})
	if err != nil {
		t.Fatal(err)
	}
	Register(app)
	server := httptest.NewServer(app.Engine)
	t.Cleanup(server.Close)

	Convey("POST /v1/life-core/task/list 返回统一分页响应", t, func() {
		status, body := requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/list", []byte(`{"page":1,"pageSize":2,"status":"todo"}`))
		t.Logf("POST /v1/life-core/task/list response: %s", body)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"code":0`)
		So(string(body), ShouldContainSubstring, `"page":1`)
		So(string(body), ShouldContainSubstring, `"pageSize":2`)
		So(string(body), ShouldContainSubstring, `"has_more":`)
		So(strings.Contains(string(body), `"items"`), ShouldBeTrue)
	})

	Convey("POST /v1/life-core/task/list 拒绝非法优先级", t, func() {
		status, body := requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/list", []byte(`{"priority":"urgent"}`))
		So(status, ShouldEqual, http.StatusBadRequest)
		So(string(body), ShouldContainSubstring, `"code":1100`)
	})

	Convey("任务 REST 接口完成创建、详情、编辑、状态和删除链路", t, func() {
		createBody := []byte(`{"title":"接口联调任务","description":"integration test","priority":"high","category":"工作","dueAt":"2026-09-04T18:00:00+08:00"}`)
		status, body := requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/create", createBody)
		So(status, ShouldEqual, http.StatusOK)
		var created struct {
			Code int `json:"code"`
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		So(json.Unmarshal(body, &created), ShouldBeNil)
		So(created.Code, ShouldEqual, 0)
		So(created.Data.ID, ShouldNotBeEmpty)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/detail", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"title":"接口联调任务"`)
		So(string(body), ShouldContainSubstring, `"dueAt":"2026-09-04T18:00:00+08:00"`)

		updateBody := []byte(`{"id":"` + created.Data.ID + `","title":"接口联调任务-已编辑","description":"updated","priority":"medium","category":"生活","dueAt":"2026-09-05T18:00:00+08:00"}`)
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/modify", updateBody)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"title":"接口联调任务-已编辑"`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/status/modify", []byte(`{"id":"`+created.Data.ID+`","status":"done"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"status":"done"`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/delete", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"deleted":true`)

		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/task/detail", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusNotFound)
	})
}

var integrationTokens sync.Map

func requestTaskJSON(t *testing.T, method, requestURL string, body []byte) (int, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, requestURL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+integrationTokenForServer(t, parsedURL.Scheme+"://"+parsedURL.Host))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %s response: %s", method, requestURL, payload)
	return response.StatusCode, payload
}

func integrationTokenForServer(t *testing.T, serverURL string) string {
	t.Helper()
	if token, ok := integrationTokens.Load(serverURL); ok {
		return token.(string)
	}
	request, err := http.NewRequest(
		http.MethodPost, serverURL+"/v1/life-core/auth/login",
		bytes.NewBufferString(`{"code":"integration-code"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || payload.Data.AccessToken == "" {
		t.Fatalf("integration login failed: status=%d", response.StatusCode)
	}
	integrationTokens.Store(serverURL, payload.Data.AccessToken)
	return payload.Data.AccessToken
}
