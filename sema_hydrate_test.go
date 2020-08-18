package main

import (
	"encoding/json"
	"testing"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
)

func TestHydratedPopulate(t *testing.T) {

	// nil
	var tree *convictJSONTree
	result, err := hydrateSecretTree(tree, map[string]resolvedSecret{})
	assert.Equal(t, nil, err)
	assert.Equal(t, nil, result)

	// simple tree
	schema, err := parseSchema([]byte(`{
    /* logging config */
    "LOG_FORMAT": { "format": ["json", "text"], "default": "json", "doc": "How to log" },
    "LOG_LEVEL": { "format": ["warn", "error"], "default": "error", "doc": "When to log" },
}`))
	client = secretmanager.NewMockClient("my-project", "log_level", "info")
	result, err = hydrateSecretTree(schema.tree, map[string]resolvedSecret{
		"LOG_FORMAT": resolvedSecretRuntime{*schema.tree.Children["LOG_FORMAT"].Leaf},
		"LOG_LEVEL":  resolvedSecretSema{key: "log_level"},
	})
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	assert.Equal(t, nil, err)
	assert.Equal(t, `{
  "LOG_LEVEL": "info"
}`, string(jsonData))
}
