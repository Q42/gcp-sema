package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestAnnotationsValid(t *testing.T) {
	semaSingleKey := &semaHandlerSingleKey{key: "config-schema.json", configSchemaFile: "server/config-schema.json"}
	semaSingleKey.cacheResolved = map[string]resolvedSecret{
		"ENCRYPTION.BRIDGE.SALT": resolvedSecretRuntime{},
		"ENCRYPTION_BRIDGE_SALT": resolvedSecretRuntime{},
	}

	handlers := map[string]SecretHandler{
		"literal":                 &literalHandler{key: "foo_bla", value: "bar"},
		"file":                    &fileHandler{key: "foo_bla", file: "my/random/file.txt", data: nil},
		"sema-schema-to-file":     semaSingleKey,
		"sema-schema-to-literals": &semaHandlerEnvironmentVariables{configSchemaFile: "config-schema.json"},
		"sema-literal":            &semaHandlerLiteral{key: "foo_bla", secret: "random;txt!"},
	}

	for n, h := range handlers {
		t.Run(n, func(t *testing.T) {
			ann := map[string]string{}
			h.Annotate(func(key, value string) { key, value, _ = postProcessAnnotation(key, value); ann[key] = value })
			for key := range ann {
				assert.Empty(t, validation.IsQualifiedName(key), "testing %q", key)
				assert.LessOrEqual(t, len(key), 63, "testing %q", key)
			}
		})
	}

	var key string
	var ok bool

	key, _, ok = postProcessAnnotation("more_than_63_chars_foo_bla_lorum_ipsum_dolor_amet________bla_lorum_ipsum_dolor_amet", "")
	assert.LessOrEqual(t, len(key), 63, "%q should be 63 chars at most")
	assert.Equal(t, key, "sema/source.more_than_63_chars_foo_bla_lorum_ipsum_dolor_amet", "%q should be correctly trimmed")
	assert.True(t, ok, "should be feasible to convert")

	key, _, ok = postProcessAnnotation("________________________________________________________________________________", "")
	assert.LessOrEqual(t, len(key), 63, "%q should be 63 chars at most")
	assert.Equal(t, key, "sema/source", "%q should be correctly trimmed")
	assert.True(t, ok, "should be feasible to convert")

	key, _, ok = postProcessAnnotation("", "")
	assert.LessOrEqual(t, len(key), 63, "%q should be 63 chars at most")
	assert.Equal(t, key, "sema/source", "%q should be correctly trimmed")
	assert.True(t, ok, "should be feasible to convert")
}
