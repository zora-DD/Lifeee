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

type billRepositoryStub struct {
	rows       []dao.BillWithCreator
	creators   []dao.BillCreator
	aggregates []dao.BillCategoryAggregate
	monthly    []dao.BillMonthlyAggregate
	users      []dao.BillUserAggregate
	query      dao.BillListQuery
	created    *model.Bill
	updated    *model.Bill
	deletedID  uint64
}

func (s *billRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.Bill) error {
	row.ID = 41
	copyRow := *row
	s.created = &copyRow
	s.rows = append(s.rows, dao.BillWithCreator{Bill: copyRow, CreatorNickname: "小林"})
	return nil
}
func (s *billRepositoryStub) GetByID(_ context.Context, _ *gorm.DB, id uint64) (*dao.BillWithCreator, error) {
	for i := range s.rows {
		if s.rows[i].ID == id {
			row := s.rows[i]
			return &row, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (s *billRepositoryStub) Update(_ context.Context, _ *gorm.DB, row *model.Bill) error {
	copyRow := *row
	s.updated = &copyRow
	for i := range s.rows {
		if s.rows[i].ID == row.ID {
			s.rows[i].Bill = copyRow
		}
	}
	return nil
}
func (s *billRepositoryStub) SoftDelete(_ context.Context, _ *gorm.DB, id uint64) error {
	s.deletedID = id
	return nil
}
func (s *billRepositoryStub) List(_ context.Context, _ *gorm.DB, query dao.BillListQuery) ([]dao.BillWithCreator, int64, error) {
	s.query = query
	return s.rows, int64(len(s.rows)), nil
}
func (s *billRepositoryStub) ListCreators(_ context.Context, _ *gorm.DB) ([]dao.BillCreator, error) {
	return s.creators, nil
}
func (s *billRepositoryStub) Aggregate(_ context.Context, _ *gorm.DB, _ uint64, _, _ time.Time) ([]dao.BillCategoryAggregate, error) {
	return s.aggregates, nil
}
func (s *billRepositoryStub) AggregateMonthly(_ context.Context, _ *gorm.DB, _ uint64, _, _ time.Time) ([]dao.BillMonthlyAggregate, error) {
	return s.monthly, nil
}
func (s *billRepositoryStub) AggregateUsers(_ context.Context, _ *gorm.DB, _, _ time.Time) ([]dao.BillUserAggregate, error) {
	return s.users, nil
}

func TestBillListAndCRUD(t *testing.T) {
	Convey("公开列表返回记账人筛选项和分页信息", t, func() {
		repository := &billRepositoryStub{
			rows:     []dao.BillWithCreator{{Bill: model.Bill{ID: 3, Type: "expense", Title: "午饭", Amount: "28.00", Category: "餐饮", OccurredAt: time.Now(), CreatedBy: 8}, CreatorNickname: "小林"}},
			creators: []dao.BillCreator{{ID: 8, Nickname: "小林"}},
		}
		svc := NewBillService(new(gorm.DB), repository)
		req := &proto.ListBillsRequest{CreatedBy: 8, Page: 1, PageSize: 1}

		So(svc.InitAndCheckListParam(context.Background(), req), ShouldBeNil)
		response, err := svc.List(context.Background(), req)

		So(err, ShouldBeNil)
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].Creator.Nickname, ShouldEqual, "小林")
		So(response.Creators, ShouldHaveLength, 1)
		So(repository.query.CreatedBy, ShouldEqual, uint64(8))
	})

	Convey("创建账单使用登录用户并规范化金额", t, func() {
		repository := &billRepositoryStub{}
		svc := NewBillService(new(gorm.DB), repository)
		ctx := authctx.WithUser(context.Background(), &model.User{ID: 88})

		response, err := svc.Create(ctx, &proto.CreateBillRequest{
			Type: "expense", Title: " 午饭 ", Amount: "28.5", Category: "餐饮",
			OccurredAt: "2026-09-05T12:00:00+08:00", InputType: "text", RawText: "午饭28.5",
		})

		So(err, ShouldBeNil)
		So(response.Amount, ShouldEqual, "28.50")
		So(repository.created.CreatedBy, ShouldEqual, uint64(88))
	})

	Convey("编辑和删除账单调用仓储层", t, func() {
		now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.Local)
		repository := &billRepositoryStub{rows: []dao.BillWithCreator{{Bill: model.Bill{ID: 9, Title: "午饭", Amount: "28.00", OccurredAt: now}}}}
		svc := NewBillService(new(gorm.DB), repository)

		updated, err := svc.Update(adminContext(), &proto.UpdateBillRequest{Id: "9", Type: "expense", Title: "晚饭", Amount: "36", Category: "餐饮", OccurredAt: "2026-09-05T18:00:00+08:00"})
		So(err, ShouldBeNil)
		So(updated.Title, ShouldEqual, "晚饭")
		So(repository.updated.Amount, ShouldEqual, "36.00")

		deleted, err := svc.Delete(adminContext(), &proto.DeleteBillRequest{Id: "9"})
		So(err, ShouldBeNil)
		So(deleted.Deleted, ShouldBeTrue)
		So(repository.deletedID, ShouldEqual, uint64(9))
	})
}

func TestBillValidationAndStats(t *testing.T) {
	Convey("历史账单分类可以用于列表筛选", t, func() {
		for _, category := range []string{"日常采购", "智能设备", "家居用品", "转账"} {
			svc := NewBillService(new(gorm.DB), &billRepositoryStub{})
			err := svc.InitAndCheckListParam(context.Background(), &proto.ListBillsRequest{Category: category})
			So(err, ShouldBeNil)
		}
	})

	Convey("非法筛选和账单字段会返回参数错误", t, func() {
		svc := NewBillService(new(gorm.DB), &billRepositoryStub{})
		So(errors.Is(svc.InitAndCheckListParam(context.Background(), &proto.ListBillsRequest{Category: "未知"}), hecode.ErrInvalidParam), ShouldBeTrue)
		invalid := []*proto.CreateBillRequest{
			{Type: "expense", Title: "午饭", Amount: "0", Category: "餐饮", OccurredAt: "2026-09-05T12:00:00+08:00"},
			{Type: "expense", Title: "午饭", Amount: "12.345", Category: "餐饮", OccurredAt: "2026-09-05T12:00:00+08:00"},
			{Type: "expense", Title: "午饭", Amount: "12", Category: "未知", OccurredAt: "2026-09-05T12:00:00+08:00"},
		}
		for _, request := range invalid {
			_, err := svc.Create(context.Background(), request)
			So(errors.Is(err, hecode.ErrInvalidParam), ShouldBeTrue)
		}
	})

	Convey("月度汇总区分支出、收入和分类", t, func() {
		repository := &billRepositoryStub{
			aggregates: []dao.BillCategoryAggregate{
				{Type: "expense", Category: "餐饮", Amount: "128.50", Count: 3},
				{Type: "expense", Category: "交通", Amount: "16.00", Count: 1},
				{Type: "income", Category: "收入", Amount: "500.00", Count: 1},
			},
			users: []dao.BillUserAggregate{{CreatedBy: 8, Nickname: "小林", Type: "expense", Amount: "144.50", Count: 4}},
		}
		response, err := NewBillService(new(gorm.DB), repository).Stats(context.Background(), &proto.BillStatsRequest{Year: 2026, Month: 9})

		So(err, ShouldBeNil)
		So(response.ExpenseTotal, ShouldEqual, "144.50")
		So(response.IncomeTotal, ShouldEqual, "500.00")
		So(response.BillCount, ShouldEqual, int32(5))
		So(response.Categories, ShouldHaveLength, 2)
		So(response.Users, ShouldHaveLength, 1)
		So(response.Users[0].ExpenseAmount, ShouldEqual, "144.50")
	})

	Convey("年度分析返回十二个月和用户支出对比", t, func() {
		repository := &billRepositoryStub{
			aggregates: []dao.BillCategoryAggregate{{Type: "expense", Category: "餐饮", Amount: "80.00", Count: 1}},
			monthly:    []dao.BillMonthlyAggregate{{Month: 8, Type: "expense", Amount: "80.00", Count: 1}},
			users: []dao.BillUserAggregate{
				{CreatedBy: 8, Nickname: "小林", AvatarEmoji: "🌿", Type: "expense", Amount: "80.00", Count: 1},
				{CreatedBy: 8, Nickname: "小林", AvatarEmoji: "🌿", Type: "income", Amount: "20.00", Count: 2},
			},
		}
		response, err := NewBillService(new(gorm.DB), repository).Stats(context.Background(), &proto.BillStatsRequest{Scope: "year", Year: 2026})

		So(err, ShouldBeNil)
		So(response.Scope, ShouldEqual, "year")
		So(response.Monthly, ShouldHaveLength, 12)
		So(response.Monthly[7].ExpenseAmount, ShouldEqual, "80.00")
		So(response.Users, ShouldHaveLength, 1)
		So(response.Users[0].User.Nickname, ShouldEqual, "小林")
		So(response.Users[0].ExpenseAmount, ShouldEqual, "80.00")
		So(response.Users[0].IncomeAmount, ShouldEqual, "20.00")
		So(response.Users[0].Count, ShouldEqual, int32(1))
	})
}
