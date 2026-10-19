package filedx

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"strings"
	"sync"
	"time"

	"bokee/global"
	"bokee/pkg/randx"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	clientInstance *minio.Client
	clientOnce     sync.Once
	initErr        error
)

// getMinioClient 获取 MinIO 客户端单例
func getMinioClient() (*minio.Client, error) {
	clientOnce.Do(func() {
		cfg := global.Config.Oss.MinioOss
		client, err := minio.New(
			cfg.Host,
			&minio.Options{
				Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
				Secure: cfg.UseSSL,
			},
		)
		if err != nil {
			initErr = fmt.Errorf("minio客户端初始化失败: %w", err)
			return
		}

		ctx := context.Background()
		exists, err := client.BucketExists(ctx, cfg.BucketName)
		if err != nil {
			initErr = fmt.Errorf("检查桶失败: %w", err)
			return
		}
		if !exists {
			if err := client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{}); err != nil {
				initErr = fmt.Errorf("创建桶失败: %w", err)
				return
			}
		}
		clientInstance = client
	})
	return clientInstance, initErr
}

// UploadToMinio 将文件字节上传到 MinIO，返回文件的可访问 URL。
func UploadToMinio(data []byte, ext string) (string, error) {
	cfg := global.Config.Oss.MinioOss

	client, err := getMinioClient()
	if err != nil {
		return "", err
	}

	// 按日期分目录，并使用 UUID 保证对象名唯一
	objectName := fmt.Sprintf("%s/%s%s", time.Now().Format("2006/01/02"), randx.RandomUUID(), ext)

	// 根据扩展名推导 Content-Type，未知类型兜底
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = client.PutObject(
		context.Background(),
		cfg.BucketName,
		objectName,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", fmt.Errorf("上传文件到 MinIO 失败: %w", err)
	}

	// 拼接对外访问地址：{file-url}/{bucket}/{objectName}
	base := strings.TrimRight(cfg.FileUrl, "/")
	if base == "" {
		return objectName, nil
	}
	return base + "/" + cfg.BucketName + "/" + objectName, nil
}
