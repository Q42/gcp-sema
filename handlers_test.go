package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

/**
 * Prevent errors like:
 * - metadata.annotations: Invalid value: "sema/source/config-env.json/CACHE.EXPIRATION": a qualified name must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]') with an optional DNS subdomain prefix and '/' (e.g. 'example.com/MyName')
 **/
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
			h.Annotate(ann)
			for key := range ann {
				t.Log("testing", key)
				assert.Empty(t, validation.IsQualifiedName(key))
			}
		})
	}

}
