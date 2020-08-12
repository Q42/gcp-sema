package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRenderArgs(t *testing.T) {
	// Test we can parse all the different source formats from the README.md
	// format: "--from-[handler]=[key]=[source]"
	args := parseRenderArgs([]string{
		"my-project",
		"--name=very-secret",
		// literals just like kubectl create secret
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

	expected := RenderCommand{
		Format: "yaml",
		Name:   "very-secret",
		Dir:    "secrets",
		Handlers: []SecretHandler{
			MakeSecretHandler("literal", "myfile.txt", "literal-value"),
			MakeSecretHandler("file", "myfile.txt", "myfile.txt"),
			MakeSecretHandler("sema-schema-to-file", "config-env.json", "config-schema.json"),
			MakeSecretHandler("sema-schema-to-literals", "config-schema.json", ""),
			MakeSecretHandler("sema-literal", "MY_APP_SECRET", "MY_APP_SECRET_NEW"),
		},
	}
	expected.Positional.Project = "my-project"

	assert.Equal(t, expected, args, "Arguments must be parsed correctly")

}

func TestRenderLiteral(t *testing.T) {
	obj := make(map[string][]byte, 0)
	args := parseRenderArgs([]string{"my-project", "--format=env", "--from-literal=text.txt=foobar"})
	args.Handlers[0].Populate(obj)
	assert.Equal(t, []byte("foobar"), obj["text.txt"], "Literal SecretHandler should work")
}

func TestRenderFormat(t *testing.T) {
	args := parseRenderArgs([]string{"my-project", "--format=env"})
	assert.Equal(t, "env", args.Format, "Should parse formats")
	args = parseRenderArgs([]string{"--format=yaml", "my-project"})
	assert.Equal(t, "yaml", args.Format, "Should parse formats")
	args = parseRenderArgs([]string{"my-project"})
	assert.Equal(t, "yaml", args.Format, "Should parse formats")
	args = parseRenderArgs([]string{"--format=files", "my-project"})
	assert.Equal(t, "files", args.Format, "Should parse formats")
}
