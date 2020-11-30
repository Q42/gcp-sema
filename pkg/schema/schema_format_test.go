package schema

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

func TestFormatArrayFlatten(t *testing.T) {
	format := convictFormatArray{}
	flatten := func(val interface{}) string {
		dat, err := format.Flatten(val)
		if err != nil {
			t.Error(err)
			return ""
		}
		return dat
	}

	value := "foo"
	assert.Equal(t, "foo,bar", flatten([]interface{}{"foo", "bar"}), "Expected to work for array with string values")
	assert.NotEqual(t, "foo,bar", flatten([]interface{}{&value, "bar"}), "Known bug: does not work for array with pointers to strings")
	assert.Equal(t, "baz", flatten("baz"), "Expected to work for single value")
	_, err := format.Flatten(nil)
	assert.Equal(t, "not found", err.Error())
}
