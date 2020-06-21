package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaParsing(t *testing.T) {
	config := parseSchema([]byte(`{
  "log": {
    "level": { "format": "string", "default": "info", "env": "LOG_LEVEL" },
    "format": { "format": ["json", "text"], "default": "json", "env": "LOG_FORMAT" }
  }
}`))

	assert.Equal(t, []convictConfiguration{
		{Path: []string{"log", "format"}, Format: "json,text", DefaultValue: "json", Doc: "", Env: "LOG_FORMAT"},
		{Path: []string{"log", "level"}, Format: "string", DefaultValue: "info", Doc: "", Env: "LOG_LEVEL"},
	}, config.flatConfigurations, "")
}
