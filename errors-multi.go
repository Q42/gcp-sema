package main

import (
	"errors"
	"strings"
)

type multiError struct {
	Errors []error
}

func multiAppend(a error, b error) error {
	var multiA multiError
	var multiB multiError
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if errors.As(a, &multiA) && errors.As(b, &multiB) {
		return multiError{Errors: append(multiA.Errors, multiB.Errors...)}
	}
	if errors.As(a, &multiA) {
		return multiError{Errors: append(multiA.Errors, b)}
	}
	if errors.As(b, &multiB) {
		return multiError{Errors: append(multiB.Errors, a)}
	}
	return multiError{Errors: []error{a, b}}
}

func (me multiError) Error() string {
	txts := []string{}
	for _, e := range me.Errors {
		txts = append(txts, "- "+e.Error())
	}
	return "Multiple errors:\n" + strings.Join(txts, "\n")
}
