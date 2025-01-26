package parser

import (
	"codegenex/internal/types"
	"strings"
)

func ParseAction(action string) types.Action {
	switch action {
	case "create":
		return types.CreateAction
	case "add_fields":
		return types.AddFieldsAction
	case "remove_fields":
		return types.RemoveFieldsAction
	case "drop":
		return types.DropAction
	default:
		return types.UnknownAction
	}
}

func ParseFields(args []string) []types.Field {
	fields := make([]types.Field, 0, len(args))
	for _, arg := range args {
		field := parseField(arg)
		fields = append(fields, field)
	}
	return fields
}

func parseField(arg string) types.Field {
	parts := strings.Split(arg, ":")
	field := types.Field{Name: parts[0]}

	if len(parts) > 1 {
		field.Type = parts[1]
		if strings.HasPrefix(field.Type, "enum[") && strings.HasSuffix(field.Type, "]") {
			field.IsEnum = true
			enumValues := strings.TrimPrefix(strings.TrimSuffix(field.Type, "]"), "enum[")
			if enumValues != "" {
				field.EnumValues = strings.Split(enumValues, ",")
			}
			field.Type = "string"
		}
	}

	for i := 2; i < len(parts); i++ {
		option := parts[i]
		switch {
		case option == "i":
			field.IsIndex = true
		case option == "unique":
			field.IsUnique = true
		case strings.HasPrefix(option, "ref"):
			field.IsReference = true
			refParts := strings.Split(option, "=")
			if len(refParts) > 1 {
				field.RefOptions = refParts[1]
			} else {
				field.RefOptions = "cascade"
			}
		case option == "null":
			field.IsNullable = true
		case strings.HasPrefix(option, "default="):
			field.DefaultValue = strings.TrimPrefix(option, "default=")
		}
	}

	return field
}
