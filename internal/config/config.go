package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	ModelDir      string `json:"model_dir"`
	MigrationDir  string `json:"migration_dir"`
	RepositoryDir string `json:"repository_dir"`
	PackageName   string `json:"package_name"`
}

var (
	config     *Config
	configOnce sync.Once
)

func GetConfig() *Config {
	configOnce.Do(func() {
		config = &Config{}
		file, err := os.Open("codegenex.json")
		if err == nil {
			defer file.Close()
			decoder := json.NewDecoder(file)
			decoder.Decode(config)
		}

		if config.ModelDir == "" {
			config.ModelDir = "_gen/models"
		}
		if config.MigrationDir == "" {
			config.MigrationDir = "_gen/migrations"
		}
		if config.RepositoryDir == "" {
			config.RepositoryDir = "_gen/repositories"
		}
		if config.PackageName == "" {
			config.PackageName = "myapp"
		}
	})
	return config
}
