package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaParsing(t *testing.T) {
	config, err := parseSchema([]byte(`{
  "log": {
    "doc": "Some properties here regarding logging",
    "level": { "format": "String", "default": "info", "env": "LOG_LEVEL" },
    "format": { "format": ["json", "text"], "default": "json", "env": "LOG_FORMAT" },
    "invalidKeyIsIgnored": null
  }
}`))
	assert.Equal(t, nil, err)

	logFormat := convictFormatString{actualFormat: []interface{}{"json", "text"}, possibleValues: []string{"json", "text"}}
	logLevel := convictFormatString{actualFormat: "String"}

	assert.Equal(t, []convictConfiguration{
		{Path: []string{"log", "format"}, Format: logFormat, DefaultValue: "json", Doc: "", Env: "LOG_FORMAT"},
		{Path: []string{"log", "level"}, Format: logLevel, DefaultValue: "info", Doc: "", Env: "LOG_LEVEL"},
	}, config.flatConfigurations, "")
}
