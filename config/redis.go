package config

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr         string `yaml:"addr"`           // 地址（host:port）
	Password     string `yaml:"password"`       // 密码
	DB           int    `yaml:"db"`             // 数据库编号
	PoolSize     int    `yaml:"pool-size"`      // 连接池大小
	MinIdleConns int    `yaml:"min-idle-conns"` // 最小空闲连接数
	DialTimeout  int    `yaml:"dial-timeout"`   // 连接超时（秒）
	ReadTimeout  int    `yaml:"read-timeout"`   // 读取超时（秒）
	WriteTimeout int    `yaml:"write-timeout"`  // 写入超时（秒）
}
