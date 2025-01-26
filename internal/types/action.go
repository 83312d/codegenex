package types

type Action string

const (
	CreateAction       Action = "create"
	AddFieldsAction    Action = "add_fields"
	RemoveFieldsAction Action = "remove_fields"
	DropAction         Action = "drop"
	UnknownAction      Action = "unknown"
)

func (a Action) String() string {
	switch a {
	case CreateAction:
		return "create"
	case AddFieldsAction:
		return "add_fields"
	case RemoveFieldsAction:
		return "remove_fields"
	case DropAction:
		return "drop"
	default:
		return "unknown"
	}
}
