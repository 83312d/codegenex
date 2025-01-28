package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
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
		err := createRepository(modelName, fields, cfg)
		if err != nil {
			return err
		}

		return updateRelatedRepositories(modelName, fields, cfg)

	case types.AddFieldsAction:
		err := addFieldsToRepository(modelName, fields, cfg)
		if err != nil {
			return err
		}
		return updateRelatedRepositories(modelName, fields, cfg)

	case types.RemoveFieldsAction:
		return removeFieldsFromRepository(modelName, fields, cfg)

	case types.DropAction:
		return removeRepository(modelName, cfg)

	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func addFieldsToRepository(modelName string, newFields []types.Field, cfg *config.Config) error {
	filePath := getRepositoryFilePath(modelName, cfg)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading repository file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing repository file: %w", err)
	}

	var structDecl *ast.TypeSpec
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == modelName+"Repository" {
			structDecl = ts
			return false
		}
		return true
	})

	if structDecl == nil {
		return fmt.Errorf("repository struct not found")
	}

	for _, field := range newFields {
		if field.IsReference {
			methodName := "GetBy" + strcase.ToCamel(strings.TrimSuffix(field.Name, "_id"))

			templateData := map[string]interface{}{
				"ModelName":      modelName,
				"MethodName":     methodName,
				"FieldName":      field.Name,
				"FieldNameLower": strcase.ToLowerCamel(field.Name),
				"TableName":      inflection.Plural(strcase.ToSnake(modelName)),
			}

			newMethod, err := executeTemplate("repository_get_by_field.tmpl", templateData)
			if err != nil {
				return fmt.Errorf("error generating method for field %s: %w", field.Name, err)
			}

			content = append(content, []byte(newMethod)...)
		}
	}

	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		return fmt.Errorf("error writing updated repository file: %w", err)
	}

	return nil
}

func removeFieldsFromRepository(modelName string, fieldsToRemove []types.Field, cfg *config.Config) error {
	filePath := getRepositoryFilePath(modelName, cfg)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading repository file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing repository file: %w", err)
	}

	fieldsToRemoveSet := make(map[string]bool)
	for _, field := range fieldsToRemove {
		if field.IsReference {
			fieldsToRemoveSet[strcase.ToCamel(strings.TrimSuffix(field.Name, "_id"))] = true
		}
	}

	var newDecls []ast.Decl
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == modelName+"Repository" {
						methodName := funcDecl.Name.Name
						if strings.HasPrefix(methodName, "GetBy") && fieldsToRemoveSet[strings.TrimPrefix(methodName, "GetBy")] {
							continue
						}
					}
				}
			}
		}
		newDecls = append(newDecls, decl)
	}

	file.Decls = newDecls

	var buf bytes.Buffer
	err = format.Node(&buf, fset, file)
	if err != nil {
		return fmt.Errorf("error formatting updated repository file: %w", err)
	}

	err = os.WriteFile(filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing updated repository file: %w", err)
	}

	return nil
}

func getRepositoryFilePath(modelName string, cfg *config.Config) string {
	fileName := fmt.Sprintf("%s_repo.go", strcase.ToSnake(modelName))
	return filepath.Join(cfg.RepositoryDir, fileName)
}

func updateRelatedRepositories(modelName string, fields []types.Field, cfg *config.Config) error {
	for _, field := range fields {
		if field.IsReference {
			referencedModel := inflection.Singular(strcase.ToCamel(strings.TrimSuffix(field.Name, "_id")))

			// one-to-many
			err := updateRelatedRepository(referencedModel, modelName, field.Name, cfg)
			if err != nil {
				log.Printf("Warning: couldn't update related repository for %s: %v", referencedModel, err)
			}

			// many-to-one
			err = updateRelatedRepository(modelName, referencedModel, field.Name, cfg)
			if err != nil {
				log.Printf("Warning: couldn't update related repository for %s: %v", modelName, err)
			}
		}
	}
	return nil
}

func updateRelatedRepository(referencedModel, currentModel, fieldName string, cfg *config.Config) error {
	filePath := getRepositoryFilePath(referencedModel, cfg)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing repository file: %w", err)
	}

	interfaceName := referencedModel + "Repository"
	var interfaceType *ast.InterfaceType

	ast.Inspect(node, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == interfaceName {
			if ifaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				interfaceType = ifaceType
				return false
			}
		}
		return true
	})

	if interfaceType == nil {
		return fmt.Errorf("interface %s not found", interfaceName)
	}

	isOneToMany := strings.HasSuffix(fieldName, "_id")
	methodName, returnType := generateRelationMethodSignature(currentModel, isOneToMany)

	for _, field := range interfaceType.Methods.List {
		if field.Names[0].Name == methodName {
			return nil
		}
	}

	newMethod := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(methodName)},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ctx")},
						Type:  ast.NewIdent("context.Context"),
					},
					{
						Names: []*ast.Ident{ast.NewIdent(strcase.ToLowerCamel(referencedModel) + "ID")},
						Type:  ast.NewIdent("int64"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent(returnType),
					},
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
	}

	interfaceType.Methods.List = append(interfaceType.Methods.List, newMethod)

	// Получаем поля модели
	fields, err := getModelFields(currentModel, cfg)
	if err != nil {
		return fmt.Errorf("error getting model fields: %w", err)
	}

	// Генерируем реализацию метода с учетом полей модели
	implementation, err := generateRelationMethod(referencedModel, currentModel, isOneToMany, fields)
	if err != nil {
		return fmt.Errorf("error generating relation method: %w", err)
	}

	newFunc, err := parser.ParseFile(fset, "", "package repository\n\n"+implementation, 0)
	if err != nil {
		return fmt.Errorf("error parsing generated method: %w", err)
	}
	node.Decls = append(node.Decls, newFunc.Decls...)

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return fmt.Errorf("error writing updated AST: %w", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("error writing updated file: %w", err)
	}

	return nil
}

func generateRelationMethodSignature(currentModel string, isOneToMany bool) (methodName, returnType string) {
	if isOneToMany {
		methodName = "Get" + inflection.Plural(currentModel)
		returnType = fmt.Sprintf("[]*model.%s", currentModel)
	} else {
		methodName = "Get" + currentModel
		returnType = fmt.Sprintf("*model.%s", currentModel)
	}
	return methodName, returnType
}

func getModelFilePathForRepo(modelName string, cfg *config.Config) string {
	fileName := fmt.Sprintf("%s.go", strcase.ToSnake(modelName))
	return filepath.Join(cfg.ModelDir, fileName)
}

func executeTemplate(templateName string, data interface{}) (string, error) {
	tmpl, err := template.New(templateName).Funcs(template.FuncMap{
		"toLower":   strings.ToLower,
		"pluralize": inflection.Plural,
	}).ParseFiles(filepath.Join("templates", "repositories", templateName))
	if err != nil {
		return "", fmt.Errorf("error parsing template %s: %v", templateName, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("error executing template %s: %v", templateName, err)
	}

	return buf.String(), nil
}

func generateRelationMethod(referencedModel, currentModel string, isOneToMany bool, fields []types.Field) (string, error) {
	varName := strcase.ToLowerCamel(currentModel)
	tableName := inflection.Plural(strcase.ToSnake(currentModel))
	referencedTableName := inflection.Plural(strcase.ToSnake(referencedModel))
	returnType := "[]*model." + currentModel
	if !isOneToMany {
		returnType = "*model." + referencedModel
	}

	var query string
	if isOneToMany {
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s_id = $1", tableName, strcase.ToSnake(referencedModel))
	} else {
		query = fmt.Sprintf("SELECT %s.* FROM %s JOIN %s ON %s.%s_id = %s.id WHERE %s.id = $1",
			referencedTableName, tableName, referencedTableName,
			tableName, strcase.ToSnake(referencedModel), referencedTableName,
			tableName)
	}

	scanFields := make([]string, len(fields))
	for i, field := range fields {
		scanFields[i] = fmt.Sprintf("&%s.%s", varName, field.Name)
	}

	templateData := map[string]interface{}{
		"ReferencedModel": referencedModel,
		"CurrentModel":    currentModel,
		"VarName":         varName,
		"ReturnType":      returnType,
		"Query":           query,
		"IsOneToMany":     isOneToMany,
		"ScanFields":      strings.Join(scanFields, ", "),
	}

	templateName := "one_to_many_relation.tmpl"
	if !isOneToMany {
		templateName = "many_to_one_relation.tmpl"
	}

	return executeTemplate(templateName, templateData)
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

	type Relation struct {
		FieldName   string
		ModelName   string
		TableName   string
		VarName     string
		ScanValues  string
		ColumnNames string
		IsOneToMany bool
	}

	relations := make([]Relation, 0)

	for _, field := range fields {
		if field.IsReference {
			referencedModel := inflection.Singular(strcase.ToCamel(strings.TrimSuffix(field.Name, "_id")))

			filePath := getModelFilePathForRepo(referencedModel, cfg)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				log.Printf("Warning: model file for %s does not exist. Skipping relation.", referencedModel)
				continue
			}

			relationName := inflection.Plural(referencedModel)
			relatedTableName := inflection.Plural(strcase.ToSnake(referencedModel))
			varName := strcase.ToLowerCamel(referencedModel)

			relatedFields, err := getModelFields(referencedModel, cfg)
			if err != nil {
				log.Printf("Warning: couldn't get fields for model %s: %v. Skipping relation.", referencedModel, err)
				continue
			}

			scanValues := make([]string, len(relatedFields))
			columnNames := make([]string, len(relatedFields))
			for i, f := range relatedFields {
				scanValues[i] = fmt.Sprintf("&%s.%s", varName, f.Name)
				columnNames[i] = strcase.ToSnake(f.Name)
			}

			isOneToMany := !strings.HasSuffix(field.Name, "_id")

			relations = append(relations, Relation{
				FieldName:   relationName,
				ModelName:   referencedModel,
				TableName:   relatedTableName,
				VarName:     varName,
				ScanValues:  strings.Join(scanValues, ", "),
				ColumnNames: strings.Join(columnNames, ", "),
				IsOneToMany: isOneToMany,
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
		"Relations":         relations,
	}
}

func getModelFields(modelName string, cfg *config.Config) ([]types.Field, error) {
	filePath := getModelFilePath(modelName, cfg)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file does not exist: %s", filePath)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing file %s: %w", filePath, err)
	}

	var structDecl *ast.TypeSpec
	ast.Inspect(node, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == modelName {
			structDecl = ts
			return false
		}
		return true
	})

	if structDecl == nil {
		return nil, fmt.Errorf("struct %s not found in file %s", modelName, filePath)
	}

	structType, ok := structDecl.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct type", modelName)
	}

	var fields []types.Field
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			fieldType := ""
			if ident, ok := field.Type.(*ast.Ident); ok {
				fieldType = ident.Name
			}
			fields = append(fields, types.Field{
				Name: field.Names[0].Name,
				Type: fieldType,
			})
		}
	}

	return fields, nil
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
