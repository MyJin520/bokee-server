package basic

import (
	"gorm.io/gorm"
	"time"
)

type BaseModel struct {
	ID        uint           `gorm:"primarykey;autoIncrement;comment:主键ID"`
	CreatedAt time.Time      `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt time.Time      `gorm:"column:updated_at;comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index;comment:删除时间"`
}
