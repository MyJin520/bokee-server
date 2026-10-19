package basic

type Files struct {
	BaseModel
	Url          string `gorm:"size:125;not null;comment:文件地址"`
	Ext          string `gorm:"size:10;not null;comment:文件后缀"`
	Size         int64  `gorm:"not null;comment:文件大小"`
	OriginalName string `gorm:"size:125;not null;comment:原始文件名"`
	Hash         string `gorm:"size:255;not null;uniqueIndex;comment:文件hash"`
}

func NewFiles(url, ext string, size int64, originalName, hash string) *Files {
	return &Files{
		Url:          url,
		Ext:          ext,
		Size:         size,
		OriginalName: originalName,
		Hash:         hash,
	}
}

func (Files) TableName() string {
	return "files"
}
