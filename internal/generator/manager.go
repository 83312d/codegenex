package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codegenex/internal/config"
	"codegenex/internal/types"
)

type Manager struct {
	Config *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{Config: cfg}
}

func (m *Manager) GenerateEntity(name string, fields []types.Field) error {
	err := m.GenerateAndSaveMigration(name, fields)
	if err != nil {
		return err
	}

	err = m.GenerateAndSaveModel(name, fields)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) GenerateAndSaveMigration(name string, fields []types.Field) error {
	migrationSQL, err := GenerateMigration(name, fields)
	if err != nil {
		return fmt.Errorf("error generating migration: %w", err)
	}

	fileName := m.generateMigrationFileName(name)
	err = m.saveMigrationToFile(migrationSQL, fileName)
	if err != nil {
		return fmt.Errorf("error saving migration: %w", err)
	}

	fmt.Printf("Migration file generated: %s\n", fileName)
	return nil
}

func (m *Manager) GenerateAndSaveModel(name string, fields []types.Field) error {
	err := GenerateModel(name, fields)
	if err != nil {
		return fmt.Errorf("error generating model: %w", err)
	}
	return nil
}

func (m *Manager) generateMigrationFileName(name string) string {
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s_%s.sql", timestamp, name)
}

func (m *Manager) saveMigrationToFile(migrationSQL, fileName string) error {
	migrationDir := m.Config.MigrationDir
	if migrationDir == "" {
		migrationDir = "migrations"
	}

	err := os.MkdirAll(migrationDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating migration directory: %w", err)
	}

	filePath := filepath.Join(migrationDir, fileName)
	err = os.WriteFile(filePath, []byte(migrationSQL), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %w", err)
	}

	return nil
}
