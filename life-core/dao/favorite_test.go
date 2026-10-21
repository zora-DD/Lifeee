package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestFavoriteDaoCRUD(t *testing.T) {
	ctx := context.Background()

	Convey("收藏可创建并回填主键", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*favorites.*").WillReturnResult(sqlmock.NewResult(21, 1))
		row := &model.Favorite{UserID: 3, TargetType: model.FavoriteTargetTask, TargetID: 7}

		err := NewFavoriteDao().Create(ctx, db, row)

		So(err, ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(21))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("可按主键和用户目标查询", t, func() {
		db, mock := newDAOTestDB(t)
		rows := func() *sqlmock.Rows {
			return sqlmock.NewRows([]string{"id", "user_id", "target_type", "target_id"}).
				AddRow(21, 3, model.FavoriteTargetTask, 7)
		}
		mock.ExpectQuery("SELECT .* FROM .*favorites.*id = .*LIMIT.*").WithArgs(uint64(21), 1).WillReturnRows(rows())
		mock.ExpectQuery("SELECT .* FROM .*favorites.*user_id.*target_type.*target_id.*LIMIT.*").
			WithArgs(uint64(3), model.FavoriteTargetTask, uint64(7), 1).WillReturnRows(rows())
		d := NewFavoriteDao()

		byID, err := d.GetByID(ctx, db, 21)
		So(err, ShouldBeNil)
		byTarget, err := d.GetByUserTarget(ctx, db, 3, model.FavoriteTargetTask, 7)
		So(err, ShouldBeNil)
		So(byID.ID, ShouldEqual, byTarget.ID)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("可按用户目标或业务目标删除", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("DELETE FROM .*favorites.*user_id.*target_type.*target_id.*").
			WithArgs(uint64(3), model.FavoriteTargetTask, uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM .*favorites.*target_type.*target_id.*").
			WithArgs(model.FavoriteTargetTask, uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
		d := NewFavoriteDao()

		So(d.DeleteByUserTarget(ctx, db, 3, model.FavoriteTargetTask, 7), ShouldBeNil)
		So(d.DeleteByTarget(ctx, db, model.FavoriteTargetTask, 7), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestFavoriteDaoListByUser(t *testing.T) {
	Convey("用户收藏按类型筛选并稳定分页", t, func() {
		db, mock := newDAOTestDB(t)
		now := time.Now()
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*favorites.*user_id.*target_type.*").
			WithArgs(uint64(3), model.FavoriteTargetPlace).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT .* FROM .*favorites.* ORDER BY created_at DESC,id DESC LIMIT .* OFFSET .*").
			WithArgs(uint64(3), model.FavoriteTargetPlace, 5, 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "target_type", "target_id", "created_at"}).
				AddRow(22, 3, model.FavoriteTargetPlace, 9, now))

		rows, total, err := NewFavoriteDao().ListByUser(context.Background(), db, 3, model.FavoriteTargetPlace, 2, 5)

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
