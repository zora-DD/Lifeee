package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/migration"
	"life-core/model"
	"life-core/proto"
	"life-core/service"
)

type importPhoto struct {
	File    string `json:"file"`
	AltText string `json:"altText"`
}

type importMemory struct {
	SourceKey   string        `json:"sourceKey"`
	Date        string        `json:"date"`
	Place       string        `json:"place"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Photos      []importPhoto `json:"photos"`
}

func main() {
	manifestPath := flag.String("manifest", "", "导入清单 JSON 路径")
	imageDir := flag.String("images", "", "压缩图片目录")
	userID := flag.Uint64("user-id", 1, "创建人用户 ID")
	flag.Parse()
	if *manifestPath == "" || *imageDir == "" || *userID == 0 {
		log.Fatal("必须提供 -manifest、-images 和有效的 -user-id")
	}
	if err := run(*manifestPath, *imageDir, *userID); err != nil {
		log.Fatal(err)
	}
}

func run(manifestPath, imageDir string, userID uint64) error {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("读取导入清单失败: %w", err)
	}
	var entries []importMemory
	if err := json.Unmarshal(manifestData, &entries); err != nil {
		return fmt.Errorf("解析导入清单失败: %w", err)
	}
	if len(entries) == 0 {
		return errors.New("导入清单为空")
	}
	if err := config.Startup(); err != nil {
		return err
	}
	if err := database.InitMySQL(); err != nil {
		return err
	}
	defer func() {
		if closeErr := database.Shutdown(context.Background()); closeErr != nil {
			log.Printf("关闭数据库失败: %v", closeErr)
		}
	}()
	ctx := context.Background()
	if err := migration.Apply(ctx, database.MySQL()); err != nil {
		return err
	}
	var user model.User
	if err := database.MySQL().WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("查询导入用户失败: %w", err)
	}
	if user.Role != model.UserRoleAdmin || user.Status != model.UserStatusApproved {
		return errors.New("导入用户必须是已通过的管理员")
	}
	adminCtx := authctx.WithUser(ctx, &user)
	memoryDAO := dao.NewMemoryDao()
	mediaService, err := service.NewConfiguredMediaService(database.MySQL(), dao.NewMediaAssetDao())
	if err != nil {
		return err
	}
	memoryService := service.NewMemoryService(database.MySQL(), memoryDAO, dao.NewMemoryPhotoDao())
	imported, skipped := 0, 0
	for index, entry := range entries {
		if entry.SourceKey == "" || len(entry.Photos) == 0 {
			return fmt.Errorf("第 %d 条记录缺少 sourceKey 或图片", index+1)
		}
		if _, err := memoryDAO.GetBySourceKey(ctx, database.MySQL(), entry.SourceKey); err == nil {
			skipped++
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("检查第 %d 条记录失败: %w", index+1, err)
		}
		images := make([]*proto.MemoryPhotoInput, 0, len(entry.Photos))
		for _, photo := range entry.Photos {
			path := filepath.Join(imageDir, filepath.Base(photo.File))
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("打开图片 %s 失败: %w", photo.File, err)
			}
			result, uploadErr := mediaService.UploadImage(ctx, userID, model.MediaBizTypeMemory, file)
			_ = file.Close()
			if uploadErr != nil {
				return fmt.Errorf("导入图片 %s 失败: %w", photo.File, uploadErr)
			}
			images = append(images, &proto.MemoryPhotoInput{
				MediaId: result.MediaID, ObjectKey: result.ObjectKey, Width: result.Width,
				Height: result.Height, AltText: photo.AltText,
			})
		}
		if _, err := memoryService.Create(adminCtx, &proto.CreateMemoryRequest{
			Date: entry.Date, Place: entry.Place, Title: entry.Title, Description: entry.Description,
			Images: images, SourceKey: entry.SourceKey,
		}); err != nil {
			return fmt.Errorf("创建第 %d 条记录失败: %w", index+1, err)
		}
		imported++
		log.Printf("已导入 %d/%d: %s %s", index+1, len(entries), entry.Date, entry.Title)
	}
	log.Printf("时光相册导入完成: 新增=%d 跳过=%d 总计=%d", imported, skipped, len(entries))
	return nil
}
