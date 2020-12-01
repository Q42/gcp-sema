package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const exampleSchema = `{
  "log": {
    "doc": "Some properties here regarding logging",
    "level": { "format": "String", "default": "info", "env": "LOG_LEVEL" },
    "format": { "format": ["json", "text"], "default": "json", "env": "LOG_FORMAT" },
    "invalidKeyIsIgnored": null
  },
  "redis": {
    "shards": { "default": null, "format": "Array", "doc": "bla", "env": "REDIS_SHARDS" },
    "port": { "default": [6379], "format": "Array" }
  }
}`

func TestSchemaParsing(t *testing.T) {
	config, err := parseSchema([]byte(exampleSchema))
	assert.Equal(t, nil, err)

	logFormat := convictFormatString{actualFormat: []interface{}{"json", "text"}, possibleValues: []string{"json", "text"}}
	logLevel := convictFormatString{actualFormat: "String"}
	arr := convictFormatArray{}

	assert.Equal(t, []ConvictConfiguration{
		{Path: []string{"log", "format"}, Format: logFormat, DefaultValue: "json", Doc: "", Env: "LOG_FORMAT"},
		{Path: []string{"log", "level"}, Format: logLevel, DefaultValue: "info", Doc: "", Env: "LOG_LEVEL"},
		{Path: []string{"redis", "port"}, Format: arr, DefaultValue: []interface{}{float64(6379)}, Doc: "", Env: ""},
		{Path: []string{"redis", "shards"}, Format: arr, DefaultValue: nil, Doc: "bla", Env: "REDIS_SHARDS"},
	}, config.FlatConfigurations, "")
}
