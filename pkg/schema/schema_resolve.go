package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Q42/gcp-sema/pkg/handlers"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/fatih/color"
)

// SchemaResolver -
type SchemaResolver interface {
	Resolve(schema ConvictConfigSchema) map[string]handlers.ResolvedSecret
	IsVerbose() bool
	GetClient() secretmanager.KVClient
}

type schemaResolver struct {
	Client  secretmanager.KVClient
	Prefix  string
	Verbose bool
	Matcher Matcher
	// private
	cachedAvailable []secretmanager.KVValue
}

// MakeSchemaResolver -
func MakeSchemaResolver(client secretmanager.KVClient, prefix string, verbose bool, matcher Matcher) SchemaResolver {
	return schemaResolver{Client: client, Prefix: prefix, Verbose: verbose, Matcher: matcher}
}

// IsVerbose -
func (r schemaResolver) IsVerbose() bool {
	return r.Verbose
}

// GetClient -
func (r schemaResolver) GetClient() secretmanager.KVClient {
	return r.Client
}

// Matcher -
type Matcher = func(ConvictConfiguration, secretmanager.KVValue, string) bool

// DefaultMatcher matches the secret with the key based on the SecretManager short-name
var DefaultMatcher Matcher = func(c ConvictConfiguration, s secretmanager.KVValue, key string) bool {
	return s.GetShortName() == key
}

// ConvictToSemaKey translates the name to something that is appropriate for SeMa storage
func ConvictToSemaKey(prefix string, path []string) (result []string) {
	if prefix != "" {
		result = append(result, strings.Join(append([]string{strings.ToLower(prefix)}, path...), "_"))
	}
	result = append(result, strings.Join(path, "_"))
	for i, v := range result {
		// Sema only allows keys to start with a lowercase character, so lets use only lowercase chars to be consistent!
		// We are a little more flexible with reading from Secret Manager, so we support old secret uploads that are not in this same case-format.
		result[i] = strings.ToLower(v)
	}
	return
}

func (r schemaResolver) resolveConf(conf ConvictConfiguration, availableSecrets []secretmanager.KVValue) (result handlers.ResolvedSecret, options []handlers.ResolvedSecret, err error) {
	// enumerate all places we want to look for this secret
	suggestedKeys := ConvictToSemaKey(r.Prefix, conf.Path)

	if r.Matcher == nil {
		r.Matcher = DefaultMatcher
	}

	for _, suggestedKey := range suggestedKeys {
		options = append(options, handlers.ResolvedSecretSema{Key: suggestedKey, Client: r.Client, KV: nil})
	}
	runtimeOpts := makeRuntimeResolve(conf)
	options = append(options, runtimeOpts...)

	// Here the keynames in Secret Manager are checked against the keys that are required by config-json
	for _, suggestedKey := range suggestedKeys {
		// enumerate all secrets that we have set in SecretManager
		for _, available := range availableSecrets {
			// if it matches, return it
			if r.Matcher(conf, available, suggestedKey) {
				return handlers.ResolvedSecretSema{Key: suggestedKey, Client: r.Client, KV: available}, options, nil
			}
		}
	}

	if len(runtimeOpts) > 0 {
		return runtimeOpts[0], options, nil
	}
	return nil, options, semaNotFoundError{conf, suggestedKeys}
}

// quick and dirty equality
func resolvedSecretEqual(a, b handlers.ResolvedSecret) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return (a.String()) == (b.String())
}

type resolvedSecretRuntime struct{ conf ConvictConfiguration }

func makeRuntimeResolve(conf ConvictConfiguration) []handlers.ResolvedSecret {
	if conf.DefaultValue != nil || conf.Env != "" || conf.Format.IsOptional() {
		return []handlers.ResolvedSecret{resolvedSecretRuntime{conf: conf}}
	}
	return nil
}

func (r resolvedSecretRuntime) Annotation() string {
	return r.String()
}

func (r resolvedSecretRuntime) String() string {
	opts := make([]string, 0)
	if r.conf.Env != "" {
		opts = append(opts, fmt.Sprintf("env: $%s", r.conf.Env))
	}
	if r.conf.DefaultValue != nil {
		data, _ := json.Marshal(r.conf.DefaultValue)
		opts = append(opts, fmt.Sprintf("default: %s", string(data)))
	}
	return fmt.Sprintf("runtime(%s)", strings.Join(opts, " or "))
}

func (r resolvedSecretRuntime) GetSecretValue() (interface{}, error) {
	return nil, nil // injected runtime
}

type semaNotFoundError struct {
	conf          ConvictConfiguration
	suggestedKeys []string
}

func (e semaNotFoundError) Is(err error) bool {
	_, isSameType := err.(semaNotFoundError)
	return isSameType
}

func (e semaNotFoundError) Error() string {
	return fmt.Sprintf("%s; Secret Manager keys: %q", e.conf.Key(), e.suggestedKeys)
}

// private function to ease testing with mock data
func (r schemaResolver) Resolve(schema ConvictConfigSchema) map[string]handlers.ResolvedSecret {
	if r.Verbose {
		log.Println(color.BlueString("SecretManager verbose output"))
	}

	// Get/cache available secrets: reused by multiple invocations
	if r.cachedAvailable == nil {
		var err error
		r.cachedAvailable, err = r.Client.ListKeys()
		panicIfErr(err)
	}

	// Resolve all configuration options
	allErrors := make([]error, 0)
	allResolved := make(map[string]handlers.ResolvedSecret, 0)
	for _, conf := range schema.FlatConfigurations {
		resolved, options, err := r.resolveConf(conf, r.cachedAvailable)
		if err != nil {
			allErrors = append(allErrors, err)
		} else {
			allResolved[conf.Key()] = resolved
		}
		if r.Verbose {
			log.Println(color.BlueString("%s:", conf.Key()))
			for _, option := range options {
				if resolvedSecretEqual(option, resolved) {
					log.Println(color.GreenString("- %s", option))
				} else {
					log.Println(color.BlueString("- %s", option))
				}
			}
		}
	}

	if r.Verbose {
		log.Println()
	}

	if len(allErrors) > 0 {
		log.Println(color.RedString("No secret value resolved for:"))
		for _, err := range allErrors {
			log.Println(color.RedString("- %s", err.Error()))
			if errors.As(semaNotFoundError{}, &err) {
				nf := err.(semaNotFoundError)
				if nf.conf.Format != nil {
					log.Println(color.RedString("  format: %s", nf.conf.Format.String()))
				}
				if nf.conf.Doc != "" {
					log.Println(color.RedString("  doc: %s", nf.conf.Doc))
				}
			}
		}
		log.Println()
	}
	return allResolved
}
