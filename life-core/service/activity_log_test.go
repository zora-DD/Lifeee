package service

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type activityLogRepositoryStub struct {
	rows      []dao.ActivityLogWithCreator
	statsRows []model.ActivityLog
	total     int64
	query     dao.ActivityLogListQuery
	created   *model.ActivityLog
	updated   *model.ActivityLog
	deletedID uint64
}

func (s *activityLogRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.ActivityLog) error {
	row.ID = 21
	copyRow := *row
	s.created = &copyRow
	s.rows = append(s.rows, dao.ActivityLogWithCreator{
		ActivityLog: copyRow, CreatorNickname: "晚风收藏家", CreatorEmoji: "🌿", CreatorColor: "#759B89",
	})
	return nil
}

func (s *activityLogRepositoryStub) GetByID(_ context.Context, _ *gorm.DB, id uint64) (*dao.ActivityLogWithCreator, error) {
	for i := range s.rows {
		if s.rows[i].ID == id {
			row := s.rows[i]
			return &row, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *activityLogRepositoryStub) Update(_ context.Context, _ *gorm.DB, row *model.ActivityLog) error {
	copyRow := *row
	s.updated = &copyRow
	for i := range s.rows {
		if s.rows[i].ID == row.ID {
			s.rows[i].ActivityLog = copyRow
		}
	}
	return nil
}

func (s *activityLogRepositoryStub) SoftDelete(_ context.Context, _ *gorm.DB, id uint64) error {
	s.deletedID = id
	return nil
}

func (s *activityLogRepositoryStub) List(_ context.Context, _ *gorm.DB, in dao.ActivityLogListQuery) ([]dao.ActivityLogWithCreator, int64, error) {
	s.query = in
	return s.rows, s.total, nil
}

func (s *activityLogRepositoryStub) ListForStats(_ context.Context, _ *gorm.DB, _ uint64, _, _ time.Time) ([]model.ActivityLog, error) {
	return s.statsRows, nil
}

func TestActivityLogList(t *testing.T) {
	Convey("生活记录列表返回稳定分页信息", t, func() {
		repository := &activityLogRepositoryStub{
			rows: []dao.ActivityLogWithCreator{{
				ActivityLog:     model.ActivityLog{ID: 2, Content: "收拾客厅", OccurredAt: time.Now(), InputType: model.ActivityInputTypeText, CreatedBy: 4},
				CreatorNickname: "云朵漫游者", CreatorEmoji: "☁️", CreatorColor: "#8AA6B8",
			}},
			total: 3,
		}
		svc := NewActivityLogService(new(gorm.DB), repository)
		req := &proto.ListActivityLogsRequest{Page: 1, PageSize: 1}

		So(svc.InitAndCheckListParam(context.Background(), req), ShouldBeNil)
		response, err := svc.List(context.Background(), req)

		So(err, ShouldBeNil)
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].Id, ShouldEqual, "2")
		So(response.Items[0].Creator.Nickname, ShouldEqual, "云朵漫游者")
		So(response.HasMore, ShouldBeTrue)
		So(repository.query.PageSize, ShouldEqual, 1)
	})

	Convey("非法输入类型和分页参数返回参数错误", t, func() {
		requests := []*proto.ListActivityLogsRequest{
			{InputType: "image"},
			{Page: -1},
			{PageSize: 101},
			{From: "bad-time"},
		}
		for _, request := range requests {
			err := NewActivityLogService(nil, nil).InitAndCheckListParam(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})
}

func TestActivityLogCreateUpdateDelete(t *testing.T) {
	Convey("创建生活记录使用登录用户并规范化字段", t, func() {
		repository := &activityLogRepositoryStub{}
		svc := NewActivityLogService(new(gorm.DB), repository)
		ctx := authctx.WithUser(context.Background(), &model.User{ID: 88})

		response, err := svc.Create(ctx, &proto.CreateActivityLogRequest{
			Content: "  收拾客厅并洗澡  ", OccurredAt: "2026-09-02T08:30:00+08:00",
			DurationMinutes: 45, InputType: model.ActivityInputTypeText,
		})

		So(err, ShouldBeNil)
		So(response.Id, ShouldEqual, "21")
		So(response.Content, ShouldEqual, "收拾客厅并洗澡")
		So(response.DurationMinutes, ShouldEqual, int32(45))
		So(response.Creator.Nickname, ShouldEqual, "晚风收藏家")
		So(repository.created, ShouldNotBeNil)
		So(repository.created.CreatedBy, ShouldEqual, uint64(88))
	})

	Convey("编辑生活记录会持久化并返回最新内容", t, func() {
		repository := &activityLogRepositoryStub{rows: []dao.ActivityLogWithCreator{{
			ActivityLog: model.ActivityLog{ID: 9, Content: "旧记录", OccurredAt: time.Date(2026, 9, 1, 8, 0, 0, 0, time.Local),
				InputType: model.ActivityInputTypeText, CreatedBy: 1},
			CreatorNickname: "生活梦想家",
		}}}
		svc := NewActivityLogService(new(gorm.DB), repository)

		response, err := svc.Update(adminContext(), &proto.UpdateActivityLogRequest{
			Id: "9", Content: "整理卧室", OccurredAt: "2026-09-02T09:00:00+08:00",
			DurationMinutes: 30, InputType: model.ActivityInputTypeVoice,
			AudioUrl: "https://example.test/audio/9.m4a",
		})

		So(err, ShouldBeNil)
		So(repository.updated, ShouldNotBeNil)
		So(response.Content, ShouldEqual, "整理卧室")
		So(response.InputType, ShouldEqual, model.ActivityInputTypeVoice)
		So(response.AudioUrl, ShouldEqual, "https://example.test/audio/9.m4a")
	})

	Convey("删除生活记录前校验记录存在", t, func() {
		repository := &activityLogRepositoryStub{rows: []dao.ActivityLogWithCreator{{ActivityLog: model.ActivityLog{ID: 7}}}}
		svc := NewActivityLogService(new(gorm.DB), repository)

		response, err := svc.Delete(adminContext(), &proto.DeleteActivityLogRequest{Id: "7"})

		So(err, ShouldBeNil)
		So(response.Deleted, ShouldBeTrue)
		So(repository.deletedID, ShouldEqual, uint64(7))
	})
}

func TestActivityLogWriteValidation(t *testing.T) {
	Convey("创建与编辑拒绝非法字段", t, func() {
		svc := NewActivityLogService(new(gorm.DB), &activityLogRepositoryStub{})
		requests := []*proto.CreateActivityLogRequest{
			{OccurredAt: "2026-09-02T08:30:00+08:00"},
			{Content: "测试", OccurredAt: "bad-time"},
			{Content: "测试", OccurredAt: "2026-09-02T08:30:00+08:00"},
			{Content: "测试", OccurredAt: "2026-09-02T08:30:00+08:00", DurationMinutes: -1},
			{Content: "测试", OccurredAt: "2026-09-02T08:30:00+08:00", DurationMinutes: 1441},
			{Content: "测试", OccurredAt: "2026-09-02T08:30:00+08:00", InputType: "image"},
		}
		for _, request := range requests {
			_, err := svc.Create(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}

		_, err := svc.Update(context.Background(), &proto.UpdateActivityLogRequest{Id: "0"})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		_, err = svc.Delete(context.Background(), &proto.DeleteActivityLogRequest{Id: "bad-id"})
		So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
	})
}

func TestActivityLogStats(t *testing.T) {
	Convey("月度统计汇总记录数、活跃天数和时长", t, func() {
		duration30 := uint16(30)
		duration60 := uint16(60)
		repository := &activityLogRepositoryStub{statsRows: []model.ActivityLog{
			{OccurredAt: time.Date(2026, 9, 2, 8, 0, 0, 0, time.Local), DurationMinutes: &duration30},
			{OccurredAt: time.Date(2026, 9, 2, 12, 0, 0, 0, time.Local)},
			{OccurredAt: time.Date(2026, 9, 3, 9, 0, 0, 0, time.Local), DurationMinutes: &duration60},
		}}
		svc := NewActivityLogService(new(gorm.DB), repository)

		response, err := svc.Stats(context.Background(), &proto.ActivityLogStatsRequest{
			Scope: "month", Year: 2026, Month: 9,
		})

		So(err, ShouldBeNil)
		So(response.RecordCount, ShouldEqual, int32(3))
		So(response.ActiveDays, ShouldEqual, int32(2))
		So(response.TotalMinutes, ShouldEqual, int32(90))
		So(response.DurationRecordCount, ShouldEqual, int32(2))
		So(response.AverageMinutes, ShouldEqual, int32(45))
		So(response.LongestMinutes, ShouldEqual, int32(60))
		So(response.DailyBreakdown, ShouldHaveLength, 2)
		So(response.MonthlyBreakdown, ShouldHaveLength, 12)
		So(response.MonthlyBreakdown[8].RecordCount, ShouldEqual, int32(3))
	})

	Convey("年度统计返回十二个月汇总且拒绝非法范围", t, func() {
		duration := uint16(25)
		repository := &activityLogRepositoryStub{statsRows: []model.ActivityLog{{
			OccurredAt: time.Date(2026, 2, 3, 9, 0, 0, 0, time.Local), DurationMinutes: &duration,
		}}}
		svc := NewActivityLogService(new(gorm.DB), repository)

		response, err := svc.Stats(context.Background(), &proto.ActivityLogStatsRequest{Scope: "year", Year: 2026})
		So(err, ShouldBeNil)
		So(response.MonthlyBreakdown, ShouldHaveLength, 12)
		So(response.MonthlyBreakdown[1].RecordCount, ShouldEqual, int32(1))

		invalid := []*proto.ActivityLogStatsRequest{
			{Scope: "week", Year: 2026},
			{Scope: "month", Year: 1999, Month: 9},
			{Scope: "month", Year: 2026, Month: 13},
		}
		for _, request := range invalid {
			_, err = svc.Stats(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})
}
