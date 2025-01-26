package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"codegenex/internal/config"
	"codegenex/internal/types"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"
)

type MigrationData struct {
	TableName  string
	Fields     []FieldData
	Indexes    []IndexData
	References []ReferenceData
	Enums      []EnumData
}

type FieldData struct {
	Name         string
	Type         string
	SQLType      string
	IsNullable   bool
	DefaultValue string
	IsEnum       bool
	EnumName     string
	IsUnique     bool
	IsReference  bool
	RefTable     string
	RefColumn    string
	OnDelete     string
}

type IndexData struct {
	Name    string
	Columns []string
}

type ReferenceData struct {
	Column    string
	RefTable  string
	RefColumn string
	OnDelete  string
}

type EnumData struct {
	Name   string
	Values []string
}

func GenerateAndSaveMigration(entityName string, fields []types.Field, action types.Action, cfg *config.Config) error {
	migrationSQL, err := GenerateMigration(entityName, fields, action)
	if err != nil {
		return fmt.Errorf("error generating migration: %w", err)
	}

	fileName := generateMigrationFileName(entityName, action)
	err = saveMigrationToFile(migrationSQL, fileName, cfg)
	if err != nil {
		return fmt.Errorf("error saving migration: %w", err)
	}

	fmt.Printf("Migration file generated: %s\n", fileName)
	return nil
}

func GenerateMigration(entityName string, fields []types.Field, action types.Action) (string, error) {
	tableName := inflection.Plural(strcase.ToSnake(entityName))

	migrationData := MigrationData{
		TableName:  tableName,
		Fields:     make([]FieldData, 0),
		Indexes:    make([]IndexData, 0),
		References: make([]ReferenceData, 0),
		Enums:      make([]EnumData, 0),
	}

	for _, field := range fields {
		fieldData := FieldData{
			Name:         field.Name,
			Type:         field.Type,
			SQLType:      getSQLType(field),
			IsNullable:   field.IsNullable,
			DefaultValue: field.DefaultValue,
			IsEnum:       field.IsEnum,
			IsUnique:     field.IsUnique,
			IsReference:  field.IsReference,
		}

		if field.IsEnum {
			enumName := fmt.Sprintf("%s_%s", tableName, inflection.Plural(field.Name))
			fieldData.EnumName = enumName
			migrationData.Enums = append(migrationData.Enums, EnumData{
				Name:   enumName,
				Values: field.EnumValues,
			})
		}

		if field.IsReference {
			refParts := strings.Split(field.RefOptions, ".")
			if len(refParts) == 2 {
				fieldData.RefTable = refParts[0]
				fieldData.RefColumn = refParts[1]
			} else {
				fieldData.RefTable = inflection.Plural(strings.TrimSuffix(field.Name, "_id"))
				fieldData.RefColumn = "id"
			}
			fieldData.OnDelete = getOnDeleteOption(field.RefOptions)
		}

		migrationData.Fields = append(migrationData.Fields, fieldData)

		if field.IsIndex {
			migrationData.Indexes = append(migrationData.Indexes, IndexData{
				Name:    fmt.Sprintf("idx_%s_%s", tableName, field.Name),
				Columns: []string{field.Name},
			})
		}

		if field.IsReference {
			migrationData.References = append(migrationData.References, ReferenceData{
				Column:    field.Name,
				RefTable:  fieldData.RefTable,
				RefColumn: fieldData.RefColumn,
				OnDelete:  fieldData.OnDelete,
			})
		}
	}

	templateName := filepath.Join("templates", "migrations", action.String()+".tmpl")

	if _, err := os.Stat(templateName); os.IsNotExist(err) {
		return "", fmt.Errorf("template for action %s does not exist: %w", action, err)
	}

	funcMap := template.FuncMap{
		"toSnake": strcase.ToSnake,
	}

	tmpl, err := template.New(filepath.Base(templateName)).Funcs(funcMap).ParseFiles(templateName)
	if err != nil {
		return "", fmt.Errorf("error parsing template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, migrationData)
	if err != nil {
		return "", fmt.Errorf("error executing migration template: %w", err)
	}

	return buf.String(), nil
}

func getSQLType(field types.Field) string {
	if field.IsEnum {
		return fmt.Sprintf("%s_%s", field.Name, "type")
	}

	isArray := strings.HasSuffix(field.Type, "[]")
	if isArray {
		field.Type = strings.TrimSuffix(field.Type, "[]")
	}

	var baseType string
	switch field.Type {
	case "int":
		baseType = "INTEGER"
	case "string":
		baseType = "VARCHAR(255)"
	case "bool":
		baseType = "BOOLEAN"
	case "time":
		baseType = "TIMESTAMP"
	case "float":
		baseType = "NUMERIC"
	case "jsonb":
		return "JSONB"
	default:
		baseType = "VARCHAR(255)"
	}

	if isArray {
		return baseType + "[]"
	}

	return baseType
}

func getOnDeleteOption(option string) string {
	switch option {
	case "cascade":
		return "CASCADE"
	case "nullify":
		return "SET NULL"
	case "restrict":
		return "RESTRICT"
	case "no_action":
		return "NO ACTION"
	default:
		return "CASCADE"
	}
}

func generateMigrationFileName(entityName string, action types.Action) string {
	timestamp := time.Now().Format("20060102150405")
	var actionStr string
	switch action {
	case types.CreateAction:
		actionStr = "create"
	case types.AddFieldsAction:
		actionStr = "add_fields_to"
	case types.RemoveFieldsAction:
		actionStr = "remove_fields_from"
	case types.DropAction:
		actionStr = "drop"
	}

	return fmt.Sprintf("%s_%s_%s.sql", timestamp, actionStr, entityName)
}

func saveMigrationToFile(migrationSQL, fileName string, cfg *config.Config) error {
	migrationDir := cfg.MigrationDir
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
