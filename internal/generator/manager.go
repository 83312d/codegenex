package generator

import (
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
	return GenerateAndSaveMigration(name, fields, m.Config)
}

func (m *Manager) GenerateAndSaveModel(name string, fields []types.Field) error {
	return GenerateModel(name, fields)
}
