package config

type Config struct {
	System SystemConfig `yaml:"system"` // 系统配置
	DB     DBConfig     `yaml:"db"`     // 数据库配置
	Redis  RedisConfig  `yaml:"redis"`  // Redis 配置
	Log    LogConfig    `yaml:"log"`    // 日志配置
	JWT    JWTConfig    `yaml:"jwt"`    // JWT 配置
	Oss    OssConfig    `yaml:"oss"`    // 存储配置
}
