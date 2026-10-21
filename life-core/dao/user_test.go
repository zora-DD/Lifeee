package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func userRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "wx_open_id", "nickname", "status", "role", "created_at", "updated_at"}).
		AddRow(71, "openid-71", "小树", model.UserStatusApproved, model.UserRoleUser, now, now)
}

func TestUserDaoCreateAndGet(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	Convey("用户可创建并回填主键", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("INSERT INTO .*users.*").WillReturnResult(sqlmock.NewResult(71, 1))
		row := &model.User{WxOpenID: "openid-71", Nickname: "小树"}

		err := NewUserDao().Create(ctx, db, row)

		So(err, ShouldBeNil)
		So(row.ID, ShouldEqual, uint64(71))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("用户可按 ID、OpenID 和 UnionID 查询", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT .* FROM .*users.*id = .*LIMIT.*").WithArgs(uint64(71), 1).WillReturnRows(userRows(now))
		mock.ExpectQuery("SELECT .* FROM .*users.*wx_open_id.*LIMIT.*").WithArgs("openid-71", 1).WillReturnRows(userRows(now))
		mock.ExpectQuery("SELECT .* FROM .*users.*wx_union_id.*LIMIT.*").WithArgs("union-71", 1).WillReturnRows(userRows(now))
		d := NewUserDao()

		byID, err := d.GetByID(ctx, db, 71)
		So(err, ShouldBeNil)
		byOpenID, err := d.GetByOpenID(ctx, db, "openid-71")
		So(err, ShouldBeNil)
		byUnionID, err := d.GetByUnionID(ctx, db, "union-71")
		So(err, ShouldBeNil)
		So(byID.ID, ShouldEqual, byOpenID.ID)
		So(byOpenID.ID, ShouldEqual, byUnionID.ID)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})

	Convey("用户总数查询返回正确结果", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*users.*").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := NewUserDao().Count(ctx, db)

		So(err, ShouldBeNil)
		So(count, ShouldEqual, int64(5))
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestUserDaoUpdates(t *testing.T) {
	Convey("用户资料、状态、角色和登录时间独立更新", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectExec("UPDATE .*users.* SET .*nickname.* WHERE id = .*").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*users.* SET .*status.*updated_at.* WHERE id = .*").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*users.* SET .*role.*updated_at.* WHERE id = .*").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE .*users.* SET .*last_login_at.*updated_at.* WHERE id = .*").WillReturnResult(sqlmock.NewResult(0, 1))
		d := NewUserDao()
		ctx := context.Background()
		now := time.Now()

		So(d.UpdateProfile(ctx, db, &model.User{ID: 71, Nickname: "新昵称"}), ShouldBeNil)
		So(d.UpdateStatus(ctx, db, 71, model.UserStatusApproved), ShouldBeNil)
		So(d.UpdateRole(ctx, db, 71, model.UserRoleAdmin), ShouldBeNil)
		So(d.UpdateLastLoginAt(ctx, db, 71, now), ShouldBeNil)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}

func TestUserDaoList(t *testing.T) {
	Convey("用户列表应用状态、角色和昵称筛选", t, func() {
		db, mock := newDAOTestDB(t)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*users.*status.*role.*nickname LIKE.*").
			WithArgs(model.UserStatusPending, model.UserRoleUser, "%树%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT .* FROM .*users.* ORDER BY id DESC LIMIT .*").
			WithArgs(model.UserStatusPending, model.UserRoleUser, "%树%", 20).
			WillReturnRows(userRows(time.Now()))

		rows, total, err := NewUserDao().List(context.Background(), db, UserListQuery{
			Status: model.UserStatusPending, Role: model.UserRoleUser, Keyword: "树", Page: 1, PageSize: 20,
		})

		So(err, ShouldBeNil)
		So(total, ShouldEqual, int64(1))
		So(rows, ShouldHaveLength, 1)
		So(mock.ExpectationsWereMet(), ShouldBeNil)
	})
}
