package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Q42/gcp-sema/pkg/dynamic"
	"github.com/Q42/gcp-sema/pkg/schema"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/fatih/color"
)

// This file defines all handlers `--from-[handler]=[key]=[value]`,

// SecretHandler is the shared interface common between all handlers:
// they can all populate values in a blob of secret data.
type SecretHandler interface {
	Prepare(bucket map[string]bool)
	Populate(bucket map[string][]byte)
	Annotate(func(key string, value string))
	InjectClient(client secretmanager.KVClient)
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

// HandlerFactory needs to be implemented to expose new input formats
type HandlerFactory interface {
	ParseCommandline(args []string) (SecretHandler, error)
	ParseConfig(arg map[string]string) (SecretHandler, error)
}

type inlineFactory struct {
	parseCommandline func(args []string) (SecretHandler, error)
	parseConfig      func(arg map[string]string) (SecretHandler, error)
}

func (f inlineFactory) ParseCommandline(args []string) (SecretHandler, error) {
	return f.parseCommandline(args)
}

func (f inlineFactory) ParseConfig(arg map[string]string) (SecretHandler, error) {
	return f.parseConfig(arg)
}

// HandlerRegistry stores HandlerFactories
var HandlerRegistry map[string]HandlerFactory = map[string]HandlerFactory{
	"literal": inlineFactory{
		parseCommandline: func(args []string) (SecretHandler, error) {
			return &literalHandler{key: args[1], value: args[2]}, nil
		},
		parseConfig: func(input map[string]string) (SecretHandler, error) {
			return &literalHandler{key: input["name"], value: input["value"]}, nil
		},
	},
	"file": inlineFactory{
		parseCommandline: func(args []string) (SecretHandler, error) {
			return &fileHandler{key: args[1], file: args[2]}, nil
		},
		parseConfig: func(input map[string]string) (SecretHandler, error) {
			return &fileHandler{key: input["name"], file: input["value"]}, nil
		},
	},
	"sema-literal": inlineFactory{
		parseCommandline: func(args []string) (SecretHandler, error) {
			return &semaHandlerLiteral{key: args[1], secret: args[2]}, nil
		},
		parseConfig: func(input map[string]string) (SecretHandler, error) {
			return &semaHandlerLiteral{key: input["name"], secret: input["semaKey"]}, nil
		},
	},
}

// MakeSecretHandler resolves the different kinds of handlers
func MakeSecretHandler(handler, name, value string) (SecretHandler, error) {
	if factory, hasFactory := HandlerRegistry[handler]; hasFactory {
		return factory.ParseCommandline([]string{handler, name, value})
	}
	// Else, if factory is not defined
	if value == "" {
		return nil, fmt.Errorf("Could not parse --from-%s=%s", handler, name)
	}
	return nil, fmt.Errorf("Could not parse --from-%s=%s=%s", handler, name, value)
}

// ParseSecretHandler parses the different types of secret definitions into correct MakeSecretHandler calls
func ParseSecretHandler(input map[string]string) (SecretHandler, error) {
	defer func() {
		// Catch any panic errors from MakeSecretHandler
		if r := recover(); r != nil {
			panic(fmt.Errorf("Could not read handler from YAML configuration: %q", input))
		}
	}()
	if factory, hasFactory := HandlerRegistry[input["type"]]; hasFactory {
		return factory.ParseConfig(input)
	}
	return nil, fmt.Errorf("Could not parse handler config %v", input)
}

type semaHandlerSingleKey struct {
	key              string
	configSchemaFile string
	resolver         schema.SchemaResolver
	// private
	cacheSchema   schema.ConvictConfigSchema
	cacheResolved map[string]dynamic.ResolvedSecret
}

func (h *semaHandlerSingleKey) Prepare(bucket map[string]bool) {
	h.cacheSchema = schema.ParseSchemaFile(h.configSchemaFile)
	h.cacheResolved = h.resolver.Resolve(h.cacheSchema)
	bucket[h.key] = true
}
func (h *semaHandlerSingleKey) Populate(bucket map[string][]byte) {
	// Shove it into a nested JSON structure
	jsonMap, err := hydrateSecretTree(h.cacheSchema.Tree, h.cacheResolved)
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

	if h.resolver.IsVerbose() {
		log.Println(color.BlueString("Generated value for key '%s':\n%s\n", h.key, string(jsonData)))
	}
	bucket[h.key] = jsonData
}
func (h *semaHandlerSingleKey) Annotate(annotate func(key string, value string)) {
	annotate(h.key, fmt.Sprintf("type=sema-schema-to-file,schema=%s", h.configSchemaFile))
	for secretName, resolved := range h.cacheResolved {
		annotate(fmt.Sprintf("%s.%s", h.key, secretName), resolved.Annotation())
	}
}
func (h *semaHandlerSingleKey) InjectClient(c secretmanager.KVClient) {
	// TODO
}

type semaHandlerEnvironmentVariables struct {
	configSchemaFile string
	resolver         schema.SchemaResolver
	// private
	cacheSchema   schema.ConvictConfigSchema
	cacheResolved map[string]dynamic.ResolvedSecret
}

func (h *semaHandlerEnvironmentVariables) Prepare(bucket map[string]bool) {
	h.cacheSchema = schema.ParseSchemaFile(h.configSchemaFile)
	h.cacheResolved = h.resolver.Resolve(h.cacheSchema)
	for _, conf := range h.cacheSchema.FlatConfigurations {
		key := conf.Key()
		if _, isSet := h.cacheResolved[key]; isSet && conf.Env != "" {
			bucket[conf.Env] = true
		}
	}
}

func (h *semaHandlerEnvironmentVariables) Populate(bucket map[string][]byte) {
	var allErrors error
	// Shove secrets in all possible environment variables
	for _, conf := range h.cacheSchema.FlatConfigurations {
		key := conf.Key()
		if r, isSet := h.cacheResolved[key]; isSet && conf.Env != "" {
			val, err := r.GetSecretValue()
			if stringVal, ok := val.(*string); ok {
				bucket[conf.Env] = []byte(*stringVal)
				if h.resolver.IsVerbose() {
					log.Println(color.BlueString("$%s=%s\n", conf.Env, val))
				}
			}
			allErrors = multiAppend(allErrors, err)
		}
	}
	panicIfErr(allErrors)
}
func (h *semaHandlerEnvironmentVariables) Annotate(annotate func(key string, value string)) {
	annotate("", fmt.Sprintf("type=sema-schema-to-literals,schema=%s", h.configSchemaFile))
	for secretName, resolved := range h.cacheResolved {
		annotate(secretName, resolved.Annotation())
	}
}

func (h *semaHandlerEnvironmentVariables) InjectClient(c secretmanager.KVClient) {
	// TODO
}

type semaHandlerLiteral struct {
	key      string
	secret   string
	resolver schema.SchemaResolver
	//private
	cacheResolved schema.ResolvedSecretSema
}

func (h *semaHandlerLiteral) Prepare(bucket map[string]bool) {
	secret, err := h.resolver.GetClient().Get(h.secret)
	panicIfErr(err)
	h.cacheResolved = schema.ResolvedSecretSema{Key: h.secret, Client: h.resolver.GetClient(), KV: secret}
	bucket[h.key] = true
}
func (h *semaHandlerLiteral) Populate(bucket map[string][]byte) {
	val, err := h.cacheResolved.GetSecretValue()
	panicIfErr(err)
	if stringVal, ok := val.(*string); ok {
		bucket[h.key] = []byte(*stringVal)
	}
}
func (h *semaHandlerLiteral) Annotate(annotate func(key string, value string)) {
	annotate(h.key, fmt.Sprintf("type=sema-literal,secret=%s", h.secret))
	annotate(fmt.Sprintf("%s.%s", h.key, alfanum(h.secret)), h.cacheResolved.Annotation())
}
func (h *semaHandlerLiteral) InjectClient(c secretmanager.KVClient) {
	// TODO
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
