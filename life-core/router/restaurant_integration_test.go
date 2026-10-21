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

func TestRestaurantHTTP(t *testing.T) {
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

	Convey("吃喝接口完成店铺、消费、汇总、推荐与删除链路", t, func() {
		name := fmt.Sprintf("接口联调饮品-%d", time.Now().UnixNano())
		createBody, _ := json.Marshal(map[string]any{
			"name": name, "category": "饮品", "cuisine": "咖啡",
			"preference": "love", "address": "测试路 1 号", "businessHours": "09:00-20:00",
		})
		status, body := requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/create", createBody)
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

		listBody, _ := json.Marshal(map[string]any{"category": "饮品", "keyword": name, "page": 1, "pageSize": 1})
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/list", listBody)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"has_more":`)
		So(string(body), ShouldContainSubstring, name)

		updateBody := []byte(`{"id":"` + created.Data.ID + `","name":"接口联调饮品-已编辑","category":"饮品","cuisine":"手冲咖啡","preference":"must","address":"测试路 2 号","businessHours":"10:00-21:00"}`)
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/modify", updateBody)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"preference":"must"`)

		firstBody := []byte(`{"restaurantId":"` + created.Data.ID + `","consumedAt":"2026-09-01","amount":"28.50","meal":"拿铁","note":"少冰"}`)
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/consumption/create", firstBody)
		So(status, ShouldEqual, http.StatusOK)
		var firstConsumption struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		So(json.Unmarshal(body, &firstConsumption), ShouldBeNil)
		So(firstConsumption.Data.ID, ShouldNotBeEmpty)

		secondBody := []byte(`{"restaurantId":"` + created.Data.ID + `","consumedAt":"2026-09-02","amount":"16.20","meal":"美式","note":"正常冰"}`)
		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/consumption/create", secondBody)
		So(status, ShouldEqual, http.StatusOK)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/detail", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"visitCount":2`)
		So(string(body), ShouldContainSubstring, `"totalSpent":"44.70"`)
		So(string(body), ShouldContainSubstring, `"lastVisitedAt":"2026-09-02"`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/consumption/list", []byte(`{"restaurantId":"`+created.Data.ID+`","page":1,"pageSize":1}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"total":2`)
		So(string(body), ShouldContainSubstring, `"has_more":true`)

		editConsumption := []byte(`{"restaurantId":"` + created.Data.ID + `","id":"` + firstConsumption.Data.ID + `","consumedAt":"2026-09-03","amount":"30.00","meal":"燕麦拿铁","note":"已编辑"}`)
		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/consumption/modify", editConsumption)
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"amount":"30.00"`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/recommend", []byte(`{"category":"饮品"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"category":"饮品"`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/consumption/delete", []byte(`{"restaurantId":"`+created.Data.ID+`","id":"`+firstConsumption.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"deleted":true`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/detail", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"visitCount":1`)
		So(string(body), ShouldContainSubstring, `"totalSpent":"16.20"`)

		status, body = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/delete", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusOK)
		So(string(body), ShouldContainSubstring, `"deleted":true`)

		status, _ = requestTaskJSON(t, http.MethodPost, server.URL+"/v1/life-core/restaurant/detail", []byte(`{"id":"`+created.Data.ID+`"}`))
		So(status, ShouldEqual, http.StatusNotFound)
	})
}
