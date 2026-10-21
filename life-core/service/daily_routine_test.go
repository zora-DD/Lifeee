package service

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"

	"life-core/model"
	"life-core/proto"
)

type dailyRoutineRepositoryStub struct {
	routines     []model.DailyRoutine
	checkins     []model.DailyCheckin
	created      *model.DailyRoutine
	updated      *model.DailyRoutine
	upserted     *model.DailyCheckin
	includeOff   bool
	requestedUID uint64
}

func (s *dailyRoutineRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.DailyRoutine) error {
	row.ID = 9
	s.created = row
	s.routines = append(s.routines, *row)
	return nil
}

func (s *dailyRoutineRepositoryStub) GetByIDAndUser(_ context.Context, _ *gorm.DB, id, userID uint64) (*model.DailyRoutine, error) {
	for i := range s.routines {
		if s.routines[i].ID == id && s.routines[i].CreatedBy == userID {
			return &s.routines[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *dailyRoutineRepositoryStub) ListByUser(_ context.Context, _ *gorm.DB, userID uint64, includeDisabled bool) ([]model.DailyRoutine, error) {
	s.requestedUID = userID
	s.includeOff = includeDisabled
	return s.routines, nil
}

func (s *dailyRoutineRepositoryStub) Update(_ context.Context, _ *gorm.DB, row *model.DailyRoutine) error {
	s.updated = row
	for i := range s.routines {
		if s.routines[i].ID == row.ID && s.routines[i].CreatedBy == row.CreatedBy {
			s.routines[i] = *row
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (s *dailyRoutineRepositoryStub) SoftDelete(_ context.Context, _ *gorm.DB, _, _ uint64) error {
	return nil
}

func (s *dailyRoutineRepositoryStub) ListCheckins(_ context.Context, _ *gorm.DB, _ uint64, _ []uint64, _, _ time.Time) ([]model.DailyCheckin, error) {
	return s.checkins, nil
}

func (s *dailyRoutineRepositoryStub) UpsertCheckin(_ context.Context, _ *gorm.DB, row *model.DailyCheckin) error {
	s.upserted = row
	return nil
}

func TestDailyRoutineList(t *testing.T) {
	Convey("习惯打卡返回今天状态、连续天数和月度记录", t, func() {
		checkedAt := time.Date(2026, 9, 20, 8, 5, 0, 0, time.Local)
		repository := &dailyRoutineRepositoryStub{
			routines: []model.DailyRoutine{{ID: 9, Title: "抹药", TimeOfDay: "08:00", Enabled: true, CreatedBy: 1, CreatedAt: time.Date(2026, 9, 14, 8, 0, 0, 0, time.Local)}},
			checkins: []model.DailyCheckin{
				{RoutineID: 9, CheckinDate: time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local), Completed: true, CheckedAt: &checkedAt},
				{RoutineID: 9, CheckinDate: time.Date(2026, 9, 19, 0, 0, 0, 0, time.Local), Completed: true, CheckedAt: &checkedAt},
				{RoutineID: 9, CheckinDate: time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local), Completed: true, CheckedAt: &checkedAt},
			},
		}
		svc := NewDailyRoutineService(new(gorm.DB), repository)

		response, err := svc.List(context.Background(), &proto.ListDailyRoutinesRequest{Date: "2026-09-20"})

		So(err, ShouldBeNil)
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].CheckedToday, ShouldBeTrue)
		So(response.Items[0].History, ShouldHaveLength, 7)
		So(response.Items[0].History[6].Date, ShouldEqual, "2026-09-20")
		So(response.Items[0].History[6].Completed, ShouldBeTrue)
		So(response.Items[0].CurrentStreak, ShouldEqual, int32(3))
		So(response.Items[0].TotalCheckins, ShouldEqual, int32(3))
		So(response.Items[0].LastSevenDayRate, ShouldEqual, int32(42))
		So(response.Items[0].MonthHistory, ShouldHaveLength, 30)
	})
}

func TestDailyRoutineMutation(t *testing.T) {
	Convey("创建每日事项会清洗名称并默认启用", t, func() {
		repository := new(dailyRoutineRepositoryStub)
		svc := NewDailyRoutineService(new(gorm.DB), repository)

		item, err := svc.Create(context.Background(), &proto.CreateDailyRoutineRequest{Title: " 抹药 ", TimeOfDay: "08:05"})

		So(err, ShouldBeNil)
		So(item.Id, ShouldEqual, "9")
		So(repository.created.Title, ShouldEqual, "抹药")
		So(repository.created.Enabled, ShouldBeTrue)
	})

	Convey("完成和撤销共用同一天的幂等打卡记录", t, func() {
		repository := &dailyRoutineRepositoryStub{routines: []model.DailyRoutine{{ID: 9, Title: "吃药", TimeOfDay: "21:00", Enabled: true, CreatedBy: 1}}}
		svc := NewDailyRoutineService(new(gorm.DB), repository)
		svc.now = func() time.Time { return time.Date(2026, 9, 20, 21, 3, 0, 0, time.Local) }

		item, err := svc.Checkin(context.Background(), &proto.CheckinDailyRoutineRequest{Id: "9", Date: "2026-09-20", Completed: true})

		So(err, ShouldBeNil)
		So(repository.upserted.Completed, ShouldBeTrue)
		So(repository.upserted.CheckedAt, ShouldNotBeNil)
		So(item.CheckedToday, ShouldBeTrue)

		_, err = svc.Checkin(context.Background(), &proto.CheckinDailyRoutineRequest{Id: "9", Date: "2026-09-20", Completed: false})
		So(err, ShouldBeNil)
		So(repository.upserted.Completed, ShouldBeFalse)
		So(repository.upserted.CheckedAt, ShouldBeNil)
	})

	Convey("非法时间和日期会返回参数错误", t, func() {
		repository := new(dailyRoutineRepositoryStub)
		svc := NewDailyRoutineService(new(gorm.DB), repository)
		_, err := svc.Create(context.Background(), &proto.CreateDailyRoutineRequest{Title: "吃药", TimeOfDay: "25:00"})
		So(err, ShouldNotBeNil)
		_, err = svc.List(context.Background(), &proto.ListDailyRoutinesRequest{Date: "今天"})
		So(err, ShouldNotBeNil)
	})
}

func TestDailyRoutineSchedule(t *testing.T) {
	Convey("每两天按开始日期计算计划日", t, func() {
		start := time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local)
		routine := &model.DailyRoutine{RepeatUnit: "day", RepeatInterval: 2, StartDate: &start}

		So(isRoutineScheduled(routine, time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local)), ShouldBeTrue)
		So(isRoutineScheduled(routine, time.Date(2026, 9, 19, 0, 0, 0, 0, time.Local)), ShouldBeFalse)
		So(isRoutineScheduled(routine, time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local)), ShouldBeTrue)
	})

	Convey("每周固定在开始日期对应的星期", t, func() {
		start := time.Date(2026, 9, 17, 0, 0, 0, 0, time.Local)
		routine := &model.DailyRoutine{RepeatUnit: "week", RepeatInterval: 1, StartDate: &start}

		So(isRoutineScheduled(routine, time.Date(2026, 9, 24, 0, 0, 0, 0, time.Local)), ShouldBeTrue)
		So(isRoutineScheduled(routine, time.Date(2026, 9, 25, 0, 0, 0, 0, time.Local)), ShouldBeFalse)
	})

	Convey("每月 31 日在短月份按月末执行", t, func() {
		start := time.Date(2026, 1, 31, 0, 0, 0, 0, time.Local)
		routine := &model.DailyRoutine{RepeatUnit: "month", RepeatInterval: 1, StartDate: &start}

		So(isRoutineScheduled(routine, time.Date(2026, 2, 28, 0, 0, 0, 0, time.Local)), ShouldBeTrue)
		So(isRoutineScheduled(routine, time.Date(2026, 3, 31, 0, 0, 0, 0, time.Local)), ShouldBeTrue)
		So(isRoutineScheduled(routine, time.Date(2026, 3, 30, 0, 0, 0, 0, time.Local)), ShouldBeFalse)
	})

	Convey("非计划日不能打卡", t, func() {
		start := time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local)
		repository := &dailyRoutineRepositoryStub{routines: []model.DailyRoutine{{
			ID: 9, Title: "抹药", TimeOfDay: "08:00", Enabled: true, CreatedBy: 1,
			RepeatUnit: "day", RepeatInterval: 2, StartDate: &start,
		}}}
		svc := NewDailyRoutineService(new(gorm.DB), repository)

		_, err := svc.Checkin(context.Background(), &proto.CheckinDailyRoutineRequest{
			Id: "9", Date: "2026-09-19", Completed: true,
		})

		So(err, ShouldNotBeNil)
		So(repository.upserted, ShouldBeNil)
	})
}
