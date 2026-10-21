package control

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vaynedu/hollow/pkg/hecode"

	"life-core/auth"
	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/service"
)

// UploadMedia 接收吃喝、家常菜或地点封面，图片内容不会写入访问日志。
func UploadMedia(c *gin.Context) {
	cfg := config.Get()
	if cfg == nil || !cfg.Media.Enabled {
		abortMediaRequest(c, fmt.Errorf("%w: media service is disabled", hecode.ErrConfig))
		return
	}
	userID, ok := auth.UserIDFromContext(c.Request.Context())
	if !ok {
		abortMediaRequest(c, hecode.ErrUnauthorized)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.Media.MaxUploadBytes+1024*1024)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		abortMediaRequest(c, fmt.Errorf("%w: invalid upload file", hecode.ErrInvalidParam))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		abortMediaRequest(c, fmt.Errorf("%w: open upload file", hecode.ErrInvalidParam))
		return
	}
	defer file.Close()

	svc, err := newMediaService()
	if err != nil {
		abortMediaRequest(c, err)
		return
	}
	result, err := svc.UploadImage(c.Request.Context(), userID, c.PostForm("bizType"), file)
	if err != nil {
		abortMediaRequest(c, err)
		return
	}
	c.Set("data", result)
}

// AuthorizeMedia 校验Nginx auth_request转发来的短期图片签名。
func AuthorizeMedia(c *gin.Context) {
	objectKey, expires, signature, err := parseMediaAuthorization(c)
	if err != nil {
		abortMediaRequest(c, err)
		return
	}
	svc, err := newMediaService()
	if err != nil {
		abortMediaRequest(c, err)
		return
	}
	asset, err := svc.AuthorizeDownload(c.Request.Context(), objectKey, expires, signature)
	if err != nil {
		abortMediaRequest(c, err)
		return
	}
	c.Header("X-Media-Object-Key", asset.ObjectKey)
	c.Set("data", gin.H{"allowed": true})
}

func newMediaService() (*service.MediaService, error) {
	return service.NewConfiguredMediaService(database.MySQL(), dao.NewMediaAssetDao())
}

func parseMediaAuthorization(c *gin.Context) (string, string, string, error) {
	rawURI := strings.TrimSpace(c.GetHeader("X-Original-URI"))
	if rawURI == "" {
		key := strings.TrimSpace(c.Query("key"))
		if key == "" {
			return "", "", "", fmt.Errorf("%w: missing media path", hecode.ErrInvalidParam)
		}
		return key, c.Query("expires"), c.Query("signature"), nil
	}
	parsed, err := url.ParseRequestURI(rawURI)
	if err != nil || !strings.HasPrefix(parsed.Path, "/media/") {
		return "", "", "", fmt.Errorf("%w: invalid media path", hecode.ErrInvalidParam)
	}
	objectKey, err := url.PathUnescape(strings.TrimPrefix(parsed.Path, "/media/"))
	if err != nil {
		return "", "", "", fmt.Errorf("%w: invalid media path", hecode.ErrInvalidParam)
	}
	return objectKey, parsed.Query().Get("expires"), parsed.Query().Get("signature"), nil
}

func abortMediaRequest(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}
