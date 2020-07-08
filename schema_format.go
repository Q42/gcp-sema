package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type convictFormat interface {
	// Coerce will take a string (like environment variable) and convert it into a JSON value
	Coerce(input string) (interface{}, error)
	Flatten(input interface{}) (string, error)
	String() string
}

type convictFormatAny struct{}

func (f convictFormatAny) String() string {
	return "*"
}

func (f convictFormatAny) Coerce(input string) (interface{}, error) {
	// TODO: where do we use this; how will it be handled in the app?
	// this is dangerous, as we wont coerce an arraylike "a,b,c" into an array for example
	return input, nil
}

func (f convictFormatAny) Flatten(input interface{}) (string, error) {
	// TODO: where do we use this; how will it be handled in the app?
	return fmt.Sprint(input), nil
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

func (f convictFormatString) Flatten(input interface{}) (string, error) {
	return fmt.Sprint(input), nil
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
func (f convictFormatArray) Flatten(input interface{}) (string, error) {
	switch v := input.(type) {
	case []interface{}:
		vals := make([]string, 0)
		for _, i := range v {
			vals = append(vals, fmt.Sprint(i))
		}
		return strings.Join(vals, ","), nil
	case []string:
		return strings.Join(v, ","), nil
	default:
		return "", fmt.Errorf("Unsupported array type: %q", reflect.TypeOf(input))
	}
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

func (f convictFormatPort) Flatten(input interface{}) (string, error) {
	return fmt.Sprint(input), nil
}
func (f convictFormatBoolean) Flatten(input interface{}) (string, error) {
	return strconv.FormatBool(input.(bool)), nil
}
func (f convictFormatInt) Flatten(input interface{}) (string, error) {
	return fmt.Sprint(input), nil
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
