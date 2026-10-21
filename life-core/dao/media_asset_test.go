package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestMediaAssetDao(t *testing.T) {
	ctx := context.Background()

	Convey("媒体资产可创建、查询和更新状态", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*media_assets.*").WillReturnResult(sqlmock.NewResult(31, 1))
		row := &model.MediaAsset{UserID: 3, BizType: model.MediaBizTypePlace, ObjectKey: "place/cover.jpg"}
		So(NewMediaAssetDao().Create(ctx, db, row), ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(31))

		assetRows := func() *sqlmock.Rows {
			return sqlmock.NewRows([]string{"id", "user_id", "biz_type", "object_key", "status"}).
				AddRow(31, 3, model.MediaBizTypePlace, "place/cover.jpg", model.MediaStatusActive)
		}
		mock.ExpectQuery("SELECT .* FROM .*media_assets.*id = .*LIMIT.*").WithArgs(uint64(31), 1).WillReturnRows(assetRows())
		mock.ExpectQuery("SELECT .* FROM .*media_assets.*object_key.*LIMIT.*").WithArgs("place/cover.jpg", 1).WillReturnRows(assetRows())
		d := NewMediaAssetDao()
		byID, err := d.GetByID(ctx, db, 31)
		So(err, ShouldBeNil)
		byKey, err := d.GetByObjectKey(ctx, db, "place/cover.jpg")
		So(err, ShouldBeNil)
		So(byID.ID, ShouldEqual, byKey.ID)

		mock.ExpectExec("UPDATE .*media_assets.* SET .*status.*updated_at.* WHERE id = .*").
			WillReturnResult(sqlmock.NewResult(0, 1))
		So(d.UpdateStatus(ctx, db, 31, model.MediaStatusDeleted), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("媒体列表应用用户、业务类型和状态筛选", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*media_assets.*user_id.*biz_type.*status.*").
			WithArgs(uint64(3), model.MediaBizTypePlace, model.MediaStatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT .* FROM .*media_assets.* ORDER BY created_at DESC,id DESC LIMIT .*").
			WithArgs(uint64(3), model.MediaBizTypePlace, model.MediaStatusActive, 20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "object_key", "created_at"}).AddRow(31, "place/cover.jpg", time.Now()))

		rows, total, err := NewMediaAssetDao().List(ctx, db, MediaAssetListQuery{
			UserID: 3, BizType: model.MediaBizTypePlace, Status: model.MediaStatusActive, Page: 1, PageSize: 20,
		})

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
