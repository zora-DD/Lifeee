package router

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	"life-core/auth"
	"life-core/model"
)

type middlewareUserRepository struct {
	user *model.User
}

type middlewarePermissionRepository struct {
	rows []model.UserModulePermission
}

func (r *middlewarePermissionRepository) ListByUserID(context.Context, *gorm.DB, uint64) ([]model.UserModulePermission, error) {
	return r.rows, nil
}

func (r *middlewareUserRepository) GetByID(context.Context, *gorm.DB, uint64) (*model.User, error) {
	if r.user == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.user, nil
}

func testAuthEngine(user *model.User, token string) (*httptest.ResponseRecorder, bool) {
	return testAuthEngineForPath(user, token, "/v1/life-core/task/list", []string{"/v1/life-core/task/*"})
}

func testAuthEngineForPath(user *model.User, token, path string, requiredPaths []string) (*httptest.ResponseRecorder, bool) {
	gin.SetMode(gin.TestMode)
	manager, _ := auth.NewTokenManager("unit-test-jwt-secret-with-at-least-32-characters", time.Hour)
	called := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Next()
		if ginErr := c.Errors.Last(); ginErr != nil {
			status, response := hecode.Failure("test", ginErr.Err)
			c.JSON(status, response)
		}
	})
	engine.Use(authMiddleware(
		&gorm.DB{}, &middlewareUserRepository{user: user}, &middlewarePermissionRepository{rows: []model.UserModulePermission{{
			UserID: 1, Module: model.PermissionModuleTask, Level: model.PermissionLevelWrite,
		}}}, manager, requiredPaths,
	))
	engine.POST(path, func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder, called
}

func TestAuthMiddleware(t *testing.T) {
	manager, _ := auth.NewTokenManager("unit-test-jwt-secret-with-at-least-32-characters", time.Hour)
	token, _, _ := manager.Issue(1)

	Convey("业务接口拒绝无 Token 请求", t, func() {
		recorder, called := testAuthEngine(nil, "")
		So(recorder.Code, ShouldEqual, http.StatusUnauthorized)
		So(called, ShouldBeFalse)
	})

	Convey("业务接口拒绝非法 access_token", t, func() {
		recorder, called := testAuthEngine(nil, "invalid-access-token")
		So(recorder.Code, ShouldEqual, http.StatusUnauthorized)
		So(called, ShouldBeFalse)
	})

	Convey("已通过用户可以访问业务接口", t, func() {
		user := &model.User{ID: 1, Status: model.UserStatusApproved, Role: model.UserRoleUser}
		recorder, called := testAuthEngine(user, token)
		So(recorder.Code, ShouldEqual, http.StatusNoContent)
		So(called, ShouldBeTrue)
	})

	Convey("待审核用户不能访问业务接口", t, func() {
		user := &model.User{ID: 1, Status: model.UserStatusPending, Role: model.UserRoleUser}
		recorder, called := testAuthEngine(user, token)
		So(recorder.Code, ShouldEqual, http.StatusForbidden)
		So(called, ShouldBeFalse)
	})

	Convey("待审核用户可以提交身份备注", t, func() {
		user := &model.User{ID: 1, Status: model.UserStatusPending, Role: model.UserRoleUser}
		recorder, called := testAuthEngineForPath(user, token, "/v1/life-core/access/apply", []string{"/v1/life-core/access/*"})
		So(recorder.Code, ShouldEqual, http.StatusNoContent)
		So(called, ShouldBeTrue)
	})
}

func TestPathRequiresAuth(t *testing.T) {
	Convey("鉴权路径支持精确匹配和模块前缀", t, func() {
		rules := []string{"/v1/life-core/user/current", "/v1/life-core/task/*", "/v1/life-core/favorite/*", "/v1/life-core/media/upload"}

		So(pathRequiresAuth("/v1/life-core/user/current", rules), ShouldBeTrue)
		So(pathRequiresAuth("/v1/life-core/task/list", rules), ShouldBeTrue)
		So(pathRequiresAuth("/v1/life-core/task/create", rules), ShouldBeTrue)
		So(pathRequiresAuth("/v1/life-core/favorite/list", rules), ShouldBeTrue)
		So(pathRequiresAuth("/v1/life-core/media/upload", rules), ShouldBeTrue)
		So(pathRequiresAuth("/v1/life-core/media/authorize", rules), ShouldBeFalse)
		So(pathRequiresAuth("/v1/life-core/health", rules), ShouldBeFalse)
		So(pathRequiresAuth("/v1/life-core/task-extra/list", rules), ShouldBeFalse)
	})
}

func TestModulePermissionRequirement(t *testing.T) {
	Convey("只读权限允许查询但不能写入", t, func() {
		rows := []model.UserModulePermission{{Module: model.PermissionModuleTask, Level: model.PermissionLevelRead}}
		readRule, ok := modulePermissionRequirement("/v1/life-core/task/list")
		So(ok, ShouldBeTrue)
		So(permissionRowsAllow(rows, readRule.modules, readRule.level), ShouldBeTrue)

		writeRule, ok := modulePermissionRequirement("/v1/life-core/task/create")
		So(ok, ShouldBeTrue)
		So(permissionRowsAllow(rows, writeRule.modules, writeRule.level), ShouldBeFalse)

		routineReadRule, ok := modulePermissionRequirement("/v1/life-core/task/routine/list")
		So(ok, ShouldBeTrue)
		So(permissionRowsAllow(rows, routineReadRule.modules, routineReadRule.level), ShouldBeTrue)
		routineWriteRule, ok := modulePermissionRequirement("/v1/life-core/task/routine/checkin")
		So(ok, ShouldBeTrue)
		So(permissionRowsAllow(rows, routineWriteRule.modules, routineWriteRule.level), ShouldBeFalse)
	})

	Convey("记录解析需要生活记录或账单写权限", t, func() {
		rule, ok := modulePermissionRequirement("/v1/life-core/record/voice/recognize")
		So(ok, ShouldBeTrue)
		So(permissionRowsAllow([]model.UserModulePermission{{Module: model.PermissionModuleBill, Level: model.PermissionLevelWrite}}, rule.modules, rule.level), ShouldBeTrue)
		So(permissionRowsAllow([]model.UserModulePermission{{Module: model.PermissionModuleBill, Level: model.PermissionLevelRead}}, rule.modules, rule.level), ShouldBeFalse)
	})

	Convey("聚合内容列表必须拥有全部相关模块读取权限", t, func() {
		rows := []model.UserModulePermission{
			{Module: model.PermissionModuleTask, Level: model.PermissionLevelRead},
			{Module: model.PermissionModuleRestaurant, Level: model.PermissionLevelWrite},
		}
		modules := []string{model.PermissionModuleTask, model.PermissionModuleRestaurant, model.PermissionModulePlace}
		So(permissionRowsAllowAll(rows, modules, model.PermissionLevelRead), ShouldBeFalse)
		rows = append(rows, model.UserModulePermission{Module: model.PermissionModulePlace, Level: model.PermissionLevelRead})
		So(permissionRowsAllowAll(rows, modules, model.PermissionLevelRead), ShouldBeTrue)
	})
}

func TestContentPermissionRequirement(t *testing.T) {
	Convey("收藏接口按目标类型校验对应模块且不会消费请求体", t, func() {
		gin.SetMode(gin.TestMode)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/life-core/favorite/modify", bytes.NewBufferString(`{"targetType":"restaurant","targetId":"8"}`))

		rule, requireAll, required, err := contentPermissionRequirement(ctx, ctx.Request.URL.Path)

		So(err, ShouldBeNil)
		So(required, ShouldBeTrue)
		So(requireAll, ShouldBeFalse)
		So(rule.modules, ShouldResemble, []string{model.PermissionModuleRestaurant})
		body, readErr := io.ReadAll(ctx.Request.Body)
		So(readErr, ShouldBeNil)
		So(string(body), ShouldContainSubstring, "restaurant")
	})

	Convey("未指定类型的贡献列表要求全部公开内容模块", t, func() {
		gin.SetMode(gin.TestMode)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/life-core/user/contribution/list", bytes.NewBufferString(`{}`))

		rule, requireAll, required, err := contentPermissionRequirement(ctx, ctx.Request.URL.Path)

		So(err, ShouldBeNil)
		So(required, ShouldBeTrue)
		So(requireAll, ShouldBeTrue)
		So(rule.modules, ShouldHaveLength, 3)
	})
}
