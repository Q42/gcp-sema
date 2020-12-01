package multierror

import (
	"errors"
	"strings"
)

// MultiError -
type MultiError struct {
	Errors []error
}

// MultiAppend -
func MultiAppend(a error, b error) error {
	var multiA MultiError
	var multiB MultiError
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if errors.As(a, &multiA) && errors.As(b, &multiB) {
		return MultiError{Errors: append(multiA.Errors, multiB.Errors...)}
	}
	if errors.As(a, &multiA) {
		return MultiError{Errors: append(multiA.Errors, b)}
	}
	if errors.As(b, &multiB) {
		return MultiError{Errors: append(multiB.Errors, a)}
	}
	return MultiError{Errors: []error{a, b}}
}

func (me MultiError) Error() string {
	txts := []string{}
	for _, e := range me.Errors {
		txts = append(txts, "- "+e.Error())
	}
	return "Multiple errors:\n" + strings.Join(txts, "\n")
}
