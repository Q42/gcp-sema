package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
)

// This file defines all handlers `--from-[handler]=[key]=[value]`,

// SecretHandler is the shared interface common between all handlers:
// they can all populate values in a blob of secret data.
type SecretHandler interface {
	Prepare(bucket map[string]bool)
	Populate(bucket map[string][]byte)
	Annotate(func(key string, value string))
}

// ConcreteSecretHandler is a way to implement the Unmarshaller (UnmarshalFlag) interface from go-flags on the interface type SecretHandler.
type ConcreteSecretHandler struct {
	SecretHandler
}

// UnmarshalFlag -
func (c *ConcreteSecretHandler) UnmarshalFlag(value string) error {
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

// InlineFactory -
type InlineFactory struct {
	parseCommandline func(args []string) (SecretHandler, error)
	parseConfig      func(arg map[string]string) (SecretHandler, error)
}

// MakeInlineFactory -
func MakeInlineFactory(argsToMap func(arg []string) (map[string]string, error), mapToSecret func(arg map[string]string) (SecretHandler, error)) HandlerFactory {
	return InlineFactory{
		parseCommandline: func(args []string) (SecretHandler, error) {
			mp, err := argsToMap(args)
			if err != nil {
				return nil, err
			}
			return mapToSecret(mp)
		},
		parseConfig: func(arg map[string]string) (SecretHandler, error) {
			return mapToSecret(arg)
		},
	}
}

// ParseCommandline -
func (f InlineFactory) ParseCommandline(args []string) (SecretHandler, error) {
	return f.parseCommandline(args)
}

// ParseConfig -
func (f InlineFactory) ParseConfig(arg map[string]string) (SecretHandler, error) {
	return f.parseConfig(arg)
}

// HandlerRegistry stores HandlerFactories
var HandlerRegistry map[string]HandlerFactory = map[string]HandlerFactory{
	"literal": MakeInlineFactory(func(arg []string) (map[string]string, error) {
		return map[string]string{"name": arg[1], "value": arg[2], "type": "literal"}, nil
	}, func(input map[string]string) (SecretHandler, error) {
		return &literalHandler{key: input["name"], value: input["value"]}, nil
	}),

	"file": MakeInlineFactory(func(arg []string) (map[string]string, error) {
		return map[string]string{"name": arg[1], "path": arg[2], "type": "file"}, nil
	}, func(input map[string]string) (SecretHandler, error) {
		return &fileHandler{key: input["name"], file: input["path"]}, nil
	}),

	"sema-literal": MakeInlineFactory(func(arg []string) (map[string]string, error) {
		return map[string]string{"name": arg[1], "semaKey": arg[2], "type": "sema-literal"}, nil
	}, func(input map[string]string) (SecretHandler, error) {
		return &semaHandlerLiteral{key: input["name"], secret: input["semaKey"]}, nil
	}),
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

// SecretHandlerWithSema implement this interface to get a SemaClient injected
type SecretHandlerWithSema interface {
	InjectSemaClient(client secretmanager.KVClient, opts SecretHandlerOptions)
}

// SecretHandlerOptions -
type SecretHandlerOptions struct {
	Prefix  string
	Mock    bool
	Verbose bool
}

// InjectSemaClient -
func InjectSemaClient(handlers []ConcreteSecretHandler, client secretmanager.KVClient, opts SecretHandlerOptions) (returned []ConcreteSecretHandler) {
	for _, h := range handlers {
		if h.SecretHandler == nil {
			continue
		}
		if sh, isInjectable := h.SecretHandler.(SecretHandlerWithSema); isInjectable {
			sh.InjectSemaClient(client, opts)
		}
	}
	return
}

// func (c handlers.ConcreteSecretHandler) injectSemaClient(schemaResolver SchemaResolver) handlers.ConcreteSecretHandler {
// 	switch s := c.SecretHandler.(type) {
// 	case *semaHandlerEnvironmentVariables:
// 		s.resolver = schemaResolver
// 		c.SecretHandler = s
// 	case *semaHandlerLiteral:
// 		s.resolver = schemaResolver
// 		c.SecretHandler = s
// 	case *semaHandlerSingleKey:
// 		s.resolver = schemaResolver
// 		c.SecretHandler = s
// 	}
// 	return c
// }
