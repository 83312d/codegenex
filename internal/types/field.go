package types

type Field struct {
	Name            string
	Type            string
	IsIndex         bool
	IsReference     bool
	RefOptions      string
	IsNullable      bool
	DefaultValue    string
	ReferencedModel string
	IsEnum          bool
	EnumValues      []string
}
