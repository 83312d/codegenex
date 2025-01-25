package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	ModelDir     string `json:"model_dir"`
	MigrationDir string `json:"migration_dir"`
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
			config.ModelDir = "models"
		}
		if config.MigrationDir == "" {
			config.MigrationDir = "migrations"
		}
	})
	return config
}
