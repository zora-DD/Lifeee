package service

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type memoryRepositoryStub struct {
	rows  []model.Memory
	total int64
	years []int
	query dao.MemoryListQuery
}

func (s *memoryRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.Memory) error {
	row.ID = 1
	return nil
}

func (s *memoryRepositoryStub) List(_ context.Context, _ *gorm.DB, query dao.MemoryListQuery) ([]model.Memory, int64, error) {
	s.query = query
	return s.rows, s.total, nil
}

func (s *memoryRepositoryStub) ListYears(_ context.Context, _ *gorm.DB) ([]int, error) {
	return s.years, nil
}

type memoryPhotoRepositoryStub struct {
	rows []model.MemoryPhoto
}

func (s *memoryPhotoRepositoryStub) CreateBatch(_ context.Context, _ *gorm.DB, _ []model.MemoryPhoto) error {
	return nil
}

func (s *memoryPhotoRepositoryStub) ListByMemoryIDs(_ context.Context, _ *gorm.DB, _ []uint64) ([]model.MemoryPhoto, error) {
	return s.rows, nil
}

func TestMemoryList(t *testing.T) {
	Convey("管理员可以按年份分页查看时光和签名图片", t, func() {
		date := time.Date(2026, 9, 6, 0, 0, 0, 0, time.Local)
		memories := &memoryRepositoryStub{
			rows:  []model.Memory{{ID: 7, MemoryDate: date, DatePrecision: model.MemoryDatePrecisionDay, Title: "周末", CreatedBy: 1}},
			total: 3,
			years: []int{2026, 2025},
		}
		photos := &memoryPhotoRepositoryStub{rows: []model.MemoryPhoto{{
			ID: 8, MemoryID: 7, MediaAssetID: 9, ObjectKey: "memory/2026/09/a.jpg", Width: 1200, Height: 900,
		}}}
		svc := NewMemoryService(new(gorm.DB), memories, photos)

		response, err := svc.List(adminContext(), &proto.ListMemoriesRequest{Year: 2026, Page: 1, PageSize: 2})

		So(err, ShouldBeNil)
		So(memories.query.Year, ShouldEqual, 2026)
		So(response.Items, ShouldHaveLength, 1)
		So(response.Items[0].Date, ShouldEqual, "2026-09-06")
		So(response.Items[0].Images, ShouldHaveLength, 1)
		So(response.HasMore, ShouldBeTrue)
		So(response.Years, ShouldResemble, []int32{2026, 2025})
	})

	Convey("普通用户不能查看私人时光相册", t, func() {
		user := &model.User{ID: 2, Status: model.UserStatusApproved, Role: model.UserRoleUser}
		_, err := NewMemoryService(new(gorm.DB), new(memoryRepositoryStub), new(memoryPhotoRepositoryStub)).
			List(authctx.WithUser(context.Background(), user), &proto.ListMemoriesRequest{})
		So(err, ShouldNotBeNil)
	})
}

func TestParseMemoryDate(t *testing.T) {
	Convey("完整日期、月份和年份都能保留原始精度", t, func() {
		day, precision, err := parseMemoryDate("2026-09-06")
		So(err, ShouldBeNil)
		So(precision, ShouldEqual, model.MemoryDatePrecisionDay)
		So(formatMemoryDate(day, precision), ShouldEqual, "2026-09-06")

		month, precision, err := parseMemoryDate("2013-05")
		So(err, ShouldBeNil)
		So(formatMemoryDate(month, precision), ShouldEqual, "2013-05")

		year, precision, err := parseMemoryDate("2012")
		So(err, ShouldBeNil)
		So(formatMemoryDate(year, precision), ShouldEqual, "2012")
	})
}
