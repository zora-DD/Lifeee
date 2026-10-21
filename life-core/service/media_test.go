package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"

	"life-core/model"
)

type mediaRepositoryStub struct {
	created *model.MediaAsset
}

func (r *mediaRepositoryStub) Create(_ context.Context, _ *gorm.DB, row *model.MediaAsset) error {
	row.ID = 9
	r.created = row
	return nil
}

func (r *mediaRepositoryStub) GetByObjectKey(_ context.Context, _ *gorm.DB, objectKey string) (*model.MediaAsset, error) {
	if r.created == nil || r.created.ObjectKey != objectKey {
		return nil, gorm.ErrRecordNotFound
	}
	return r.created, nil
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestMediaService(t *testing.T) {
	Convey("图片上传后压缩落盘并只保存对象键", t, func() {
		now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.Local)
		repository := new(mediaRepositoryStub)
		storagePath := t.TempDir()
		svc, err := NewMediaService(&gorm.DB{}, repository, MediaOptions{
			StoragePath: storagePath, SigningSecret: strings.Repeat("s", 32),
			MaxUploadBytes: 5 << 20, MaxImageEdge: 1200, JPEGQuality: 82,
			SignedURLTTL: 2 * time.Hour, Now: func() time.Time { return now },
		})
		So(err, ShouldBeNil)

		result, err := svc.UploadImage(context.Background(), 3, model.MediaBizTypeRestaurant, bytes.NewReader(testPNG(t, 1600, 800)))

		So(err, ShouldBeNil)
		So(result.MediaID, ShouldEqual, "9")
		So(result.Width, ShouldEqual, uint32(1200))
		So(result.Height, ShouldEqual, uint32(600))
		So(result.ObjectKey, ShouldStartWith, "restaurant/2026/09/")
		So(result.URL, ShouldContainSubstring, "/media/restaurant/2026/09/")
		So(repository.created.FileURL, ShouldEqual, "")
		_, statErr := os.Stat(filepath.Join(storagePath, filepath.FromSlash(result.ObjectKey)))
		So(statErr, ShouldBeNil)

		parsed := strings.Split(result.URL, "?")
		query, queryErr := url.ParseQuery(parsed[1])
		So(queryErr, ShouldBeNil)
		asset, authErr := svc.AuthorizeDownload(context.Background(), result.ObjectKey, query.Get("expires"), query.Get("signature"))
		So(authErr, ShouldBeNil)
		So(asset.ID, ShouldEqual, uint64(9))

		snackImage, snackErr := svc.UploadImage(context.Background(), 3, model.MediaBizTypeSnack, bytes.NewReader(testPNG(t, 80, 80)))
		So(snackErr, ShouldBeNil)
		So(snackImage.ObjectKey, ShouldStartWith, "snack/2026/09/")
	})

	Convey("篡改签名或上传非图片会被拒绝", t, func() {
		repository := new(mediaRepositoryStub)
		svc, err := NewMediaService(&gorm.DB{}, repository, MediaOptions{
			StoragePath: t.TempDir(), SigningSecret: strings.Repeat("s", 32),
			MaxUploadBytes: 1024, MaxImageEdge: 1200, JPEGQuality: 82,
			SignedURLTTL: time.Hour,
		})
		So(err, ShouldBeNil)
		_, err = svc.UploadImage(context.Background(), 3, model.MediaBizTypePlace, strings.NewReader("not an image"))
		So(err, ShouldNotBeNil)
		_, err = svc.AuthorizeDownload(context.Background(), "place/2026/09/a.jpg", "9999999999", "bad-signature")
		So(err, ShouldNotBeNil)
	})

	Convey("过期的图片地址会被拒绝", t, func() {
		now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.Local)
		repository := &mediaRepositoryStub{created: &model.MediaAsset{
			ID: 9, ObjectKey: "place/2026/09/a.jpg", Status: model.MediaStatusActive,
		}}
		svc, err := NewMediaService(&gorm.DB{}, repository, MediaOptions{
			StoragePath: t.TempDir(), SigningSecret: strings.Repeat("s", 32),
			MaxUploadBytes: 1024, MaxImageEdge: 1200, JPEGQuality: 82,
			SignedURLTTL: time.Hour, Now: func() time.Time { return now },
		})
		So(err, ShouldBeNil)

		signedURL := svc.SignedURL(repository.created.ObjectKey)
		parsedURL, parseErr := url.Parse(signedURL)
		So(parseErr, ShouldBeNil)
		svc.options.Now = func() time.Time { return now.Add(2 * time.Hour) }

		_, err = svc.AuthorizeDownload(
			context.Background(),
			repository.created.ObjectKey,
			parsedURL.Query().Get("expires"),
			parsedURL.Query().Get("signature"),
		)
		So(err, ShouldNotBeNil)
	})

	Convey("同一小时内图片地址稳定，跨小时后自动轮换", t, func() {
		secret := strings.Repeat("s", 32)
		objectKey := "restaurant/2026/09/a.jpg"
		first := signedMediaURL(objectKey, secret, 2*time.Hour, time.Date(2026, 9, 15, 10, 1, 0, 0, time.Local))
		sameHour := signedMediaURL(objectKey, secret, 2*time.Hour, time.Date(2026, 9, 15, 10, 59, 0, 0, time.Local))
		nextHour := signedMediaURL(objectKey, secret, 2*time.Hour, time.Date(2026, 9, 15, 11, 0, 0, 0, time.Local))

		So(sameHour, ShouldEqual, first)
		So(nextHour, ShouldNotEqual, first)
	})
}
