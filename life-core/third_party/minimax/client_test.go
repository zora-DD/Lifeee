package minimax

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"life-core/config"
)

func TestMiniMaxOrganize(t *testing.T) {
	Convey("调用兼容接口并解析 JSON 草稿", t, func() {
		apiKey := make(chan string, 1)
		anthropicVersion := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey <- r.Header.Get("x-api-key")
			anthropicVersion <- r.Header.Get("anthropic-version")
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"content":[{"type":"thinking","thinking":"整理中"},{"type":"text","text":"`+
				"```json\\n{\\\"content\\\":\\\"收拾客厅并洗澡\\\",\\\"occurred_at\\\":\\\"2026-09-03T08:00:00+08:00\\\",\\\"duration_minutes\\\":30}\\n```"+
				`"}]}`)
		}))
		defer server.Close()
		client, err := NewMiniMaxClient(config.MiniMaxConfig{
			Endpoint: server.URL, APIKey: "test-key", Model: "test-model", Timeout: time.Second,
		})
		So(err, ShouldBeNil)

		draft, err := client.Organize(context.Background(), "我收拾客厅和洗澡，半小时", time.Now())

		So(err, ShouldBeNil)
		So(<-apiKey, ShouldEqual, "test-key")
		So(<-anthropicVersion, ShouldEqual, "2023-06-01")
		So(draft.Content, ShouldEqual, "收拾客厅并洗澡")
		So(draft.DurationMinutes, ShouldEqual, 30)
	})

	Convey("拒绝无效的模型响应", t, func() {
		_, err := parseActivityDraft("not-json", time.Now())
		So(err, ShouldNotBeNil)

		_, err = parseActivityDraft(`{"content":"记录","occurred_at":"wrong","duration_minutes":0}`, time.Now())
		So(err, ShouldNotBeNil)
	})
}

func TestMiniMaxOrganizeTask(t *testing.T) {
	Convey("调用兼容接口并解析任务草稿", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"content":[{"type":"text","text":"`+
				`{\"title\":\"购买零食\",\"description\":\"顺便买香蕉\",\"category\":\"生活\"}`+
				`"}]}`)
		}))
		defer server.Close()
		client, err := NewMiniMaxClient(config.MiniMaxConfig{
			Endpoint: server.URL, APIKey: "test-key", Model: "test-model", Timeout: time.Second,
		})
		So(err, ShouldBeNil)

		draft, err := client.OrganizeTask(context.Background(), "记得买零食，顺便买香蕉", time.Now())

		So(err, ShouldBeNil)
		So(draft.Title, ShouldEqual, "购买零食")
		So(draft.Description, ShouldEqual, "顺便买香蕉")
		So(draft.Category, ShouldEqual, "生活")
	})

	Convey("未知分类回退为其他", t, func() {
		draft, err := parseTaskDraft(`{"title":"办理证件","description":"","category":"办事"}`)

		So(err, ShouldBeNil)
		So(draft.Category, ShouldEqual, "其他")
	})
}

func TestMiniMaxOrganizeQuickRecord(t *testing.T) {
	Convey("一句话可拆分为多笔账单", t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"content":[{"type":"text","text":"`+
				`{\"intent\":\"bill\",\"activity\":null,\"bills\":[{\"type\":\"expense\",\"title\":\"午饭\",\"amount\":\"28\",\"category\":\"餐饮\",\"occurred_at\":\"2026-09-04T12:00:00+08:00\",\"note\":\"\"},{\"type\":\"expense\",\"title\":\"打车\",\"amount\":\"16\",\"category\":\"交通\",\"occurred_at\":\"2026-09-04T13:00:00+08:00\",\"note\":\"\"}]}`+
				`"}]}`)
		}))
		defer server.Close()
		client, err := NewMiniMaxClient(config.MiniMaxConfig{Endpoint: server.URL, APIKey: "test-key", Model: "test-model", Timeout: time.Second})
		So(err, ShouldBeNil)

		draft, err := client.OrganizeQuickRecord(context.Background(), "昨天午饭28，打车16", time.Now())

		So(err, ShouldBeNil)
		So(draft.Intent, ShouldEqual, "bill")
		So(draft.Bills, ShouldHaveLength, 2)
		So(draft.Bills[0].Amount, ShouldEqual, "28.00")
	})

	Convey("生活事项返回生活记录草稿", t, func() {
		draft, err := parseQuickRecordDraft(`{"intent":"activity_log","activity":{"content":"收拾客厅","occurred_at":"2026-09-05T09:00:00+08:00","duration_minutes":30},"bills":[]}`, time.Now())

		So(err, ShouldBeNil)
		So(draft.Activity.Content, ShouldEqual, "收拾客厅")
		So(draft.Bills, ShouldBeEmpty)
	})
}

func TestParseSummaryDraft(t *testing.T) {
	Convey("解析总结时清理空项并限制数量", t, func() {
		draft, err := parseSummaryDraft("```json\n{\"summary\":\"  今天先处理逾期任务  \",\"highlights\":[\" 完成记录 \"],\"suggestions\":[\" 第一项 \",\"\",\"第二项\",\"第三项\",\"第四项\"]}\n```", true)

		So(err, ShouldBeNil)
		So(draft.Summary, ShouldEqual, "今天先处理逾期任务")
		So(draft.Highlights, ShouldResemble, []string{"完成记录"})
		So(draft.Suggestions, ShouldResemble, []string{"第一项", "第二项", "第三项"})
	})

	Convey("任务总结忽略亮点字段", t, func() {
		draft, err := parseSummaryDraft(`{"summary":"先做一件重要的事","highlights":["不应返回"],"suggestions":[]}`, false)

		So(err, ShouldBeNil)
		So(draft.Highlights, ShouldBeNil)
	})

	Convey("拒绝无 JSON 或空总结", t, func() {
		_, err := parseSummaryDraft("not-json", true)
		So(err, ShouldNotBeNil)

		_, err = parseSummaryDraft(`{"summary":"   "}`, true)
		So(err, ShouldNotBeNil)
	})
}
