package config

type DBConfig struct {
	UserName        string `yaml:"user-name"`
	Password        string `yaml:"password"`
	DBName          string `yaml:"db-name"`
	Config          string `yaml:"config"`
	Path            string `yaml:"path"`
	LogMode         string `yaml:"log-mode"`
	MaxIdleConn     int    `yaml:"max-idle-conn"`
	MaxOpenConn     int    `yaml:"max-open-conn"`
	ConnMaxLifeTime int    `yaml:"conn-max-life-time"`
}
