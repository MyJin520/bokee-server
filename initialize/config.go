package initialize

import (
	"bokee/config"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	envAppEnv = "APP_ENV"

	configProdPath = "./config.yaml"
	configDevPath  = "./config.dev.yaml"
)

func resolveConfigPath() string {
	if os.Getenv(envAppEnv) == "production" {
		fmt.Println("===当前项目为《生产环境》===")
		return configProdPath
	}
	fmt.Println("===当前项目为开发环境===")
	return configDevPath
}

func initLoadConfig() (*config.Config, error) {
	configPath := resolveConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg config.Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
