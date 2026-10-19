package config

type SystemConfig struct {
	DbType               string `yaml:"db-type"`                // 数据库类型
	OssType              string `yaml:"oss-type"`               // 存储类型
	Addr                 string `yaml:"addr"`                   // 服务监听地址
	DefaultAdminPassword string `yaml:"default-admin-password"` // 初始管理员密码（空则自动生成随机密码）
}
