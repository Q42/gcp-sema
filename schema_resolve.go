package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/fatih/color"
)

type schemaResolver struct {
	Client  secretmanager.KVClient
	Prefix  string
	Verbose bool
	// private
	cachedAvailable []secretmanager.KVValue
}

func (r schemaResolver) resolveConf(conf convictConfiguration, availableSecrets []secretmanager.KVValue) (result resolvedSecret, options []resolvedSecret, err error) {
	// enumerate all places we want to look for this secret
	suggestedKeys := convictToSemaKey(r.Prefix, conf.Path)

	for _, suggestedKey := range suggestedKeys {
		options = append(options, resolvedSecretSema{key: suggestedKey, client: r.Client, kv: nil})
	}
	runtimeOpts := makeRuntimeResolve(conf)
	options = append(options, runtimeOpts...)

	// Here the keynames in Secret Manager are checked against the keys that are required by config-schema.json
	for _, suggestedKey := range suggestedKeys {
		// enumerate all secrets that we have set in SecretManager
		for _, available := range availableSecrets {
			// if it matches, return it
			if available.GetShortName() == suggestedKey {
				return resolvedSecretSema{key: suggestedKey, client: r.Client, kv: available}, options, nil
			}
		}
	}

	if len(runtimeOpts) > 0 {
		return runtimeOpts[0], options, nil
	}
	return nil, options, semaNotFoundError{conf, suggestedKeys}
}

// quick and dirty equality
func resolvedSecretEqual(a, b resolvedSecret) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return (a.String()) == (b.String())
}

type resolvedSecret interface {
	String() string
	Annotation() string
	GetSecretValue() (interface{}, error)
}
type resolvedSecretRuntime struct{ conf convictConfiguration }

func makeRuntimeResolve(conf convictConfiguration) []resolvedSecret {
	if conf.DefaultValue != nil || conf.Env != "" || conf.Format.IsOptional() {
		return []resolvedSecret{resolvedSecretRuntime{conf: conf}}
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

type resolvedSecretSema struct {
	key    string
	client secretmanager.KVClient
	kv     secretmanager.KVValue
}

func (r resolvedSecretSema) Annotation() string {
	if r.kv != nil {
		return fmt.Sprintf("secretmanager(fullname: %s)", r.kv.GetFullName())
	}
	return r.String()
}

func (r resolvedSecretSema) String() string {
	return fmt.Sprintf("secretmanager(key: %s)", r.key)
}

func (r resolvedSecretSema) GetSecretValue() (interface{}, error) {
	var err error
	secret := r.kv
	if secret == nil {
		secret, err = r.client.Get(r.key)
		if err != nil {
			return nil, err
		}
	}
	val, err := secret.GetValue()
	if err != nil {
		return nil, err
	}
	stringValue := string(val)
	return &stringValue, nil
}

type semaNotFoundError struct {
	conf          convictConfiguration
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
func (r schemaResolver) Resolve(schema convictConfigSchema) map[string]resolvedSecret {
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
	allResolved := make(map[string]resolvedSecret, 0)
	for _, conf := range schema.flatConfigurations {
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
