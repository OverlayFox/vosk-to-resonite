package resonite

type Command struct {
	Type  CommandType
	Value float64
	Name  string
}

type CommandType string

const (
	CommandTypeGrow   CommandType = "grow"
	CommandTypeShrink CommandType = "shrink"
)
