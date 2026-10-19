package filedx

import (
	"bokee/global"
	"bokee/pkg/randx"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func UploadToLocal(data []byte, ext string) (string, error) {
	basePath := global.Config.Oss.LocalOss.Path
	if basePath == "" {
		basePath = "./file_directory"
	}

	subDir := time.Now().Format("2006/01/02")
	fullDir := filepath.Join(basePath, subDir)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return "", fmt.Errorf("创建存储目录失败: %w", err)
	}

	fileName := fmt.Sprintf("%s%s", randx.RandomUUID(), ext)
	fullPath := filepath.Join(fullDir, fileName)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return filepath.ToSlash(filepath.Join(subDir, fileName)), nil
}
