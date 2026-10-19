package files

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"bokee/global"
	"bokee/internal/mods/basic"
	"bokee/internal/mods/response"
	"bokee/pkg/filedx"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type FileService struct{}

// Uploads 根据配置的 oss 类型上传文件，返回入库后的文件记录
func (s *FileService) Uploads(fileHeader *multipart.FileHeader) (*response.FileResp, error) {
	src, err := fileHeader.Open()
	if err != nil {
		global.Log.Error("打开文件失败", zap.Error(err), zap.String("filename", fileHeader.Filename))
		return nil, fmt.Errorf("打开文件失败")
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		global.Log.Error("读取文件失败", zap.Error(err), zap.String("filename", fileHeader.Filename))
		return nil, fmt.Errorf("读取文件失败")
	}
	const maxSize = 32 << 20 // 32MB
	if len(data) > maxSize {
		global.Log.Warn("文件上传失败：文件过大", zap.String("filename", fileHeader.Filename), zap.Int("size", len(data)))
		return nil, fmt.Errorf("文件过大，最大允许 %d MB", maxSize>>20)
	}

	mime, err := filedx.GetFileMIME(bytes.NewReader(data))
	if err != nil {
		global.Log.Warn("文件上传失败：无法识别文件类型", zap.String("filename", fileHeader.Filename))
		return nil, fmt.Errorf("无法识别文件类型")
	}
	if !filedx.CheckIfAllowed(mime.MIME.Value) {
		global.Log.Warn("文件上传失败：不允许的文件类型",
			zap.String("filename", fileHeader.Filename), zap.String("mime", mime.MIME.Value))
		return nil, fmt.Errorf("不允许的文件类型: %s", mime.MIME.Value)
	}

	hashBytes := md5.Sum(data)
	hashStr := hex.EncodeToString(hashBytes[:])
	ext := "." + mime.Extension

	var existing basic.Files
	if err := global.DB.Where("hash = ?", hashStr).First(&existing).Error; err == nil {
		return response.NewFileResp(existing), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		global.Log.Error("查询文件记录失败", zap.Error(err), zap.String("hash", hashStr))
		return nil, fmt.Errorf("查询文件记录失败")
	}

	var url string
	switch global.Config.System.OssType {
	case "local":
		url, err = filedx.UploadToLocal(data, ext)
	case "minio":
		url, err = filedx.UploadToMinio(data, ext)
	case "ali":
		err = fmt.Errorf("阿里云 OSS 上传暂未实现")
	default:
		err = fmt.Errorf("不支持的 oss 类型: %s", global.Config.System.OssType)
	}
	if err != nil {
		global.Log.Error("上传文件失败", zap.Error(err), zap.String("filename", fileHeader.Filename))
		return nil, fmt.Errorf("上传文件失败，请稍后重试")
	}

	fileRecord := basic.NewFiles(url, ext, int64(len(data)), fileHeader.Filename, hashStr)
	if err := global.DB.Create(fileRecord).Error; err != nil {
		global.Log.Error("文件记录入库失败", zap.Error(err), zap.String("filename", fileHeader.Filename))
		return nil, fmt.Errorf("文件记录入库失败")
	}
	return response.NewFileResp(*fileRecord), nil
}
