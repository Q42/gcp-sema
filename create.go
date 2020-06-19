package main

import (
	"os"
	"regexp"

	"github.com/go-errors/errors"
	flags "github.com/jessevdk/go-flags"
)

// The 'sema create' command
func create(args []string) {
	opts := parseCreateArgs(args)

	// Preamble, depending on the format
	if opts.Format == "" || opts.Format == "yaml" {
		log.Println(`kind: Secret
apiVersion: v1
metadata:
  name: mysecret
type: Opaque
data:`)
	}

	// Give all handlers a go to write to the secret data
	data := make(map[string][]byte, 0)
	for _, h := range opts.Handlers {
		h.Populate(data)
	}

	// Print all values in the correct format
	for key, value := range data {
		switch opts.Format {
		case "env":
			log.Printf("%s=%s", key, string(value))
		default:
			log.Printf("  %s: %s", key, string(value))
		}
	}
}

// Usage describes how to use the create command
type Usage struct {
	Project  string          `positional-args:"0" description:"Google Cloud project"`
	Verbose  []bool          `short:"v" long:"verbose" description:"Show verbose debug information"`
	Format   string          `short:"o" long:"format" default:"yaml" description:"How to output: 'yaml' is a fully specified Kubernetes secret, 'env' will generate a *.env file format that can be used for Docker (Compose)."`
	Handlers []SecretHandler `no-flag:"y" description:"multiple ways to specify a secret source, the format is --from-[handler]=[key]=[source/value]"`
}

func parseCreateArgs(args []string) Usage {
	opts := Usage{}
	parser := flags.NewParser(&opts, flags.Default)

	// formats of cli-arg is "{reArgName}{reArgValue}"
	reArgName := regexp.MustCompile(`from-([^=]+)`)
	reArgValue := regexp.MustCompile(`([^=]+)(=([^=]+))?`)

	// UnknownOptionHandler parses all --from-... arguments into opts.Handlers
	parser.UnknownOptionHandler = func(option string, arg flags.SplitArgument, args []string) (nextArgs []string, outErr error) {
		value, hasValue := arg.Value()
		if matchedKey := reArgName.FindStringSubmatch(option); len(matchedKey) == 2 && hasValue {
			if matchedValue := reArgValue.FindStringSubmatch(value); len(matchedValue) > 2 {
				defer func() {
					// Catch any panic errors down the line
					if r := recover(); r != nil {
						outErr = errors.New(r)
						return
					}
				}()
				handler := MakeSecretHandler(matchedKey[1], matchedValue[1], matchedValue[3])
				opts.Handlers = append(opts.Handlers, handler)
				return args, nil
			}
		}
		return args, nil
	}

	// Do it
	_, err := parser.ParseArgs(args)
	if err != nil {
		os.Exit(1)
	}
	return opts
}
