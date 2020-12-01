package main

import (
	"testing"

	"github.com/Q42/gcp-sema/pkg/handlers"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestAnnotationsValid(t *testing.T) {
	mapping := map[string]handlers.SecretHandler{
		"literal":                 makeSecretWrapper("literal", "foo_bla", "bar"),
		"file":                    makeSecretWrapper("file", "foo_bla", "my/random/file.txt"),
		"sema-schema-to-file":     makeSecretWrapper("sema-schema-to-file", "config-schema.json", "server/config-schema.json"),
		"sema-schema-to-literals": makeSecretWrapper("sema-schema-to-literals", "config-schema.json", ""),
		"sema-literal":            makeSecretWrapper("sema-literal", "foo_bla", "random;txt!"),
	}

	mockClient := secretmanager.NewMockClient("dummy")
	for n, h := range mapping {
		if n, isInjectable := h.(handlers.SecretHandlerWithSema); isInjectable {
			n.InjectSemaClient(mockClient, handlers.SecretHandlerOptions{})
		}

		t.Run(n, func(t *testing.T) {
			ann := map[string]string{}
			h.Annotate(func(key, value string) { key, value, _ = postProcessAnnotation(key, value); ann[key] = value })
			for key := range ann {
				t.Logf("Annotation %q, value %q", key, ann[key])
				assert.Empty(t, validation.IsQualifiedName(key), "testing %q", key)
				assert.LessOrEqual(t, len(key), 63, "testing %q", key)
			}
		})
	}

}

func TestAnnotationPostprocessing(t *testing.T) {
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
