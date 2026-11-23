package config

type JWTConfig struct {
	Secret   string   `yaml:"secret"`
	Expire   string   `yaml:"expire"`
	Issuer   string   `yaml:"issuer"`
	Audience []string `yaml:"audience"`
}
