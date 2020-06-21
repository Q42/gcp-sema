package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"

	"github.com/go-errors/errors"
	flags "github.com/jessevdk/go-flags"
)

var (
	// formats of cli-arg is "{reArgName}{reArgValue}"
	reArgName     = regexp.MustCompile(`from-([^=]+)`)
	reArgValue    = regexp.MustCompile(`([^=]+)(=([^=]+))?`)
	createCommand = &CreateCommand{}
)

var createDescription = `Create combines the Secret Manager data and generates the output that can be applied to Kubernetes or Docker Compose.`
var createDescriptionLong = `Create combines the Secret Manager data and generates the output that can be applied to Kubernetes or Docker Compose.

There are multiple ways to specify a secret source, the format is --from-[handler]=[key]=[source/value].
The following options are implemented:

  # literals just like kubectl create secret --from-literal=myfile.txt=foo-bar
  --from-literal=myfile.txt=foo-bar

  # plain files just like kubectl create secret --from-file=myfile.txt=./myfile.txt
  --from-file=myfile.txt=./myfile.txt

  # extract according to schema into a single property 'config-env.json'
  --from-sema-schema-to-file=config-env.json=config-schema.json

  # extract according to schema into environment variable literals
  --from-sema-schema-to-literals=config-schema.json

  # extract key value from SeMa into literals
  --from-sema-literal=MY_APP_SECRET=MY_APP_SECRET_NEW
`

func init() {
	_, err := parser.AddCommand("create", createDescription, createDescriptionLong, createCommand)
	panicIfErr(err)
	parser.UnknownOptionHandler = cliParseFromHandlers
}

// Execute of CreateCommand is the 'sema create' command
func (opts *CreateCommand) Execute(args []string) error {
	GcloudProject = opts.Positional.Project

	// Give all handlers a go to write to the secret data
	data := make(map[string][]byte, 0)
	for _, h := range opts.Handlers {
		h.Populate(data)
	}

	// Preamble, depending on the format
	if opts.Format == "yaml" {
		os.Stdout.WriteString(`kind: Secret
apiVersion: v1
metadata:
  name: mysecret
type: Opaque
data:
`)
	}

	// Print all values in the correct format
	for key, value := range data {
		switch opts.Format {
		case "env":
			os.Stdout.WriteString(fmt.Sprintf("%s=%s\n", key, string(value)))
		default:
			os.Stdout.WriteString(fmt.Sprintf("  %s: %s\n", key, base64.StdEncoding.EncodeToString([]byte(value))))
		}
	}
	return nil
}

// CreateCommand describes how to use the create command
type CreateCommand struct {
	Positional struct {
		Project string `required:"yes" description:"Google Cloud project" positional-arg-name:"project"`
	} `positional-args:"yes"`
	Verbose  []bool          `short:"v" long:"verbose" description:"Show verbose debug information"`
	Format   string          `short:"o" long:"format" default:"yaml" description:"How to output: 'yaml' is a fully specified Kubernetes secret, 'env' will generate a *.env file format that can be used for Docker (Compose)."`
	Prefix   string          `long:"prefix" description:"A SecretManager prefix that will override non-prefixed keys"`
	Handlers []SecretHandler `no-flag:"y"`
}

// For testing, repeatably executable
func parseCreateArgs(args []string) CreateCommand {
	opts := CreateCommand{}
	parser := flags.NewParser(&opts, flags.Default)
	parser.UnknownOptionHandler = cliParseFromHandlers

	// Do it
	createCommand.Handlers = []SecretHandler{}
	_, err := parser.ParseArgs(args)
	opts.Handlers = createCommand.Handlers
	if err != nil {
		os.Exit(1)
	}
	return opts
}

// UnknownOptionHandler parses all --from-... arguments into opts.Handlers
func cliParseFromHandlers(option string, arg flags.SplitArgument, args []string) (nextArgs []string, outErr error) {
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
			createCommand.Handlers = append(createCommand.Handlers, handler)
			return args, nil
		}
	}
	return args, nil
}
