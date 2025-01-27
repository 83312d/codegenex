package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"codegenex/internal/config"
	"codegenex/internal/types"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"
)

func GenerateRepository(entityName string, fields []types.Field, action types.Action) error {
	cfg := config.GetConfig()
	modelName := inflection.Singular(strcase.ToCamel(entityName))

	switch action {
	case types.CreateAction:
		return createRepository(modelName, fields, cfg)
	case types.AddFieldsAction, types.RemoveFieldsAction:
		return updateRepository(modelName, fields, cfg)
	case types.DropAction:
		return removeRepository(modelName, cfg)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func createRepository(modelName string, fields []types.Field, cfg *config.Config) error {
	repoData := prepareRepositoryData(modelName, fields, cfg)

	funcMap := template.FuncMap{
		"trimSuffix": strings.TrimSuffix,
		"title":      strings.Title,
	}

	tmpl, err := template.New("repository.tmpl").Funcs(funcMap).ParseFiles("templates/repositories/repository.tmpl")
	if err != nil {
		return fmt.Errorf("error parsing repository template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, repoData)
	if err != nil {
		return fmt.Errorf("error executing repository template: %w", err)
	}

	return saveRepositoryFile(modelName, buf.Bytes(), cfg)
}

func updateRepository(modelName string, fields []types.Field, cfg *config.Config) error {
	// For simplicity, we'll recreate the repository file
	return createRepository(modelName, fields, cfg)
}

func removeRepository(modelName string, cfg *config.Config) error {
	fileName := fmt.Sprintf("%s_repo.go", strcase.ToSnake(modelName))
	filePath := filepath.Join(cfg.RepositoryDir, fileName)

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing repository file: %w", err)
	}

	fmt.Printf("Repository file removed: %s\n", filePath)
	return nil
}

func prepareRepositoryData(modelName string, fields []types.Field, cfg *config.Config) map[string]interface{} {
	varName := strcase.ToLowerCamel(modelName)
	tableName := inflection.Plural(strcase.ToSnake(modelName))

	columnNames := make([]string, len(fields))
	placeholders := make([]string, len(fields))
	insertValues := make([]string, len(fields))
	scanValues := make([]string, len(fields))
	updateSet := make([]string, len(fields))
	updateValues := make([]string, len(fields))

	type FieldInfo struct {
		types.Field
		MethodName string
		ParamName  string
		ColumnName string
	}

	fieldInfos := make([]FieldInfo, len(fields))

	for i, field := range fields {
		columnNames[i] = strcase.ToSnake(field.Name)
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		insertValues[i] = fmt.Sprintf("%s.%s", varName, field.Name)
		scanValues[i] = fmt.Sprintf("&%s.%s", varName, field.Name)
		updateSet[i] = fmt.Sprintf("%s = $%d", columnNames[i], i+1)
		updateValues[i] = fmt.Sprintf("%s.%s", varName, field.Name)

		fieldInfo := FieldInfo{Field: field, ColumnName: columnNames[i]}

		if field.IsReference {
			referencedModel := inflection.Singular(strcase.ToCamel(strings.TrimSuffix(field.Name, "_id")))
			fieldInfo.MethodName = referencedModel
			fieldInfo.ParamName = strcase.ToLowerCamel(referencedModel) + "ID"
		}

		fieldInfos[i] = fieldInfo
	}

	hasManyRelations := make([]struct {
		FieldName  string
		ModelName  string
		TableName  string
		VarName    string
		ScanValues string
	}, 0)

	for _, field := range fields {
		if field.IsReference {
			referencedModel := inflection.Singular(strcase.ToCamel(strings.TrimSuffix(field.Name, "_id")))
			relationName := inflection.Plural(referencedModel)
			tableName := inflection.Plural(strcase.ToSnake(referencedModel))
			varName := strcase.ToLowerCamel(referencedModel)

			scanValues := "/* TODO: Add scan values for " + referencedModel + " */"

			hasManyRelations = append(hasManyRelations, struct {
				FieldName  string
				ModelName  string
				TableName  string
				VarName    string
				ScanValues string
			}{
				FieldName:  relationName,
				ModelName:  referencedModel,
				TableName:  tableName,
				VarName:    varName,
				ScanValues: scanValues,
			})
		}
	}

	return map[string]interface{}{
		"Name":              modelName,
		"VarName":           varName,
		"PackageName":       cfg.PackageName,
		"TableName":         tableName,
		"Fields":            fieldInfos,
		"ColumnNames":       strings.Join(columnNames, ", "),
		"Placeholders":      strings.Join(placeholders, ", "),
		"InsertValues":      strings.Join(insertValues, ", "),
		"ScanValues":        strings.Join(scanValues, ", "),
		"UpdateSet":         strings.Join(updateSet, ", "),
		"UpdatePlaceholder": fmt.Sprintf("%d", len(fields)+1),
		"UpdateValues":      strings.Join(append(updateValues, varName+".ID"), ", "),
		"HasManyRelations":  hasManyRelations,
	}
}

func saveRepositoryFile(modelName string, content []byte, cfg *config.Config) error {
	fileName := fmt.Sprintf("%s_repo.go", strcase.ToSnake(modelName))
	filePath := filepath.Join(cfg.RepositoryDir, fileName)

	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("error creating repository directory: %w", err)
	}

	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		return fmt.Errorf("error writing repository file: %w", err)
	}

	fmt.Printf("Repository file generated: %s\n", filePath)
	return nil
}
