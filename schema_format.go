package main

import (
	"fmt"
	"strconv"
	"strings"
)

type convictFormat interface {
	// Coerce will take a string (like environment variable) and convert it into a JSON value
	Coerce(input string) (interface{}, error)
	String() string
}

type convictFormatString struct {
	actualFormat   interface{}
	possibleValues []string
}

func (f convictFormatString) Coerce(input string) (interface{}, error) {
	if stringFmt, isString := f.actualFormat.(string); isString && strings.Contains(stringFmt, "optional") && input == "" {
		return nil, nil
	}
	if len(f.possibleValues) > 0 {
		for _, possible := range f.possibleValues {
			if possible == input {
				return input, nil
			}
		}
		return nil, fmt.Errorf("Invalid value '%s' for format %v", input, f.possibleValues)
	}
	return input, nil
}

func (f convictFormatString) String() string {
	if len(f.possibleValues) > 0 {
		return "[" + strings.Join(f.possibleValues, ",") + "]"
	}
	return fmt.Sprintf("%v", f.actualFormat)
}

type convictFormatArray struct {
}

func (f convictFormatArray) Coerce(input string) (interface{}, error) {
	return strings.Split(input, ","), nil
}
func (f convictFormatArray) String() string {
	return "Array"
}

type convictFormatPort struct{}
type convictFormatBoolean struct{}
type convictFormatInt struct{ actualFormat string }

func (f convictFormatPort) Coerce(input string) (interface{}, error) {
	return strconv.ParseInt(input, 10, 16)
}
func (f convictFormatBoolean) Coerce(input string) (interface{}, error) {
	return strconv.ParseBool(input)
}
func (f convictFormatInt) Coerce(input string) (interface{}, error) {
	return strconv.ParseInt(input, 10, 64)
}

func (f convictFormatPort) String() string {
	return "port"
}
func (f convictFormatBoolean) String() string {
	return "Boolean"
}
func (f convictFormatInt) String() string {
	return f.actualFormat
}
