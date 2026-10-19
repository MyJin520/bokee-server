package global

import (
	"bokee/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	Redis  *redis.Client
	Config *config.Config
	Log    *zap.Logger
)

const (
	SuperRoleCode = 1
	// UserRoleCode 注册用户默认角色（普通用户）标识，仅拥有账户自助与个人文章管理权限
	UserRoleCode = 2
)
