package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"

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

type ModelData struct {
	Name               string
	Fields             []ModelField
	Imports            []string
	HasManyRelations   []Relation
	BelongsToRelations []Relation
	Enums              []EnumData
}

type ModelField struct {
	Name     string
	Type     string
	IsEnum   bool
	EnumType string
}

type Relation struct {
	ModelName string
	FieldName string
}

func GenerateMigrationAndModel(name string, fields []types.Field) (string, error) {
	migration, err := GenerateMigration(name, fields)
	if err != nil {
		return "", err
	}

	err = GenerateModel(name, fields)
	if err != nil {
		return "", err
	}

	return migration, nil
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

func GenerateModel(name string, fields []types.Field) error {
	modelName := inflection.Singular(strcase.ToCamel(strings.TrimPrefix(name, "create_")))
	modelData := ModelData{
		Name:               modelName,
		Fields:             make([]ModelField, 0),
		Enums:              make([]EnumData, 0),
		HasManyRelations:   make([]Relation, 0),
		BelongsToRelations: make([]Relation, 0),
		Imports:            make([]string, 0),
	}

	needsTimeImport := false

	for _, field := range fields {
		modelField := ModelField{
			Name: strcase.ToCamel(field.Name),
			Type: getGoType(field),
		}

		if field.Type == "time" {
			needsTimeImport = true
		}

		if field.IsEnum {
			enumName := fmt.Sprintf("%s%sType", modelName, strcase.ToCamel(field.Name))
			modelField.Type = enumName
			modelData.Enums = append(modelData.Enums, EnumData{
				Name:   enumName,
				Values: field.EnumValues,
			})
		}

		if field.IsReference {
			referencedModel := field.ReferencedModel
			if referencedModel == "" {
				referencedModel = inflection.Singular(strcase.ToCamel(strings.TrimSuffix(field.Name, "_id")))
			}
			modelData.BelongsToRelations = append(modelData.BelongsToRelations, Relation{
				ModelName: referencedModel,
				FieldName: strcase.ToCamel(strings.TrimSuffix(field.Name, "_id")),
			})
		}

		modelData.Fields = append(modelData.Fields, modelField)
	}

	if needsTimeImport {
		modelData.Imports = append(modelData.Imports, "time")
	}

	err := updateReferencedModel(modelName, modelData.BelongsToRelations)
	if err != nil {
		return fmt.Errorf("error updating referenced model: %w", err)
	}

	funcMap := template.FuncMap{
		"toCamel": strcase.ToCamel,
	}

	tmpl, err := template.New("model.tmpl").Funcs(funcMap).ParseFiles("templates/model.tmpl")
	if err != nil {
		return fmt.Errorf("error parsing model template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, modelData)
	if err != nil {
		return fmt.Errorf("error executing model template: %w", err)
	}

	cfg := config.GetConfig()
	modelDir := cfg.ModelDir
	if modelDir == "" {
		modelDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %w", err)
		}
	}

	err = os.MkdirAll(modelDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating model directory: %w", err)
	}

	fileName := fmt.Sprintf("%s.go", strings.ToLower(modelName))
	filePath := filepath.Join(modelDir, fileName)

	err = os.WriteFile(filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing model file: %w", err)
	}

	fmt.Printf("Model file generated: %s\n", filePath)

	return nil
}

func updateReferencedModel(currentModel string, relations []Relation) error {
	cfg := config.GetConfig()
	modelDir := cfg.ModelDir
	if modelDir == "" {
		var err error
		modelDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %w", err)
		}
	}

	for _, relation := range relations {
		filePath := filepath.Join(modelDir, strings.ToLower(relation.ModelName)+".go")

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("error parsing file %s: %w", filePath, err)
		}

		var structDecl *ast.TypeSpec
		ast.Inspect(node, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == relation.ModelName {
				structDecl = ts
				return false
			}
			return true
		})

		if structDecl == nil {
			return fmt.Errorf("struct %s not found in file %s", relation.ModelName, filePath)
		}

		newField := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(inflection.Plural(currentModel))},
			Type: &ast.ArrayType{
				Elt: &ast.StarExpr{X: ast.NewIdent(currentModel)},
			},
		}

		structType, ok := structDecl.Type.(*ast.StructType)
		if !ok {
			return fmt.Errorf("%s is not a struct type", relation.ModelName)
		}

		structType.Fields.List = append(structType.Fields.List, newField)

		var buf bytes.Buffer
		err = format.Node(&buf, fset, node)
		if err != nil {
			return fmt.Errorf("error formatting updated file: %w", err)
		}

		err = os.WriteFile(filePath, buf.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("error writing updated file: %w", err)
		}
	}

	return nil
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

func getGoType(field types.Field) string {
	if field.IsEnum {
		return "string" // ENUM в Go представляется как string
	}

	isArray := strings.HasSuffix(field.Type, "[]")
	if isArray {
		field.Type = strings.TrimSuffix(field.Type, "[]")
	}

	var baseType string
	switch field.Type {
	case "int":
		baseType = "int64"
	case "string":
		baseType = "string"
	case "bool":
		baseType = "bool"
	case "time":
		baseType = "time.Time"
	case "float":
		baseType = "float64"
	case "jsonb":
		baseType = "map[string]interface{}"
	default:
		baseType = "interface{}"
	}

	if isArray {
		return "[]" + baseType
	}

	return baseType
}

func getOnDeleteOption(option string) string {
	switch option {
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
