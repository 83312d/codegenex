package generator

import (
	"codegenex/internal/config"
	"codegenex/internal/types"
	"fmt"
)

type Manager struct {
	Config *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{Config: cfg}
}

func (m *Manager) GenerateEntity(entityName string, action types.Action, fields []types.Field) error {
	switch action {
	case types.CreateAction:
		return m.handleCreateAction(entityName, fields)
	case types.AddFieldsAction:
		return m.handleAddFieldsAction(entityName, fields)
	case types.RemoveFieldsAction:
		return m.handleRemoveFieldsAction(entityName, fields)
	case types.DropAction:
		return m.handleDropAction(entityName)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (m *Manager) handleCreateAction(entityName string, fields []types.Field) error {
	err := m.GenerateAndSaveMigration(entityName, fields, types.CreateAction)
	if err != nil {
		return err
	}

	err = m.GenerateAndSaveModel(entityName, fields, types.CreateAction)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) handleAddFieldsAction(entityName string, fields []types.Field) error {
	err := m.GenerateAndSaveMigration(entityName, fields, types.AddFieldsAction)
	if err != nil {
		return err
	}

	err = m.GenerateAndSaveModel(entityName, fields, types.AddFieldsAction)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) handleRemoveFieldsAction(entityName string, fields []types.Field) error {
	err := m.GenerateAndSaveMigration(entityName, fields, types.RemoveFieldsAction)
	if err != nil {
		return err
	}

	err = m.GenerateAndSaveModel(entityName, fields, types.RemoveFieldsAction)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) handleDropAction(entityName string) error {
	err := m.GenerateAndSaveMigration(entityName, nil, types.DropAction)
	if err != nil {
		return err
	}

	err = m.RemoveModel(entityName)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) GenerateAndSaveMigration(entityName string, fields []types.Field, action types.Action) error {
	return GenerateAndSaveMigration(entityName, fields, action, m.Config)
}

func (m *Manager) GenerateAndSaveModel(entityName string, fields []types.Field, action types.Action) error {
	return GenerateModel(entityName, fields, action)
}

func (m *Manager) RemoveModel(entityName string) error {
	// TODO:
	return nil
}
