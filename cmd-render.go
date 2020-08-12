package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/go-errors/errors"
	flags "github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

var (
	// formats of cli-arg is "{reArgName}{reArgValue}"
	reArgName         = regexp.MustCompile(`from-([^=]+)`)
	reArgValue        = regexp.MustCompile(`([^=]+)(=([^=]+))?`)
	renderCommandOpts = &RenderCommand{}
)

var renderDescription = `Render combines the Secret Manager data and generates the output that can be applied to Kubernetes or Docker Compose.`
var renderDescriptionLong = `Render combines the Secret Manager data and generates the output that can be applied to Kubernetes or Docker Compose.

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

Configuration can also be done through YAML in .secrets-config.yml in key 'secretGenerator'.
`

func init() {
	_, err := parser.AddCommand("render", renderDescription, renderDescriptionLong, renderCommandOpts)
	panicIfErr(err)
	parser.UnknownOptionHandler = cliParseFromHandlers
}

// RenderPrefix is used by --from-sema-schema-to-file amongst others
var RenderPrefix string

// Verbose is used by --from-sema-schema-to-file amongst others
var Verbose bool

// Execute of RenderCommand is the 'sema render' command
func (opts *RenderCommand) Execute(args []string) error {
	prepareSemaClient(opts.Positional.Project)
	// Load defaults from config file
	configRenderCommand := opts.parseConfigFile()
	opts.mergeCommandOptions(configRenderCommand)
	// Default secret name to folder basename
	if opts.Name == "" {
		cwdpath, err := os.Getwd()
		panicIfErr(err)
		opts.Name = path.Base(cwdpath)
	}

	// Globally retrieved variables:
	RenderPrefix = opts.Prefix
	Verbose = len(opts.Verbose) > 0

	// Give all handlers a go to write to the secret data
	data := make(map[string][]byte, 0)
	for _, h := range opts.Handlers {
		h.Populate(data)
	}

	// Preamble, depending on the format
	if opts.Format == "yaml" {
		os.Stdout.WriteString(fmt.Sprintf(`kind: Secret
apiVersion: v1
metadata:
  name: %s
type: Opaque
data:
`, strconv.Quote(opts.Name)))
	}

	// Print all values in the correct format
	for _, key := range sortedKeys(data) {
		value := data[key]
		switch opts.Format {
		case "env":
			os.Stdout.WriteString(fmt.Sprintf("%s=%q\n", key, string(value)))
		case "files":
			writeDevSecretFile(opts.Dir, key, value)
		case "yaml":
			os.Stdout.WriteString(fmt.Sprintf("  %s: %s\n", key, base64.StdEncoding.EncodeToString([]byte(value))))
		default:
			panic(fmt.Errorf("Unknown format %q (use [env,files,yaml])", opts.Format))
		}
	}
	return nil
}

// Allows storing flags in a config file (uses yaml prefix 'secretGenerator')
func (opts *RenderCommand) parseConfigFile() RenderCommand {
	var configRenderCommand RenderCommand
	if opts.ConfigFile == "" {
		opts.ConfigFile = ".secrets-config.yml"
	}
	if _, err := os.Stat(opts.ConfigFile); err == nil {
		data, err := ioutil.ReadFile(opts.ConfigFile)
		if err != nil {
			panic(err)
		}
		configRenderCommand = parseConfigFileData(data)
	}
	return configRenderCommand
}

func (opts *RenderCommand) mergeCommandOptions(configFileOptions RenderCommand) {
	cmd := parser.Find("render")
	for _, option := range cmd.Options() {
		opt := struct {
			Name         string
			Value        interface{}
			IsSet        bool
			IsSetDefault bool
		}{option.LongName, option.Value(), option.IsSet(), option.IsSetDefault()}
		os.Stderr.WriteString(fmt.Sprintf(`%+v`, opt))
	}

	prefixOption := cmd.FindOptionByLongName("prefix")
	if !prefixOption.IsSet() && configFileOptions.Prefix != "" {
		opts.Prefix = configFileOptions.Prefix
	}
	nameOption := cmd.FindOptionByLongName("name")
	if !nameOption.IsSet() && configFileOptions.Name != "" {
		opts.Name = configFileOptions.Name
	}
	dirOption := cmd.FindOptionByLongName("dir")
	if dirOption.IsSetDefault() && configFileOptions.Dir != "" {
		opts.Dir = configFileOptions.Dir
	}
	opts.Handlers = append(opts.Handlers, configFileOptions.Handlers...)

}

func writeDevSecretFile(directory, key string, value []byte) {
	_, err := os.Stat(directory)
	filepath := filepath.Join(directory, key)
	if os.IsNotExist(err) {
		os.MkdirAll(directory, 0755)
	}
	_, err = os.Stat(filepath)
	if err == nil {
		fmt.Println(fmt.Sprintf("You are about to overwrite '%s', are you sure? [y/N]:", filepath))
		confirmed := askForConfirmation()
		if confirmed {
			err = ioutil.WriteFile(filepath, value, 0755)
			if err != nil {
				panic(fmt.Sprintf("error writing to file %s\n err: %s", filepath, err.Error()))
			}
		}
	} else {
		err = ioutil.WriteFile(filepath, value, 0755)
		if err != nil {
			panic(fmt.Sprintf("error writing to file %s\nerr: %s", filepath, err.Error()))
		}
	}
}

// RenderCommand describes how to use the render command
type RenderCommand struct {
	Positional struct {
		Project string `required:"yes" description:"Google Cloud project" positional-arg-name:"project"`
	} `positional-args:"yes"`
	Verbose    []bool          `short:"v" long:"verbose" description:"Show verbose debug information"`
	Format     string          `short:"f" long:"format" default:"yaml" description:"How to output: 'yaml' is a fully specified Kubernetes secret, 'env' will generate a *.env file format that can be used for Docker (Compose). 'files' will generate files per secret in the secrets folder"`
	Prefix     string          `long:"prefix" description:"A SecretManager prefix that will override non-prefixed keys"`
	Handlers   []SecretHandler `no-flag:"y"`
	Name       string          `long:"name" description:"Name of Kubernetes secret. NB: with Kustomize this will just be the prefix!"`
	Dir        string          `short:"d" long:"dir" default:"secrets" description:"Specify output directory when writing out to files, only used in combination with --format=files"`
	ConfigFile string          `short:"c" long:"config" description:"We read flags from this file, when present. Default location: .secrets-config.yml."`
}

// RenderConfigYAML is the same as RenderCommand but easily parsable
type RenderConfigYAML struct {
	Name    *string             `yaml:"name"`
	Prefix  *string             `yaml:"prefix"`
	Dir     *string             `yaml:"dir"`
	Secrets []map[string]string `yaml:"secrets"`
}

// For testing, repeatably executable
func parseRenderArgs(args []string) RenderCommand {
	opts := RenderCommand{}
	parser := flags.NewParser(&opts, flags.Default)
	parser.UnknownOptionHandler = cliParseFromHandlers

	// Do it
	renderCommandOpts.Handlers = []SecretHandler{}
	_, err := parser.ParseArgs(args)
	opts.Handlers = renderCommandOpts.Handlers
	if err != nil {
		os.Exit(1)
	}
	return opts
}

// Parse a yaml bytearray into a RenderCommand for easy testing
func parseConfigFileData(data []byte) RenderCommand {
	opts := RenderCommand{}
	flags.NewParser(&opts, flags.Default)
	var parsed RenderConfigYAML
	err := yaml.Unmarshal([]byte(data), &parsed)
	if err != nil {
		panic(err)
	}
	opts.Name = *parsed.Name
	opts.Prefix = *parsed.Prefix
	opts.Handlers = []SecretHandler{}
	for _, val := range parsed.Secrets {
		if _, ok := val["type"]; ok {
			handler, err := ParseSecretHandler(val)
			if err == nil {
				opts.Handlers = append(opts.Handlers, handler)
			} else {
				panic(err)
			}

		}
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
			renderCommandOpts.Handlers = append(renderCommandOpts.Handlers, handler)
			return args, nil
		}
	}
	return args, nil
}

func sortedKeys(mp map[string][]byte) (keys []string) {
	for v := range mp {
		keys = append(keys, v)
	}
	sort.Strings(keys)
	return
}
