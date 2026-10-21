package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vaynedu/hollow/pkg/hecode"
	xdraw "golang.org/x/image/draw"
	"gorm.io/gorm"

	"life-core/config"
	"life-core/model"
)

const maxDecodedImagePixels = int64(40_000_000)

type mediaAssetRepository interface {
	Create(context.Context, *gorm.DB, *model.MediaAsset) error
	GetByObjectKey(context.Context, *gorm.DB, string) (*model.MediaAsset, error)
}

// MediaOptions 定义媒体服务运行参数，方便单元测试注入临时目录和时间。
type MediaOptions struct {
	StoragePath    string
	SigningSecret  string
	MaxUploadBytes int64
	MaxImageEdge   int
	JPEGQuality    int
	SignedURLTTL   time.Duration
	Now            func() time.Time
}

// MediaUploadResult 是上传成功后返回给小程序的媒体信息。
type MediaUploadResult struct {
	MediaID   string `json:"mediaId"`
	ObjectKey string `json:"objectKey"`
	URL       string `json:"url"`
	MimeType  string `json:"mimeType"`
	FileSize  uint64 `json:"fileSize"`
	Width     uint32 `json:"width"`
	Height    uint32 `json:"height"`
}

// MediaService 负责图片校验、压缩、落盘和短期访问签名。
type MediaService struct {
	db         *gorm.DB
	repository mediaAssetRepository
	options    MediaOptions
}

// NewConfiguredMediaService 使用当前配置创建媒体服务。
func NewConfiguredMediaService(db *gorm.DB, repository mediaAssetRepository) (*MediaService, error) {
	cfg := config.Get()
	if cfg == nil || !cfg.Media.Enabled {
		return nil, fmt.Errorf("%w: media service is disabled", hecode.ErrConfig)
	}
	return NewMediaService(db, repository, MediaOptions{
		StoragePath:    cfg.Media.StoragePath,
		SigningSecret:  cfg.MediaSigningSecret(),
		MaxUploadBytes: cfg.Media.MaxUploadBytes,
		MaxImageEdge:   cfg.Media.MaxImageEdge,
		JPEGQuality:    cfg.Media.JPEGQuality,
		SignedURLTTL:   cfg.Media.SignedURLTTL,
	})
}

// NewMediaService 创建可注入配置的媒体服务。
func NewMediaService(db *gorm.DB, repository mediaAssetRepository, options MediaOptions) (*MediaService, error) {
	options.StoragePath = strings.TrimSpace(options.StoragePath)
	options.SigningSecret = strings.TrimSpace(options.SigningSecret)
	if db == nil || repository == nil || options.StoragePath == "" || len(options.SigningSecret) < 32 ||
		options.MaxUploadBytes <= 0 || options.MaxImageEdge <= 0 || options.JPEGQuality < 1 ||
		options.JPEGQuality > 100 || options.SignedURLTTL <= 0 {
		return nil, fmt.Errorf("%w: invalid media service options", hecode.ErrConfig)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &MediaService{db: db, repository: repository, options: options}, nil
}

// UploadImage 校验并压缩一张业务图片。
func (s *MediaService) UploadImage(ctx context.Context, userID uint64, bizType string, reader io.Reader) (*MediaUploadResult, error) {
	if userID == 0 || reader == nil {
		return nil, fmt.Errorf("%w: invalid media upload", hecode.ErrInvalidParam)
	}
	bizType = strings.TrimSpace(bizType)
	if bizType != model.MediaBizTypeRestaurant && bizType != model.MediaBizTypePlace && bizType != model.MediaBizTypeMemory && bizType != model.MediaBizTypeHomeDish && bizType != model.MediaBizTypeSnack {
		return nil, fmt.Errorf("%w: unsupported media bizType", hecode.ErrInvalidParam)
	}

	raw, err := io.ReadAll(io.LimitReader(reader, s.options.MaxUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read image: %v", hecode.ErrService, err)
	}
	if int64(len(raw)) == 0 || int64(len(raw)) > s.options.MaxUploadBytes {
		return nil, fmt.Errorf("%w: image size exceeds limit", hecode.ErrInvalidParam)
	}
	mimeType := http.DetectContentType(raw)
	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return nil, fmt.Errorf("%w: only JPEG and PNG images are supported", hecode.ErrInvalidParam)
	}
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || imageConfig.Width <= 0 || imageConfig.Height <= 0 ||
		int64(imageConfig.Width)*int64(imageConfig.Height) > maxDecodedImagePixels {
		return nil, fmt.Errorf("%w: invalid image dimensions", hecode.ErrInvalidParam)
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: decode image", hecode.ErrInvalidParam)
	}

	width, height := fittedImageSize(imageConfig.Width, imageConfig.Height, s.options.MaxImageEdge)
	processed := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(processed, processed.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(processed, processed.Bounds(), source, source.Bounds(), xdraw.Over, nil)

	now := s.options.Now()
	objectKey := fmt.Sprintf("%s/%04d/%02d/%s.jpg", bizType, now.Year(), int(now.Month()), uuid.NewString())
	fullPath, err := s.fullPath(objectKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return nil, fmt.Errorf("%w: create media directory: %v", hecode.ErrService, err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(fullPath), ".upload-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("%w: create temporary image: %v", hecode.ErrService, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := jpeg.Encode(temporary, processed, &jpeg.Options{Quality: s.options.JPEGQuality}); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("%w: encode image: %v", hecode.ErrService, err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("%w: close image: %v", hecode.ErrService, err)
	}
	if err := os.Chmod(temporaryPath, 0o640); err != nil {
		return nil, fmt.Errorf("%w: protect image: %v", hecode.ErrService, err)
	}
	if err := os.Rename(temporaryPath, fullPath); err != nil {
		return nil, fmt.Errorf("%w: save image: %v", hecode.ErrService, err)
	}
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("%w: stat image: %v", hecode.ErrService, err)
	}

	widthValue, heightValue := uint32(width), uint32(height)
	row := &model.MediaAsset{
		UserID: userID, BizType: bizType, ObjectKey: objectKey, FileURL: "",
		MimeType: "image/jpeg", FileSize: uint64(fileInfo.Size()),
		Width: &widthValue, Height: &heightValue, Status: model.MediaStatusActive,
	}
	if err := s.repository.Create(ctx, s.db, row); err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("%w: create media asset: %v", hecode.ErrDBInsert, err)
	}
	return &MediaUploadResult{
		MediaID: strconv.FormatUint(row.ID, 10), ObjectKey: objectKey,
		URL: s.SignedURL(objectKey), MimeType: row.MimeType, FileSize: row.FileSize,
		Width: widthValue, Height: heightValue,
	}, nil
}

// AuthorizeDownload 校验短期签名，并确认媒体仍处于可用状态。
func (s *MediaService) AuthorizeDownload(ctx context.Context, objectKey, expires, signature string) (*model.MediaAsset, error) {
	if err := validateObjectKey(objectKey); err != nil || !s.verify(objectKey, expires, signature) {
		return nil, hecode.ErrForbidden
	}
	row, err := s.repository.GetByObjectKey(ctx, s.db, objectKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, hecode.ErrNotFound
		}
		return nil, fmt.Errorf("%w: get media asset: %v", hecode.ErrDBQuery, err)
	}
	if row.Status != model.MediaStatusActive {
		return nil, hecode.ErrNotFound
	}
	return row, nil
}

// SignedURL 返回不包含域名的短期签名地址，由前端按当前环境补齐域名。
func (s *MediaService) SignedURL(objectKey string) string {
	return signedMediaURL(objectKey, s.options.SigningSecret, s.options.SignedURLTTL, s.options.Now())
}

func (s *MediaService) verify(objectKey, expires, signature string) bool {
	expiresAt, err := strconv.ParseInt(strings.TrimSpace(expires), 10, 64)
	if err != nil || expiresAt < s.options.Now().Unix() || strings.TrimSpace(signature) == "" {
		return false
	}
	expected := s.signature(objectKey, expires)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (s *MediaService) signature(objectKey, expires string) string {
	mac := hmac.New(sha256.New, []byte(s.options.SigningSecret))
	_, _ = io.WriteString(mac, objectKey+"\n"+expires)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *MediaService) fullPath(objectKey string) (string, error) {
	if err := validateObjectKey(objectKey); err != nil {
		return "", err
	}
	root, err := filepath.Abs(s.options.StoragePath)
	if err != nil {
		return "", fmt.Errorf("%w: resolve media storage: %v", hecode.ErrConfig, err)
	}
	fullPath := filepath.Join(root, filepath.FromSlash(objectKey))
	if !strings.HasPrefix(fullPath, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: invalid media object key", hecode.ErrInvalidParam)
	}
	return fullPath, nil
}

func validateObjectKey(objectKey string) error {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" || strings.HasPrefix(objectKey, "/") || strings.Contains(objectKey, "\\") ||
		strings.Contains(objectKey, "..") || filepath.ToSlash(filepath.Clean(objectKey)) != objectKey {
		return fmt.Errorf("%w: invalid media object key", hecode.ErrInvalidParam)
	}
	if !strings.HasPrefix(objectKey, model.MediaBizTypeRestaurant+"/") &&
		!strings.HasPrefix(objectKey, model.MediaBizTypePlace+"/") &&
		!strings.HasPrefix(objectKey, model.MediaBizTypeMemory+"/") &&
		!strings.HasPrefix(objectKey, model.MediaBizTypeHomeDish+"/") &&
		!strings.HasPrefix(objectKey, model.MediaBizTypeSnack+"/") {
		return fmt.Errorf("%w: unsupported media object key", hecode.ErrInvalidParam)
	}
	return nil
}

func fittedImageSize(width, height, maxEdge int) (int, int) {
	if width <= maxEdge && height <= maxEdge {
		return width, height
	}
	if width >= height {
		return maxEdge, max(1, height*maxEdge/width)
	}
	return max(1, width*maxEdge/height), maxEdge
}

func configuredMediaURL(objectKey, legacyURL string) string {
	if strings.TrimSpace(objectKey) == "" {
		return strings.TrimSpace(legacyURL)
	}
	cfg := config.Get()
	if cfg == nil || !cfg.Media.Enabled {
		return ""
	}
	return signedMediaURL(objectKey, cfg.MediaSigningSecret(), cfg.Media.SignedURLTTL, time.Now())
}

func signedMediaURL(objectKey, secret string, ttl time.Duration, now time.Time) string {
	if validateObjectKey(objectKey) != nil || len(strings.TrimSpace(secret)) < 32 || ttl <= 0 {
		return ""
	}
	// 过期时间按小时对齐，使同一图片在缓存窗口内获得稳定 URL，便于客户端命中缓存。
	expires := strconv.FormatInt(now.Truncate(time.Hour).Add(ttl).Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, objectKey+"\n"+expires)
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return "/media/" + objectKey + "?expires=" + expires + "&signature=" + url.QueryEscape(signature)
}
