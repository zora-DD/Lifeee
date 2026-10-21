package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestBillDaoCRUD(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.Local)

	Convey("账单可创建并回填主键", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*bills.*").WillReturnResult(sqlmock.NewResult(31, 1))
		row := &model.Bill{Type: model.BillTypeExpense, Title: "午饭", Amount: "28.00", Category: "餐饮", OccurredAt: now, CreatedBy: 2}

		err := NewBillDao().Create(ctx, db, row)

		So(err, ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(31))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("账单详情携带记账人信息", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT bills\\.\\*,.*creator_nickname.*FROM .*bills.*LEFT JOIN users.*bills\\.id.*deleted_at.*IS NULL.*LIMIT").
			WithArgs(uint64(31), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "type", "title", "amount", "category", "occurred_at", "created_by", "creator_nickname"}).
				AddRow(31, "expense", "午饭", "28.00", "餐饮", now, 2, "小林"))

		row, err := NewBillDao().GetByID(ctx, db, 31)

		So(err, ShouldBeNil)
		So(row.Title, ShouldEqual, "午饭")
		So(row.CreatorNickname, ShouldEqual, "小林")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("账单更新限制字段且删除为软删除", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*bills.* SET .*type.*title.*amount.*category.*occurred_at.* WHERE id = .*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*bills.* SET .*deleted_at.* WHERE .*id.*deleted_at.*IS NULL").
			WillReturnResult(sqlmock.NewResult(0, 1))
		d := NewBillDao()

		So(d.Update(ctx, db, &model.Bill{ID: 31, Type: "expense", Title: "晚饭", Amount: "35.00", Category: "餐饮", OccurredAt: now}), ShouldBeNil)
		So(d.SoftDelete(ctx, db, 31), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestBillDaoQueries(t *testing.T) {
	Convey("列表支持记账人和分类筛选", t, func() {
		db, mock := newDAOTestDB(t)
		from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		to := from.AddDate(0, 1, 0)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*bills.*type.*category.*created_by.*occurred_at.*").
			WithArgs("expense", "餐饮", "%饭%", "%饭%", uint64(2), from, to).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT bills\\.\\*,.*FROM .*bills.*LEFT JOIN users.*ORDER BY bills\\.occurred_at DESC,bills\\.id DESC LIMIT").
			WithArgs("expense", "餐饮", "%饭%", "%饭%", uint64(2), from, to, 20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "amount", "occurred_at", "created_by"}).AddRow(31, "午饭", "28.00", from, 2))

		rows, total, err := NewBillDao().List(context.Background(), db, BillListQuery{
			Type: "expense", Category: "餐饮", Keyword: "饭", CreatedBy: 2, From: &from, To: &to, Page: 1, PageSize: 20,
		})

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("记账人和分类汇总查询可执行", t, func() {
		db, mock := newDAOTestDB(t)
		from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		to := from.AddDate(0, 1, 0)
		mock.ExpectQuery("SELECT users\\.id.*FROM .*bills.*JOIN users.*deleted_at.*GROUP BY.*ORDER BY").
			WillReturnRows(sqlmock.NewRows([]string{"id", "nickname", "avatar_emoji", "avatar_color"}).AddRow(2, "小林", "🌿", "#789b8b"))
		mock.ExpectQuery("SELECT type, category, CAST\\(SUM\\(amount\\) AS CHAR\\) AS amount, COUNT\\(\\*\\) AS count FROM .*bills.*occurred_at.*created_by.*GROUP BY type, category").
			WithArgs(from, to, uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"type", "category", "amount", "count"}).AddRow("expense", "餐饮", "28.00", 1))
		d := NewBillDao()

		creators, err := d.ListCreators(context.Background(), db)
		So(err, ShouldBeNil)
		So(creators, ShouldHaveLength, 1)
		rows, err := d.Aggregate(context.Background(), db, 2, from, to)
		So(err, ShouldBeNil)
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("年度月度汇总可按记账人筛选", t, func() {
		db, mock := newDAOTestDB(t)
		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
		to := from.AddDate(1, 0, 0)
		mock.ExpectQuery("SELECT MONTH\\(occurred_at\\) AS month, type, CAST\\(SUM\\(amount\\) AS CHAR\\) AS amount, COUNT\\(\\*\\) AS count FROM .*bills.*occurred_at.*created_by.*GROUP BY MONTH\\(occurred_at\\), type.*ORDER BY month ASC, type ASC").
			WithArgs(from, to, uint64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"month", "type", "amount", "count"}).AddRow(8, "expense", "1080.00", 7))

		rows, err := NewBillDao().AggregateMonthly(context.Background(), db, 2, from, to)

		So(err, ShouldBeNil)
		So(rows, ShouldHaveLength, 1)
		So(rows[0].Month, ShouldEqual, 8)
		So(rows[0].Amount, ShouldEqual, "1080.00")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("年度用户汇总携带用户展示信息", t, func() {
		db, mock := newDAOTestDB(t)
		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
		to := from.AddDate(1, 0, 0)
		mock.ExpectQuery("SELECT bills\\.created_by,.*FROM .*bills.*LEFT JOIN users.*deleted_at.*occurred_at.*GROUP BY bills\\.created_by.*ORDER BY SUM\\(bills\\.amount\\) DESC").
			WithArgs(from, to).
			WillReturnRows(sqlmock.NewRows([]string{"created_by", "nickname", "avatar_emoji", "avatar_color", "type", "amount", "count"}).
				AddRow(2, "小林", "🌿", "#789b8b", "expense", "1080.00", 7))

		rows, err := NewBillDao().AggregateUsers(context.Background(), db, from, to)

		So(err, ShouldBeNil)
		So(rows, ShouldHaveLength, 1)
		So(rows[0].CreatedBy, ShouldEqual, uint64(2))
		So(rows[0].Nickname, ShouldEqual, "小林")
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
