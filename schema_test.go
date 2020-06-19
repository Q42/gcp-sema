package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaParsing(t *testing.T) {
	config := parseSchema([]byte(`{
  "log": {
    "level": { "format": "string", "default": "info", "env": "LOG_LEVEL" },
    "format": { "format": "string", "default": "json", "env": "LOG_FORMAT" }
  }
}`))

	assert.Equal(t, []convictConfiguration{
		{path: []string{"log", "level"}, format: "string", defaultValue: "info", doc: "", env: "LOG_LEVEL"},
		{path: []string{"log", "format"}, format: "string", defaultValue: "json", doc: "", env: "LOG_FORMAT"},
	}, config.flatConfigurations, "")
}
