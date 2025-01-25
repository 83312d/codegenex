package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codegenex/internal/config"
)

func GenerateFileName(name string) string {
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s_%s.sql", timestamp, name)
}

func SaveToFile(migrationSQL, fileName string, cfg *config.Config) error {
	migrationDir := cfg.MigrationDir
	if migrationDir == "" {
		migrationDir = "migrations"
	}

	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return fmt.Errorf("error creating migration directory: %w", err)
	}

	filePath := filepath.Join(migrationDir, fileName)
	return os.WriteFile(filePath, []byte(migrationSQL), 0644)
}
