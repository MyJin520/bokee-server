package response

import (
	"bokee/internal/mods/basic"
	"time"
)

// FileResp 文件上传响应结构体
type FileResp struct {
	ID           uint      `json:"id"`
	Url          string    `json:"url"`
	Ext          string    `json:"ext"`
	Size         int64     `json:"size"`
	OriginalName string    `json:"originalName"`
	Hash         string    `json:"hash"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// NewFileResp 将文件模型转换为文件响应结构体
func NewFileResp(f basic.Files) *FileResp {
	return &FileResp{
		ID:           f.ID,
		Url:          f.Url,
		Ext:          f.Ext,
		Size:         f.Size,
		OriginalName: f.OriginalName,
		Hash:         f.Hash,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}
