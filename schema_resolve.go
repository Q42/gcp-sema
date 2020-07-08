package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

func schemaResolveSecret(conf convictConfiguration, availableSecretKeys []string) (result resolvedSecret, options []resolvedSecret, err error) {
	// enumerate all places we want to look for this secret
	suggestedKeys := convictToSemaKey(RenderPrefix, conf.Path)

	for _, suggestedKey := range suggestedKeys {
		options = append(options, resolvedSecretSema{key: suggestedKey})
	}
	runtimeOpts := makeRuntimeResolve(conf)
	options = append(options, runtimeOpts...)

	for _, suggestedKey := range suggestedKeys {
		// enumerate all secrets that we have set in SecretManager
		for _, availableKey := range availableSecretKeys {
			// if it matches, return it
			if availableKey == suggestedKey {
				return resolvedSecretSema{key: suggestedKey}, options, nil
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
	GetSecretValue() *string
}
type resolvedSecretRuntime struct{ conf convictConfiguration }

func makeRuntimeResolve(conf convictConfiguration) []resolvedSecret {
	if conf.DefaultValue != nil || conf.Env != "" {
		return []resolvedSecret{resolvedSecretRuntime{conf: conf}}
	}
	return nil
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

func (r resolvedSecretRuntime) GetSecretValue() *string {
	return nil // injected runtime
}

type resolvedSecretSema struct{ key string }

func (r resolvedSecretSema) String() string {
	return fmt.Sprintf("secretmanager(key: %s)", r.key)
}

func (r resolvedSecretSema) GetSecretValue() *string {
	version := getLastSecretVersion(fmt.Sprintf("projects/%s/secrets/%s", GcloudProject, r.key))
	val := getSecretValue(version)
	stringValue := string(val.Data)
	return &stringValue
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
	return fmt.Sprintf("%s; Secret Manager keys: %q", strings.Join(e.conf.Path, "."), e.suggestedKeys)
}

// private function to ease testing with mock data
func schemaResolveSecrets(schema convictConfigSchema, availableSecretKeys []string) map[string]resolvedSecret {
	if Verbose {
		log.Println(color.BlueString("SecretManager verbose output"))
	}

	// Resolve all configuration options
	allErrors := make([]error, 0)
	allResolved := make(map[string]resolvedSecret, 0)
	for _, conf := range schema.flatConfigurations {
		resolved, options, err := schemaResolveSecret(conf, availableSecretKeys)
		if err != nil {
			allErrors = append(allErrors, err)
		} else {
			allResolved[strings.Join(conf.Path, ".")] = resolved
		}
		if Verbose {
			log.Println(color.BlueString("%s:", strings.Join(conf.Path, ".")))
			for _, option := range options {
				if resolvedSecretEqual(option, resolved) {
					log.Println(color.GreenString("- %s", option))
				} else {
					log.Println(color.BlueString("- %s", option))
				}
			}
		}
	}

	if Verbose {
		log.Println()
	}

	if len(allErrors) > 0 {
		log.Println(color.RedString("No secret value resolved for:"))
		for _, err := range allErrors {
			log.Println(color.RedString("- %s", err.Error()))
			if errors.Is(err, semaNotFoundError{}) {
				log.Println(color.RedString("  format: %s", err.(semaNotFoundError).conf.Format.String()))
			}
			if errors.Is(err, semaNotFoundError{}) && err.(semaNotFoundError).conf.Doc != "" {
				log.Println(color.RedString("  doc: %s", err.(semaNotFoundError).conf.Doc))
			}
		}
		log.Println()
	}
	return allResolved
}
