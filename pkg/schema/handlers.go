package schema

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Q42/gcp-sema/pkg/handlers"
	"github.com/Q42/gcp-sema/pkg/multierror"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/fatih/color"
)

// Register schema handlers
func init() {
	handlers.HandlerRegistry["sema-schema-to-file"] = handlers.MakeInlineFactory(func(arg []string) (map[string]string, error) {
		return map[string]string{"name": arg[1], "schema": arg[2], "type": "sema-schema-to-file"}, nil
	}, func(input map[string]string) (handlers.SecretHandler, error) {
		return &semaHandlerSingleKey{key: input["name"], configSchemaFile: input["schema"]}, nil
	})

	handlers.HandlerRegistry["sema-schema-to-literals"] = handlers.MakeInlineFactory(func(arg []string) (map[string]string, error) {
		return map[string]string{"schema": arg[1], "type": "sema-schema-to-literals"}, nil
	}, func(input map[string]string) (handlers.SecretHandler, error) {
		return &semaHandlerEnvironmentVariables{configSchemaFile: input["schema"]}, nil
	})
}

type semaHandlerSingleKey struct {
	key              string
	configSchemaFile string
	resolver         SchemaResolver
	// private
	cacheSchema   ConvictConfigSchema
	cacheResolved map[string]handlers.ResolvedSecret
}

type semaHandlerEnvironmentVariables struct {
	configSchemaFile string
	resolver         SchemaResolver
	// private
	cacheSchema   ConvictConfigSchema
	cacheResolved map[string]handlers.ResolvedSecret
}

/* Test they conforms to interfaces */
var _ handlers.SecretHandler = &semaHandlerSingleKey{}
var _ handlers.SecretHandlerWithSema = &semaHandlerSingleKey{}
var _ handlers.SecretHandler = &semaHandlerEnvironmentVariables{}
var _ handlers.SecretHandlerWithSema = &semaHandlerEnvironmentVariables{}

/* Implement SecretHanderWithSema methods */
func (h *semaHandlerSingleKey) InjectSemaClient(client secretmanager.KVClient, opts handlers.SecretHandlerOptions) {
	if opts.Mock {
		h.resolver = &CatchAllResolver{}
		return
	}
	h.resolver = MakeSchemaResolver(client, opts.Prefix, opts.Verbose, DefaultMatcher)
}
func (h *semaHandlerEnvironmentVariables) InjectSemaClient(client secretmanager.KVClient, opts handlers.SecretHandlerOptions) {
	if opts.Mock {
		h.resolver = &CatchAllResolver{}
		return
	}
	h.resolver = MakeSchemaResolver(client, opts.Prefix, opts.Verbose, DefaultMatcher)
}

/* Implement SecretHandler methods */
func (h *semaHandlerSingleKey) Prepare(bucket map[string]bool) {
	h.cacheSchema = ParseSchemaFile(h.configSchemaFile)
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

func (h *semaHandlerEnvironmentVariables) Prepare(bucket map[string]bool) {
	h.cacheSchema = ParseSchemaFile(h.configSchemaFile)
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
			allErrors = multierror.MultiAppend(allErrors, err)
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
