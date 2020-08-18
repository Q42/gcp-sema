package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/fatih/color"
)

// This file defines all handlers `--from-[handler]=[key]=[value]`,

// SecretHandler is the shared interface common between all handlers:
// they can all populate values in a blob of secret data.
type SecretHandler interface {
	Populate(bucket map[string][]byte)
}

// concreteSecretHandler is a way to implement the Unmarshaller (UnmarshalFlag) interface from go-flags on the interface type SecretHandler.
type concreteSecretHandler struct {
	SecretHandler
}

func (c *concreteSecretHandler) UnmarshalFlag(value string) error {
	// parse value
	args := strings.SplitN(strings.TrimSpace(value), "=", 3)
	var err error
	switch len(args) {
	case 2:
		c.SecretHandler, err = MakeSecretHandler(args[0], args[1], "")
	case 3:
		c.SecretHandler, err = MakeSecretHandler(args[0], args[1], args[2])
	default:
		return errors.New("--secrets array options should contain 2 or 3 values")
	}
	return err
}

type unstructuredHandler struct {
	Type  string
	Key   string
	Value string
}

// MakeSecretHandler resolves the different kinds of handlers
func MakeSecretHandler(handler, name, value string) (SecretHandler, error) {
	switch handler {
	case "literal":
		return &literalHandler{key: name, value: value}, nil
	case "file":
		return &fileHandler{key: name, file: value}, nil
	case "sema-schema-to-file":
		return &semaHandlerSingleKey{key: name, configSchemaFile: value}, nil
	case "sema-schema-to-literals":
		return &semaHandlerEnvironmentVariables{configSchemaFile: name}, nil
	case "sema-literal":
		return &semaHandlerLiteral{key: name, secret: value}, nil
	default:
		if value == "" {
			return nil, fmt.Errorf("Could not parse --from-%s=%s", handler, name)
		}
		return nil, fmt.Errorf("Could not parse --from-%s=%s=%s", handler, name, value)
	}
}

// ParseSecretHandler parses the different types of secret definitions into correct MakeSecretHandler calls
func ParseSecretHandler(input map[string]string) (SecretHandler, error) {
	defer func() {
		// Catch any panic errors from MakeSecretHandler
		if r := recover(); r != nil {
			panic(fmt.Errorf("Could not read handler from YAML configuration: %q", input))
		}
	}()
	switch input["type"] {
	case "sema-schema-to-file":
		return MakeSecretHandler(input["type"], input["name"], input["schema"])
	case "sema-literal":
		return MakeSecretHandler(input["type"], input["name"], input["semaKey"])
	case "file":
		return MakeSecretHandler(input["type"], input["name"], input["path"])
	default:
		return MakeSecretHandler(input["type"], input["name"], input["value"])
	}
}

type literalHandler struct {
	key   string
	value string
}

func (h *literalHandler) Populate(bucket map[string][]byte) {
	bucket[h.key] = []byte(h.value)
}

type fileHandler struct {
	key  string
	file string
}

func (h *fileHandler) Populate(bucket map[string][]byte) {
	data, err := ioutil.ReadFile(h.file)
	panicIfErr(err)
	bucket[h.key] = data
}

type semaHandlerSingleKey struct {
	key              string
	configSchemaFile string
}

func (h *semaHandlerSingleKey) Populate(bucket map[string][]byte) {
	availableSecrets, err := client.ListKeys()
	availableSecretKeys := secretmanager.SecretShortNames(availableSecrets)
	panicIfErr(err)

	mapStrings(availableSecretKeys, trimPathPrefix) // otherwise they wont match during 'schemaResolveSecrets'

	schema := parseSchemaFile(h.configSchemaFile)
	allResolved := schemaResolveSecrets(schema, availableSecretKeys)

	// Shove it into a nested JSON structure
	jsonMap, err := hydrateSecretTree(schema.tree, allResolved)
	if err != nil {
		panic(err)
	}
	if jsonMap == nil {
		// if the whole tree is empty, still return an empty JSON object
		jsonMap = make(map[string]interface{}, 0)
	}
	jsonData, err := json.MarshalIndent(jsonMap, "", "  ")
	if err != nil {
		panic(err)
	}

	if Verbose {
		log.Println(color.BlueString("Generated value for key '%s':\n%s\n", h.key, string(jsonData)))
	}
	bucket[h.key] = jsonData
}

type semaHandlerEnvironmentVariables struct {
	configSchemaFile string
}

func (h *semaHandlerEnvironmentVariables) Populate(bucket map[string][]byte) {
	availableSecrets, err := client.ListKeys()
	availableSecretKeys := secretmanager.SecretShortNames(availableSecrets)
	panicIfErr(err)

	schema := parseSchemaFile(h.configSchemaFile)
	allResolved := schemaResolveSecrets(schema, availableSecretKeys)

	var allErrors error
	// Shove secrets in all possible environment variables
	for _, conf := range schema.flatConfigurations {
		key := conf.Key()
		if r, isSet := allResolved[key]; isSet && conf.Env != "" {
			val, err := r.GetSecretValue()
			if stringVal, ok := val.(*string); ok {
				bucket[conf.Env] = []byte(*stringVal)
				if Verbose {
					log.Println(color.BlueString("$%s=%s\n", conf.Env, val))
				}
			}
			allErrors = multiAppend(allErrors, err)
		}
	}
	panicIfErr(allErrors)
}

type semaHandlerLiteral struct {
	key    string
	secret string
}

func (h *semaHandlerLiteral) Populate(bucket map[string][]byte) {
	secret, err := client.Get(h.secret)
	panicIfErr(err)
	data, err := secret.GetValue()
	bucket[h.key] = data
}

// If the input is a path like "a/long/path/to/something" the output is "something"
func trimPathPrefix(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}

// Convert slice in-place
func mapStrings(slice []string, fn func(string) string) []string {
	for i, v := range slice {
		slice[i] = fn(v)
	}
	return slice
}
