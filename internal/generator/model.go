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

func GenerateModel(entityName string, fields []types.Field, action types.Action) error {
	cfg := config.GetConfig()
	modelName := inflection.Singular(strcase.ToCamel(entityName))

	switch action {
	case types.CreateAction:
		return createModel(modelName, fields, cfg)
	case types.AddFieldsAction:
		return addFieldsToModel(modelName, fields, cfg)
	case types.RemoveFieldsAction:
		return removeFieldsFromModel(modelName, fields, cfg)
	case types.DropAction:
		return removeModel(modelName, cfg)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func createModel(modelName string, fields []types.Field, cfg *config.Config) error {
	modelData := prepareModelData(modelName, fields)

	funcMap := template.FuncMap{
		"toCamel":   strcase.ToCamel,
		"toSnake":   strcase.ToSnake,
		"pluralize": inflection.Plural,
	}

	tmpl, err := template.New("model.tmpl").Funcs(funcMap).ParseFiles("templates/models/model.tmpl")
	if err != nil {
		return fmt.Errorf("error parsing model template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, modelData)
	if err != nil {
		return fmt.Errorf("error executing model template: %w", err)
	}

	err = saveModelToFile(modelName, buf.Bytes(), cfg)
	if err != nil {
		return err
	}

	err = updateReferencedModel(modelName, modelData.BelongsToRelations, cfg)
	if err != nil {
		return fmt.Errorf("error updating referenced models: %w", err)
	}

	return nil
}

func prepareModelData(modelName string, fields []types.Field) ModelData {
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

	return modelData
}

func addFieldsToModel(modelName string, newFields []types.Field, cfg *config.Config) error {
	filePath := getModelFilePath(modelName, cfg)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file %s: %w", filePath, err)
	}

	// find struct
	var structDecl *ast.TypeSpec
	ast.Inspect(node, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == modelName {
			structDecl = ts
			return false
		}
		return true
	})

	if structDecl == nil {
		return fmt.Errorf("struct %s not found in file %s", modelName, filePath)
	}

	// add fields to struct
	structType, ok := structDecl.Type.(*ast.StructType)
	if !ok {
		return fmt.Errorf("%s is not a struct type", modelName)
	}

	for _, field := range newFields {
		newField := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(strcase.ToCamel(field.Name))},
			Type:  ast.NewIdent(getGoType(field)),
		}
		structType.Fields.List = append(structType.Fields.List, newField)
	}

	var buf bytes.Buffer
	err = format.Node(&buf, fset, node)
	if err != nil {
		return fmt.Errorf("error formatting updated file: %w", err)
	}

	err = saveModelToFile(modelName, buf.Bytes(), cfg)
	if err != nil {
		return err
	}

	modelData := prepareModelData(modelName, newFields)
	err = updateReferencedModel(modelName, modelData.BelongsToRelations, cfg)
	if err != nil {
		return fmt.Errorf("error updating referenced models: %w", err)
	}

	return nil
}

func removeFieldsFromModel(modelName string, fieldsToRemove []types.Field, cfg *config.Config) error {
	filePath := getModelFilePath(modelName, cfg)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file %s: %w", filePath, err)
	}

	// find struct def
	var structDecl *ast.TypeSpec
	ast.Inspect(node, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == modelName {
			structDecl = ts
			return false
		}
		return true
	})

	if structDecl == nil {
		return fmt.Errorf("struct %s not found in file %s", modelName, filePath)
	}

	// rem fields from struct
	structType, ok := structDecl.Type.(*ast.StructType)
	if !ok {
		return fmt.Errorf("%s is not a struct type", modelName)
	}

	fieldsToRemoveMap := make(map[string]bool)
	for _, field := range fieldsToRemove {
		fieldsToRemoveMap[strcase.ToCamel(field.Name)] = true
	}

	newFields := make([]*ast.Field, 0)
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 && !fieldsToRemoveMap[field.Names[0].Name] {
			newFields = append(newFields, field)
		}
	}
	structType.Fields.List = newFields

	var buf bytes.Buffer
	err = format.Node(&buf, fset, node)
	if err != nil {
		return fmt.Errorf("error formatting updated file: %w", err)
	}

	return saveModelToFile(modelName, buf.Bytes(), cfg)
}

func removeModel(modelName string, cfg *config.Config) error {
	filePath := getModelFilePath(modelName, cfg)
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("error removing model file %s: %w", filePath, err)
	}
	fmt.Printf("Model file removed: %s\n", filePath)
	return nil
}

func getModelFilePath(modelName string, cfg *config.Config) string {
	modelDir := cfg.ModelDir
	if modelDir == "" {
		modelDir = "models"
	}
	fileName := fmt.Sprintf("%s.go", strcase.ToSnake(modelName))
	return filepath.Join(modelDir, fileName)
}

func saveModelToFile(modelName string, content []byte, cfg *config.Config) error {
	filePath := getModelFilePath(modelName, cfg)

	modelDir := filepath.Dir(filePath)
	err := os.MkdirAll(modelDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating model directory: %w", err)
	}

	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		return fmt.Errorf("error writing model file: %w", err)
	}

	fmt.Printf("Model file updated: %s\n", filePath)
	return nil
}

func updateReferencedModel(currentModel string, relations []Relation, cfg *config.Config) error {
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

func getGoType(field types.Field) string {
	if field.IsEnum {
		return "string"
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
