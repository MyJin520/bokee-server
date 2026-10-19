package config

type JWTConfig struct {
	Secret        string   `yaml:"secret"`
	Expire        string   `yaml:"expire"`
	RefreshExpire string   `yaml:"refresh-expire"`
	Issuer        string   `yaml:"issuer"`
	Audience      []string `yaml:"audience"`
}
