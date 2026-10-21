package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	"life-core/auth"
	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/model"
)

const authorizationPrefix = "Bearer "

type authUserRepository interface {
	GetByID(context.Context, *gorm.DB, uint64) (*model.User, error)
}

type authPermissionRepository interface {
	ListByUserID(context.Context, *gorm.DB, uint64) ([]model.UserModulePermission, error)
}

// configuredAuthMiddleware 在请求阶段读取配置，因为 Hollow 的 Startup 晚于路由注册执行。
func configuredAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.Get()
		if cfg == nil {
			abortWithError(c, fmt.Errorf("%w: config is not initialized", hecode.ErrConfig))
			return
		}
		if !cfg.Auth.Enabled || !pathRequiresAuth(c.Request.URL.Path, cfg.Auth.RequiredPaths) {
			c.Next()
			return
		}
		tokens, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
		if err != nil {
			abortWithError(c, fmt.Errorf("%w: initialize token manager: %v", hecode.ErrConfig, err))
			return
		}
		authMiddleware(database.MySQL(), dao.NewUserDao(), dao.NewUserModulePermissionDao(), tokens, cfg.Auth.RequiredPaths)(c)
	}
}

// authMiddleware 校验配置命中的接口 Token，并实时检查数据库中的账号状态。
func authMiddleware(
	db *gorm.DB,
	users authUserRepository,
	permissions authPermissionRepository,
	tokens *auth.TokenManager,
	requiredPaths []string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !pathRequiresAuth(path, requiredPaths) {
			c.Next()
			return
		}
		authorization := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(authorization, authorizationPrefix) {
			abortWithError(c, hecode.ErrUnauthorized)
			return
		}
		accessToken := strings.TrimSpace(strings.TrimPrefix(authorization, authorizationPrefix))
		userID, err := tokens.Parse(accessToken)
		if err != nil {
			abortWithError(c, hecode.ErrUnauthorized)
			return
		}
		user, err := users.GetByID(c.Request.Context(), db, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			abortWithError(c, hecode.ErrUnauthorized)
			return
		}
		if err != nil {
			abortWithError(c, fmt.Errorf("%w: load current user: %v", hecode.ErrDBQuery, err))
			return
		}
		if user.Status == model.UserStatusDisabled {
			abortWithError(c, hecode.ErrForbidden)
			return
		}
		requestContext := auth.WithUser(c.Request.Context(), user)
		c.Request = c.Request.WithContext(requestContext)
		if path == "/v1/life-core/user/current" ||
			path == "/v1/life-core/access/status" ||
			path == "/v1/life-core/access/apply" {
			c.Next()
			return
		}
		if user.Status != model.UserStatusApproved {
			abortWithError(c, hecode.ErrForbidden)
			return
		}
		if strings.HasPrefix(path, "/v1/life-core/admin/") && user.Role != model.UserRoleAdmin {
			abortWithError(c, hecode.ErrPermissionDenied)
			return
		}
		if user.Role != model.UserRoleAdmin {
			requirement, required := modulePermissionRequirement(path)
			requireAll := false
			if !required {
				var requirementErr error
				requirement, requireAll, required, requirementErr = contentPermissionRequirement(c, path)
				if requirementErr != nil {
					abortWithError(c, fmt.Errorf("%w: read permission target: %v", hecode.ErrInvalidParam, requirementErr))
					return
				}
			}
			if required {
				rows, permissionErr := permissions.ListByUserID(c.Request.Context(), db, user.ID)
				if permissionErr != nil {
					abortWithError(c, fmt.Errorf("%w: load module permissions: %v", hecode.ErrDBQuery, permissionErr))
					return
				}
				allowed := permissionRowsAllow(rows, requirement.modules, requirement.level)
				if requireAll {
					allowed = permissionRowsAllowAll(rows, requirement.modules, requirement.level)
				}
				if !allowed {
					abortWithError(c, hecode.ErrForbidden)
					return
				}
			}
		}
		c.Next()
	}
}

// contentPermissionRequirement 为收藏和贡献接口按目标内容类型补充权限校验。
// 列表未指定类型时要求三个内容模块均可读，避免聚合结果泄露被隐藏的模块。
func contentPermissionRequirement(c *gin.Context, path string) (modulePermissionRule, bool, bool, error) {
	listPath := path == "/v1/life-core/user/contribution/list" || path == "/v1/life-core/favorite/list"
	if !listPath && path != "/v1/life-core/favorite/status" && path != "/v1/life-core/favorite/modify" {
		return modulePermissionRule{}, false, false, nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return modulePermissionRule{}, false, false, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var payload struct {
		TargetType      string `json:"targetType"`
		TargetTypeSnake string `json:"target_type"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			return modulePermissionRule{}, false, false, err
		}
	}
	targetType := strings.TrimSpace(payload.TargetType)
	if targetType == "" {
		targetType = strings.TrimSpace(payload.TargetTypeSnake)
	}
	modules := []string{
		model.PermissionModuleTask,
		model.PermissionModuleRestaurant,
		model.PermissionModulePlace,
	}
	if targetType == "" {
		if listPath {
			return modulePermissionRule{modules: modules, level: model.PermissionLevelRead}, true, true, nil
		}
		return modulePermissionRule{}, false, false, nil
	}
	moduleByTarget := map[string]string{
		model.FavoriteTargetTask:       model.PermissionModuleTask,
		model.FavoriteTargetRestaurant: model.PermissionModuleRestaurant,
		model.FavoriteTargetPlace:      model.PermissionModulePlace,
	}
	module, ok := moduleByTarget[targetType]
	if !ok {
		return modulePermissionRule{}, false, false, nil
	}
	return modulePermissionRule{modules: []string{module}, level: model.PermissionLevelRead}, false, true, nil
}

type modulePermissionRule struct {
	modules []string
	level   string
}

func modulePermissionRequirement(path string) (modulePermissionRule, bool) {
	readPaths := map[string]struct{}{
		"/v1/life-core/task/list": {}, "/v1/life-core/task/detail": {}, "/v1/life-core/task/ai_summary": {}, "/v1/life-core/task/routine/list": {},
		"/v1/life-core/activity_log/list": {}, "/v1/life-core/activity_log/detail": {}, "/v1/life-core/activity_log/stats": {},
		"/v1/life-core/bill/list": {}, "/v1/life-core/bill/detail": {}, "/v1/life-core/bill/stats": {},
		"/v1/life-core/restaurant/list": {}, "/v1/life-core/restaurant/detail": {}, "/v1/life-core/restaurant/recommend": {},
		"/v1/life-core/restaurant/consumption/list": {},
		"/v1/life-core/restaurant/home/list":        {}, "/v1/life-core/restaurant/home/detail": {}, "/v1/life-core/restaurant/home/recommend": {},
		"/v1/life-core/place/list": {}, "/v1/life-core/place/detail": {},
	}
	module := ""
	switch {
	case strings.HasPrefix(path, "/v1/life-core/task/"):
		module = model.PermissionModuleTask
	case strings.HasPrefix(path, "/v1/life-core/activity_log/"):
		module = model.PermissionModuleActivity
	case strings.HasPrefix(path, "/v1/life-core/bill/"):
		module = model.PermissionModuleBill
	case strings.HasPrefix(path, "/v1/life-core/restaurant/"):
		module = model.PermissionModuleRestaurant
	case strings.HasPrefix(path, "/v1/life-core/place/"):
		module = model.PermissionModulePlace
	case strings.HasPrefix(path, "/v1/life-core/record/"):
		return modulePermissionRule{modules: []string{model.PermissionModuleActivity, model.PermissionModuleBill}, level: model.PermissionLevelWrite}, true
	case strings.HasPrefix(path, "/v1/life-core/report/"):
		return modulePermissionRule{modules: []string{model.PermissionModuleActivity}, level: model.PermissionLevelRead}, true
	default:
		return modulePermissionRule{}, false
	}
	level := model.PermissionLevelWrite
	if _, readOnly := readPaths[path]; readOnly {
		level = model.PermissionLevelRead
	}
	return modulePermissionRule{modules: []string{module}, level: level}, true
}

func permissionRowsAllow(rows []model.UserModulePermission, modules []string, requiredLevel string) bool {
	for _, row := range rows {
		for _, module := range modules {
			if row.Module == module && permissionLevelAllows(row.Level, requiredLevel) {
				return true
			}
		}
	}
	return false
}

func permissionRowsAllowAll(rows []model.UserModulePermission, modules []string, requiredLevel string) bool {
	for _, module := range modules {
		if !permissionRowsAllow(rows, []string{module}, requiredLevel) {
			return false
		}
	}
	return true
}

func permissionLevelAllows(actual, required string) bool {
	if actual == model.PermissionLevelWrite {
		return true
	}
	return required == model.PermissionLevelRead && actual == model.PermissionLevelRead
}

// pathRequiresAuth 支持精确路径和以 * 结尾的前缀规则。
func pathRequiresAuth(path string, requiredPaths []string) bool {
	for _, rule := range requiredPaths {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if strings.HasSuffix(rule, "*") {
			if strings.HasPrefix(path, strings.TrimSuffix(rule, "*")) {
				return true
			}
			continue
		}
		if path == rule {
			return true
		}
	}
	return false
}

func abortWithError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}
