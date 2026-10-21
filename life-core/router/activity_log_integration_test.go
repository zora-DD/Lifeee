//go:build integration

package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow"

	"life-core/config"
	"life-core/infrastructure/database"
)

func TestActivityLogHTTP(t *testing.T) {
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

	Convey("生活记录接口完成创建、分页、详情、编辑、月年统计和删除链路", t, func() {
		marker := fmt.Sprintf("全链路生活记录-%d", time.Now().UnixNano())
		firstBody, _ := json.Marshal(map[string]any{
			"content": marker, "occurredAt": "2026-09-02T08:30:00+08:00",
			"durationMinutes": 35, "inputType": "text",
		})
		status, body := requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/create", firstBody)
		So(status, ShouldEqual, http.StatusOK)
		var first struct {
			Code int `json:"code"`
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		So(json.Unmarshal(body, &first), ShouldBeNil)
		So(first.Code, ShouldEqual, 0)
		So(first.Data.ID, ShouldNotBeEmpty)

		secondBody, _ := json.Marshal(map[string]any{
			"content": marker + "-第二条", "occurredAt": "2026-09-03T20:00:00+08:00",
			"durationMinutes": 15, "inputType": "voice", "audioUrl": "https://example.test/audio/demo.m4a",
		})
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/create", secondBody)
		So(status, ShouldEqual, http.StatusOK)
		var second struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		So(json.Unmarshal(body, &second), ShouldBeNil)
		So(second.Data.ID, ShouldNotBeEmpty)

		listBody := []byte(`{"from":"2026-09-01T00:00:00+08:00","to":"2026-10-01T00:00:00+08:00","page":1,"pageSize":1}`)
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/list", listBody)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"pageSize":1`)
		So(string(body), ShouldContainSubstring, `"has_more":true`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/detail", []byte(`{"id":"`+first.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, marker)
		So(string(body), ShouldContainSubstring, `"durationMinutes":35`)

		updateBody, _ := json.Marshal(map[string]any{
			"id": first.Data.ID, "content": marker + "-已编辑",
			"occurredAt": "2026-09-04T09:15:00+08:00", "durationMinutes": 50, "inputType": "text",
		})
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/modify", updateBody)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, marker+"-已编辑")
		So(string(body), ShouldContainSubstring, `"durationMinutes":50`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/stats", []byte(`{"scope":"month","year":2026,"month":9}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"recordCount":`)
		So(string(body), ShouldContainSubstring, `"dailyBreakdown":`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/stats", []byte(`{"scope":"year","year":2026}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"monthlyBreakdown":`)

		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/list", []byte(`{"inputType":"image"}`))
		So(status, ShouldEqual, http.StatusBadRequest)
		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/list", []byte(`{"from":"2026-10-01T00:00:00+08:00","to":"2026-09-01T00:00:00+08:00"}`))
		So(status, ShouldEqual, http.StatusBadRequest)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/delete", []byte(`{"id":"`+first.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"deleted":true`)
		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/detail", []byte(`{"id":"`+first.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusNotFound)

		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/activity_log/delete", []byte(`{"id":"`+second.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
	})
}
