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

func TestPlaceHTTP(t *testing.T) {
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

	Convey("地点接口完成创建、筛选、详情、编辑和删除链路", t, func() {
		name := fmt.Sprintf("接口联调湖畔-%d", time.Now().UnixNano())
		createBody, _ := json.Marshal(map[string]any{
			"name": name, "category": "自然风景", "icon": "🌿", "visited": true,
			"visitCount": 2, "lastVisitedAt": "2026-08-17", "feature": "湖边日落和大片草坪",
			"note": "适合傍晚散步", "address": "公园东门", "bestTime": "17:00 后",
			"latitude": "30.1234567", "longitude": "120.7654321",
		})
		status, body := requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/place/create", createBody)
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

		listBody, _ := json.Marshal(map[string]any{"category": "自然风景", "status": "visited", "keyword": name, "page": 1, "pageSize": 1})
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/place/list", listBody)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, name)
		So(string(body), ShouldContainSubstring, `"has_more":`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/place/detail", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"visitCount":2`)
		So(string(body), ShouldContainSubstring, `"lastVisitedAt":"2026-08-17"`)

		updateBody := []byte(`{"id":"` + created.Data.ID + `","name":"接口联调湖畔-想去","category":"自然风景","icon":"🏞️","visited":false,"visitCount":9,"lastVisitedAt":"2026-08-17","feature":"更新后的地点特色","note":"下次去看看","address":"公园西门","bestTime":"清晨"}`)
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/place/modify", updateBody)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"visited":false`)
		So(string(body), ShouldContainSubstring, `"visitCount":0`)
		So(string(body), ShouldContainSubstring, `"lastVisitedAt":""`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/place/list", []byte(`{"status":"wishlist","page":1,"pageSize":100}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"name":"接口联调湖畔-想去"`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/place/delete", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"deleted":true`)

		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/place/detail", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusNotFound)
	})
}
