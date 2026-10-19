package filedx

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
	"os"
	"testing"
)

func TestMinio(t *testing.T) {
	ctx := context.Background()
	// minio配置
	minioHost := "127.0.0.1:39000"
	accesskeyID := "kimJin"
	accesskeySecret := "kimJin123456"
	useSSL := false
	// 初始化客户端
	minioClient, err := minio.New(
		minioHost,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accesskeyID, accesskeySecret, ""),
			Secure: useSSL,
		},
	)
	if err != nil {
		fmt.Println("初始化客户端失败：: " + err.Error())
		return
	}
	// 准备上传
	bucketName := "dor"
	objectName := "image.png"
	filePath := "C:\\DevelopMyJin\\developTools\\ideas\\IdeaProjects\\bokee\\src\\file_directory\\2026\\09\\10\\8e15e703-3b0a-441c-ad65-90b8f89d802d.png"
	// 检查桶是否存在
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		fmt.Println("检查桶失败: " + err.Error())
	}
	if !exists {
		err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			fmt.Println("创建桶失败: " + err.Error())
			return
		}
		fmt.Println("创建桶成功: " + bucketName)

	}

	// 方法一，直接传入本地文件
	//uploadInfo, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{})

	// 方法二，通过文件字节的形式
	fileReader, err := os.Open(filePath)
	defer func(fileReader *os.File) {
		_ = fileReader.Close()
	}(fileReader)
	// 上传文件
	fileStat, err := fileReader.Stat()
	fileSize := fileStat.Size()
	uploadInfo, err := minioClient.PutObject(ctx, bucketName, objectName, fileReader, fileSize, minio.PutObjectOptions{})
	if err != nil {
		fmt.Println("上传文件失败: " + err.Error())
		return
	}
	log.Printf("文件上传成功！对象名: %s, 大小: %d 字节\n", objectName, uploadInfo.Size)
}
