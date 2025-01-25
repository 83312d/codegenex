package parser

import (
	"strings"

	"github.com/yourusername/codegenex/internal/types"
)

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
