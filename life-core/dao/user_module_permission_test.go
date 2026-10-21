package dao

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"life-core/model"
)

func TestUserModulePermissionDao(t *testing.T) {
	db, mock := newDAOTestDB(t)
	ctx := context.Background()
	mock.ExpectQuery("SELECT .* FROM .*user_module_permissions.*user_id.*ORDER BY id ASC").
		WithArgs(uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "module", "permission_level", "granted_by"}).
			AddRow(1, 7, model.PermissionModuleTask, model.PermissionLevelNone, 1).
			AddRow(2, 7, model.PermissionModulePlace, model.PermissionLevelRead, 1))
	rows, err := NewUserModulePermissionDao().ListByUserID(ctx, db, 7)
	if err != nil || len(rows) != 2 || rows[1].Level != model.PermissionLevelRead {
		t.Fatalf("ListByUserID() rows=%+v err=%v", rows, err)
	}

	mock.ExpectExec("INSERT INTO .*user_module_permissions.*ON DUPLICATE KEY UPDATE.*").
		WillReturnResult(sqlmock.NewResult(0, 2))
	err = NewUserModulePermissionDao().UpsertAll(ctx, db, []model.UserModulePermission{
		{UserID: 7, Module: model.PermissionModuleTask, Level: model.PermissionLevelNone, GrantedBy: 1},
		{UserID: 7, Module: model.PermissionModulePlace, Level: model.PermissionLevelWrite, GrantedBy: 1},
	})
	if err != nil {
		t.Fatalf("UpsertAll() err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
