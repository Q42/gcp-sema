package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatStringFlatten(t *testing.T) {
	format := convictFormatString{actualFormat: "String", possibleValues: []string{}}
	flatten := func(val interface{}) string {
		dat, err := format.Flatten(val)
		if err != nil {
			t.Error(err)
			return ""
		}
		return dat
	}

	empty := ""
	assert.Equal(t, "", flatten(empty))
	assert.Equal(t, "", flatten(&empty))
	assert.Equal(t, "foobar", flatten("foobar"))

	_, err := format.Flatten(nil)
	assert.Equal(t, "not found", err.Error())
}
