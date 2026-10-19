package filedx

import (
	"fmt"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
	"io"
)

var AllowedTypes = map[string]struct{}{
	// 图片
	"image/jpeg": {},
	"image/png":  {},
	"image/gif":  {},
	"image/webp": {},
	// 文档
	"application/pdf": {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       {},
}

// GetFileMIME 获取文件类型
func GetFileMIME(r io.Reader) (types.Type, error) {
	match, err := filetype.MatchReader(r)
	if match == filetype.Unknown || err != nil {
		return types.Type{}, fmt.Errorf("未知的文件类型")
	}
	return match, nil
}

func CheckIfAllowed(mimeValue string) bool {
	_, ok := AllowedTypes[mimeValue]
	return ok
}
