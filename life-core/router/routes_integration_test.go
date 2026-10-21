//go:build integration

package router

import (
	"net/http"
	"os"
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow"
)

func TestRegisteredRoutes(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })
	app, err := hollow.NewApp(hollow.AppOption{ConfigPath: ".", ConfigName: "conf"})
	if err != nil {
		t.Fatal(err)
	}
	Register(app)

	expected := []string{
		"GET /v1/life-core/health",
		"POST /v1/life-core/auth/login", "POST /v1/life-core/user/current",
		"POST /v1/life-core/user/contribution/list",
		"POST /v1/life-core/favorite/list", "POST /v1/life-core/favorite/status", "POST /v1/life-core/favorite/modify",
		"POST /v1/life-core/access/status", "POST /v1/life-core/access/apply", "POST /v1/life-core/admin/access_request/list", "POST /v1/life-core/admin/access_request/modify",
		"POST /v1/life-core/user/permission/list", "POST /v1/life-core/admin/user_permission/detail", "POST /v1/life-core/admin/user_permission/modify",
		"POST /v1/life-core/task/list", "POST /v1/life-core/task/detail", "POST /v1/life-core/task/create", "POST /v1/life-core/task/modify", "POST /v1/life-core/task/status/modify", "POST /v1/life-core/task/delete", "POST /v1/life-core/task/voice/recognize", "POST /v1/life-core/task/ai_summary",
		"POST /v1/life-core/task/routine/list", "POST /v1/life-core/task/routine/create", "POST /v1/life-core/task/routine/modify", "POST /v1/life-core/task/routine/delete", "POST /v1/life-core/task/routine/checkin",
		"POST /v1/life-core/activity_log/list", "POST /v1/life-core/activity_log/detail", "POST /v1/life-core/activity_log/create", "POST /v1/life-core/activity_log/modify", "POST /v1/life-core/activity_log/delete", "POST /v1/life-core/activity_log/stats",
		"POST /v1/life-core/activity_log/voice/recognize",
		"POST /v1/life-core/bill/list", "POST /v1/life-core/bill/detail", "POST /v1/life-core/bill/create", "POST /v1/life-core/bill/modify", "POST /v1/life-core/bill/delete", "POST /v1/life-core/bill/stats",
		"POST /v1/life-core/record/parse", "POST /v1/life-core/record/voice/recognize",
		"POST /v1/life-core/report/life_summary",
		"POST /v1/life-core/restaurant/list", "POST /v1/life-core/restaurant/detail", "POST /v1/life-core/restaurant/create", "POST /v1/life-core/restaurant/modify", "POST /v1/life-core/restaurant/delete", "POST /v1/life-core/restaurant/recommend",
		"POST /v1/life-core/restaurant/consumption/list", "POST /v1/life-core/restaurant/consumption/create", "POST /v1/life-core/restaurant/consumption/modify", "POST /v1/life-core/restaurant/consumption/delete",
		"POST /v1/life-core/restaurant/home/list", "POST /v1/life-core/restaurant/home/detail", "POST /v1/life-core/restaurant/home/create", "POST /v1/life-core/restaurant/home/modify", "POST /v1/life-core/restaurant/home/delete", "POST /v1/life-core/restaurant/home/cook", "POST /v1/life-core/restaurant/home/recommend",
		"POST /v1/life-core/restaurant/snack/list", "POST /v1/life-core/restaurant/snack/detail", "POST /v1/life-core/restaurant/snack/create", "POST /v1/life-core/restaurant/snack/modify", "POST /v1/life-core/restaurant/snack/delete", "POST /v1/life-core/restaurant/snack/taste", "POST /v1/life-core/restaurant/snack/recommend",
		"POST /v1/life-core/place/list", "POST /v1/life-core/place/detail", "POST /v1/life-core/place/create", "POST /v1/life-core/place/modify", "POST /v1/life-core/place/delete",
		"POST /v1/life-core/memory/list", "POST /v1/life-core/memory/create",
		"POST /v1/life-core/media/upload", "GET /v1/life-core/media/authorize",
	}
	actual := make([]string, 0, len(app.Engine.Routes()))
	for _, route := range app.Engine.Routes() {
		actual = append(actual, route.Method+" "+route.Path)
	}
	sort.Strings(expected)
	sort.Strings(actual)

	Convey("路由总表仅包含约定的 76 条 /v1 接口", t, func() {
		So(actual, ShouldResemble, expected)
		So(actual, ShouldHaveLength, 76)
		So(actual[0], ShouldNotContainSubstring, "/api/")
	})

	Convey("健康检查路由为 GET", t, func() {
		found := false
		for _, route := range app.Engine.Routes() {
			if route.Method == http.MethodGet && route.Path == "/v1/life-core/health" {
				found = true
			}
		}
		So(found, ShouldBeTrue)
	})
}
