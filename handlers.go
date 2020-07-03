package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

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
	bucket[h.key] = []byte(base64.StdEncoding.EncodeToString(jsonData))
}

type semaHandlerEnvironmentVariables struct {
	configSchemaFile string
}

func (h *semaHandlerEnvironmentVariables) Populate(bucket map[string][]byte) {
	// TODO read config-schema.json, find a secret for each key / or rely on defaults
	// Put each key / env in their own literal
	bucket["LOG_LEVEL"] = []byte("info")
	bucket["PORT"] = []byte("8080")
	panic("Not implemented!")
}

type semaHandlerLiteral struct {
	key    string
	secret string
}

func (h *semaHandlerLiteral) Populate(bucket map[string][]byte) {
	version := getLastSecretVersion(fmt.Sprintf("projects/%s/secrets/%s", GcloudProject, h.secret))
	bucket[h.key] = getSecretValue(version).Data
}
