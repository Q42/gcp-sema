package schema

import (
	"encoding/json"
	"testing"

	"github.com/Q42/gcp-sema/pkg/handlers"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
)

func TestHydrateNil(t *testing.T) {
	var tree *ConvictJSONTree
	result, err := hydrateSecretTree(tree, map[string]handlers.ResolvedSecret{})
	assert.Equal(t, nil, err)
	assert.Equal(t, nil, result)
}

func TestHydrateFlatTree(t *testing.T) {
	client := secretmanager.NewMockClient("my-project", "log_level", "warn")

	schema, err := parseSchema([]byte(`{
    /* logging config */
    "LOG_FORMAT": { "format": ["json", "text"], "default": "json", "doc": "How to log" },
    "LOG_LEVEL": { "format": ["warn", "error"], "default": "error", "doc": "When to log" },
}`))
	result, err := hydrateSecretTree(schema.tree, map[string]handlers.ResolvedSecret{
		"LOG_FORMAT": resolvedSecretRuntime{*schema.tree.Children["LOG_FORMAT"].Leaf},
		"LOG_LEVEL":  handlers.ResolvedSecretSema{key: "log_level", client: client},
	})
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	assert.Equal(t, nil, err)
	assert.Equal(t, `{
  "LOG_LEVEL": "warn"
}`, string(jsonData))

}

func TestHydrateNestedTree(t *testing.T) {
	client := secretmanager.NewMockClient("my-project", "logging_level", "warn")

	schema, err := parseSchema([]byte(`{
    /* logging config */
    "LOGGING": {
      "FORMAT": { "format": ["json", "text"], "default": "json", "doc": "How to log" },
      "LEVEL": { "format": ["warn", "error"], "default": "error", "doc": "When to log" },
    }
}`))
	assert.Equal(t, nil, err)
	assert.NotNil(t, schema.tree.Children["LOGGING"], "LOGGING")
	assert.NotNil(t, schema.tree.Children["LOGGING"].Children["FORMAT"], "FORMAT")
	assert.NotNil(t, schema.tree.Children["LOGGING"].Children["LEVEL"], "LEVEL")

	// One is runtime, other is resolved
	resolved := schemaResolver{Client: client, Verbose: true}.Resolve(schema)
	assert.IsType(t, resolvedSecretRuntime{}, resolved["LOGGING.FORMAT"], "LOGGING.FORMAT")
	assert.IsType(t, handlers.ResolvedSecretSema{}, resolved["LOGGING.LEVEL"], "LOGGING.LEVEL")
	assert.Equal(t, client, resolved["LOGGING.LEVEL"].(handlers.ResolvedSecretSema).client, "LOGGING.LEVEL")

	result, err := hydrateSecretTree(schema.tree.Children["LOGGING"].Children["FORMAT"], resolved)
	assert.Equal(t, nil, result)

	result, err = hydrateSecretTree(schema.tree, resolved)
	levelValue, err := resolved["LOGGING.LEVEL"].GetSecretValue()
	assert.Equal(t, map[string]interface{}(map[string]interface{}{"LOGGING": map[string]interface{}{"LEVEL": levelValue}}), result)
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	assert.Equal(t, nil, err)
	assert.Equal(t, `{
  "LOGGING": {
    "LEVEL": "warn"
  }
}`, string(jsonData))
}
