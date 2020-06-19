package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseArgs(t *testing.T) {
	args := parseArgs([]string{
		"--format=env",
		"--from-[handler]=[key]=[source]",
		"--from-literal=myfile.txt=literal-value",
		// plain files just like kubectl create secret
		"--from-file=myfile.txt=myfile.txt",
		// extract according to schema into a single property 'config-env.json'
		"--from-sema-schema-to-file=config-env.json=config-schema.json",
		// extract according to schema into environment variable literals
		"--from-sema-schema-to-literals=config-schema.json",
		// extract key value from SeMa into literals
		"--from-sema-literal=MY_APP_SECRET=MY_APP_SECRET_NEW",
	})

	assert.Equal(t, Usage{
		Format: "env",
		Handlers: []*SecretHandler{
			MakeSecretHandler("[handler]", "[key]", "[source]"),
			MakeSecretHandler("literal", "myfile.txt", "literal-value"),
			MakeSecretHandler("file", "myfile.txt", "myfile.txt"),
			MakeSecretHandler("sema-schema-to-file", "config-env.json", "config-schema.json"),
			MakeSecretHandler("sema-literal", "MY_APP_SECRET", "MY_APP_SECRET_NEW"),
		},
	}, args, "Arguments must be parsed correctly")
}
