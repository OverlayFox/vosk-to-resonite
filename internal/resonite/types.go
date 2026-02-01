package resonite

import (
	"fmt"
	"strings"
)

type Command struct {
	Type  CommandType
	Value float64
	Unit  CommandUnit
}

func init() {
	for key := range StringToCommandTypeMap {
		StringToCommandTypeList = append(StringToCommandTypeList, key)
	}

	for key := range StringToCommandUnitMap {
		StringToCommandUnitList = append(StringToCommandUnitList, key)
	}

	for expr := range ExpressionToPercentage {
		Expressions = append(Expressions, expr)
	}
}

func (c Command) ToCommandString() string {
	switch c.Type {
	case CommandTypeUndefined:
		return ""
	case CommandTypeGrow, CommandTypeShrink:
		if c.Value == 0 {
			return ""
		}

		if c.Unit == CommandUnitTimes {
			c.Unit = CommandUnitPercent
			c.Value = c.Value * 100
		}

		return fmt.Sprintf("%s-%.6f-%s", c.Type, c.Value, c.Unit)
	}
	return ""
}

type CommandType string

const (
	CommandTypeUndefined CommandType = "undefined"
	CommandTypeGrow      CommandType = "grow"
	CommandTypeShrink    CommandType = "shrink"
	CommandTypeSelect    CommandType = "select"
)

var (
	StringToCommandTypeMap = map[string]CommandType{
		"grows":   CommandTypeGrow,
		"grow":    CommandTypeGrow,
		"shrink":  CommandTypeShrink,
		"shrinks": CommandTypeShrink,
		"select":  CommandTypeSelect,
		"selects": CommandTypeSelect,
	}
	StringToCommandTypeList = []string{}
)

func StringToCommandType(s string) CommandType {
	if unit, ok := StringToCommandTypeMap[strings.ToLower(s)]; ok {
		return unit
	}
	return CommandTypeUndefined
}

type CommandUnit string

const (
	CommandUnitPercent     CommandUnit = "percent"
	CommandUnitCentimeters CommandUnit = "centimeters"
	CommandUnitMeters      CommandUnit = "meters"
	CommandUnitInches      CommandUnit = "inches"
	CommandUnitTimes       CommandUnit = "times"
)

var (
	StringToCommandUnitMap = map[string]CommandUnit{
		"centimeter":  CommandUnitCentimeters,
		"centimeters": CommandUnitCentimeters,
		"meter":       CommandUnitMeters,
		"meters":      CommandUnitMeters,
		"inch":        CommandUnitInches,
		"inches":      CommandUnitInches,
		"percent":     CommandUnitPercent,
		"per cent":    CommandUnitPercent,
		"times":       CommandUnitTimes,
		"time":        CommandUnitTimes,
	}
	StringToCommandUnitList = []string{}
)

func StringToCommandUnit(s string) CommandUnit {
	if unit, ok := StringToCommandUnitMap[strings.ToLower(s)]; ok {
		return unit
	}
	return CommandUnitPercent
}

var (
	ExpressionToPercentage = map[string]float64{
		"quarter":   25.0,
		"third":     33.33,
		"half":      50.0,
		"double":    200.0,
		"triple":    300.0,
		"quadruple": 400.0,
	}
	Expressions = []string{}
)

func ExpressionToPercent(expr string) (float64, bool) {
	if val, ok := ExpressionToPercentage[strings.ToLower(expr)]; ok {
		return val, true
	}
	return 0, false
}
