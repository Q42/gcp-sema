package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/fatih/color"
)

// This file defines all handlers `--from-[handler]=[key]=[value]`,

// SecretHandler is the shared interface common between all handlers:
// they can all populate values in a blob of secret data.
type SecretHandler interface {
	Populate(bucket map[string][]byte)
}

// MakeSecretHandler resolves the different kinds of handlers
func MakeSecretHandler(handler, name, value string) SecretHandler {
	switch handler {
	case "literal":
		return &literalHandler{key: name, value: value}
	case "file":
		return &fileHandler{key: name, file: value}
	case "sema-schema-to-file":
		return &semaHandlerSingleKey{key: name, configSchemaFile: value}
	case "sema-schema-to-literals":
		return &semaHandlerEnvironmentVariables{configSchemaFile: name}
	case "sema-literal":
		return &semaHandlerLiteral{key: name, secret: value}
	default:
		if value == "" {
			panic(fmt.Errorf("Could not parse --from-%s=%s", handler, name))
		} else {
			panic(fmt.Errorf("Could not parse --from-%s=%s=%s", handler, name, value))
		}
	}
}

type literalHandler struct {
	key   string
	value string
}

func (h *literalHandler) Populate(bucket map[string][]byte) {
	bucket[h.key] = []byte(h.value)
}

type unknownHandler struct {
	handler string
	key     string
	value   string
}

func (h *unknownHandler) Populate(bucket map[string][]byte) {
	panic("Not Implemented!")
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
	availableSecretKeys := getAllSecretsInProject()
	mapStrings(availableSecretKeys, trimPrefix) // otherwise they wont match during 'schemaResolveSecrets'

	schema := parseSchemaFile(h.configSchemaFile)
	allResolved := schemaResolveSecrets(schema, availableSecretKeys)

	// Shove it into a nested JSON structure
	jsonMap := hydrateSecretTree(schema.tree, allResolved)
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
	availableSecretKeys := getAllSecretsInProject()
	schema := parseSchemaFile(h.configSchemaFile)
	allResolved := schemaResolveSecrets(schema, availableSecretKeys)

	// Shove secrets in all possible environment variables
	for _, conf := range schema.flatConfigurations {
		key := strings.Join(conf.Path, ".")
		if r, isSet := allResolved[key]; isSet && conf.Env != "" {
			val := r.GetSecretValue()
			if val != nil {
				bucket[conf.Env] = []byte(*val)
				if Verbose {
					log.Println(color.BlueString("$%s=%s\n", conf.Env, val))
				}
			}
		}
	}
}

type semaHandlerLiteral struct {
	key    string
	secret string
}

func (h *semaHandlerLiteral) Populate(bucket map[string][]byte) {
	version := getLastSecretVersion(fmt.Sprintf("projects/%s/secrets/%s", GcloudProject, h.secret))
	bucket[h.key] = getSecretValue(version).Data
}

// If the input is a path like "a/long/path/to/something" the output is "something"
func trimPrefix(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}

// Convert slice in-place
func mapStrings(slice []string, fn func(string) string) []string {
	for i, v := range slice {
		slice[i] = fn(v)
	}
	return slice
}
