package main

import (
	"fmt"
	"testing"

	"github.com/Q42/gcp-sema/pkg/handlers"
	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
)

func TestParseRenderArgs(t *testing.T) {
	// Test we can parse all the different source formats from the README.md
	// format: "-s [handler]=[key]=[source]"
	args := parseRenderArgs([]string{
		"my-project",
		"--name=very-secret",
		// literals just like kubectl create secret
		"-s literal=myfile.txt=literal-value",
		// plain files just like kubectl create secret
		"-s file=myfile.txt=myfile.txt",
		// extract according to schema into a single property 'config-env.json'
		"-s sema-schema-to-file=config-env.json=config-schema.json",
		// extract according to schema into environment variable literals
		"-s sema-schema-to-literals=config-schema.json",
		// extract key value from SeMa into literals
		"-s sema-literal=MY_APP_SECRET=MY_APP_SECRET_NEW",
		"test",
	})

	expected := RenderCommand{
		Format: "yaml",
		Name:   "very-secret",
		Dir:    "secrets",
		Handlers: []handlers.ConcreteSecretHandler{
			{SecretHandler: makeSecretWrapper("literal", "myfile.txt", "literal-value")},
			{SecretHandler: makeSecretWrapper("file", "myfile.txt", "myfile.txt")},
			{SecretHandler: makeSecretWrapper("sema-schema-to-file", "config-env.json", "config-schema.json")},
			{SecretHandler: makeSecretWrapper("sema-schema-to-literals", "config-schema.json", "")},
			{SecretHandler: makeSecretWrapper("sema-literal", "MY_APP_SECRET", "MY_APP_SECRET_NEW")},
		},
	}
	expected.Positional.Project = "my-project"

	assert.Equal(t, expected, args, "Arguments must be parsed correctly")

}

func TestRenderLiteral(t *testing.T) {
	obj := make(map[string][]byte, 0)
	args := parseRenderArgs([]string{"my-project", "--format=env", "-s literal=text.txt=foobar"})
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

func TestParseSecretConfig(t *testing.T) {
	config := fmt.Sprintf(`
name: myapp1-v4
prefix: myapp1_v4
secrets:
- path: config-env.json
  name: config-env.json
  schema: "server/config-schema.json"
  type: sema-schema-to-file`)
	parsedConfig := parseConfigFileData([]byte(config))
	expected := RenderCommand{
		Name:   "myapp1-v4",
		Prefix: "myapp1_v4",
		Handlers: []handlers.ConcreteSecretHandler{
			{SecretHandler: makeSecretWrapper("sema-schema-to-file", "config-env.json", "server/config-schema.json")},
		},
	}
	assert.Equal(t, expected, parsedConfig, "Configfile must be parsed correctly")
}

func TestMergeConfig(t *testing.T) {
	// Mock data config
	config := fmt.Sprintf(`
name: myapp1-v4
prefix: myapp1_v4
secrets:
- path: config-env.json
  name: config-env.json
  schema: "server/config-schema.json"
  type: sema-schema-to-file`)
	// Mock cmd arguments
	args := []string{
		"render",
		"--name=very-secret",
		// literals just like kubectl create secret
		"-s literal=myfile.txt=literal-value",
		// plain files just like kubectl create secret
		"-s file=myfile.txt=myfile.txt",
		// extract according to schema into a single property 'config-env.json'
		"-s sema-schema-to-file=config-env.json=config-schema.json",
		// extract according to schema into environment variable literals
		"-s sema-schema-to-literals=config-schema.json",
		// extract key value from SeMa into literals
		"-s sema-literal=MY_APP_SECRET=MY_APP_SECRET_NEW",
		"my-project",
		"test",
	}

	// Setup flag parser
	opts := RenderCommand{}
	p := flags.NewParser(&struct{}{}, flags.Default)

	cmd, err := p.AddCommand("render", renderDescription, renderDescriptionLong, &opts)
	panicIfErr(err)
	parsedConfig := parseConfigFileData([]byte(config))
	_, err = p.ParseArgs(args)

	opts.mergeCommandOptions(cmd, parsedConfig)

	expected := RenderCommand{
		Name:   "very-secret",
		Prefix: "myapp1_v4",
		Format: "yaml",
		Dir:    "secrets",
		Handlers: []handlers.ConcreteSecretHandler{
			{SecretHandler: makeSecretWrapper("literal", "myfile.txt", "literal-value")},
			{SecretHandler: makeSecretWrapper("file", "myfile.txt", "myfile.txt")},
			{SecretHandler: makeSecretWrapper("sema-schema-to-file", "config-env.json", "config-schema.json")},
			{SecretHandler: makeSecretWrapper("sema-schema-to-literals", "config-schema.json", "")},
			{SecretHandler: makeSecretWrapper("sema-literal", "MY_APP_SECRET", "MY_APP_SECRET_NEW")},
			{SecretHandler: makeSecretWrapper("sema-schema-to-file", "config-env.json", "server/config-schema.json")},
		},
	}
	expected.Positional.Project = "my-project"
	assert.Equal(t, expected, opts, "Config and command line options should be merged correctly")

}

func makeSecretWrapper(handler string, name string, value string) handlers.SecretHandler {
	secretHandler, err := handlers.MakeSecretHandler(handler, name, value)
	panicIfErr(err)
	return secretHandler
}
