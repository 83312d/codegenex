package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

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

func GenerateMigration(name string, fields []types.Field) (string, error) {
	tableName := inflection.Plural(strcase.ToSnake(strings.TrimPrefix(name, "create_")))

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
		}

		if field.IsEnum {
			enumName := fmt.Sprintf("%s_%s", tableName, inflection.Plural(field.Name))
			fieldData.EnumName = enumName
			migrationData.Enums = append(migrationData.Enums, EnumData{
				Name:   enumName,
				Values: field.EnumValues,
			})
		}

		migrationData.Fields = append(migrationData.Fields, fieldData)

		if field.IsIndex {
			migrationData.Indexes = append(migrationData.Indexes, IndexData{
				Name:    fmt.Sprintf("idx_%s_%s", tableName, field.Name),
				Columns: []string{field.Name},
			})
		}

		if field.IsReference {
			refTable := inflection.Plural(strings.TrimSuffix(field.Name, "_id"))
			migrationData.References = append(migrationData.References, ReferenceData{
				Column:    field.Name,
				RefTable:  refTable,
				RefColumn: "id",
				OnDelete:  getOnDeleteOption(field.RefOptions),
			})
		}
	}

	funcMap := template.FuncMap{
		"toSnake": strcase.ToSnake,
	}

	tmpl, err := template.New("migration.tmpl").Funcs(funcMap).ParseFiles("templates/migration.tmpl")
	if err != nil {
		return "", fmt.Errorf("error parsing migration template: %w", err)
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
