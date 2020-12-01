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

	"github.com/Q42/gcp-sema/pkg/handlers"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/go-errors/errors"
	flags "github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

var (
	// formats of cli-arg is "{reArgName}{reArgValue}"
	reArgName         = regexp.MustCompile(`from-([^=]+)`)
	reArgValue        = regexp.MustCompile(`([^=]+)(=([^=]+))?`)
	renderCommandOpts = &RenderCommand{}
	renderCommand     *flags.Command
)

var renderDescription = `Render combines the Secret Manager data and generates the output that can be applied to Kubernetes or Docker Compose.`
var renderDescriptionLong = fmt.Sprintf(`Render combines the Secret Manager data and generates the output that can be applied to Kubernetes or Docker Compose.

There are multiple ways to specify a secret source, the format is -s [handler],[key],[source/value].
The following options are implemented:

  # literals just like kubectl create secret --from-literal=myfile.txt=foo-bar
  -s literal=myfile.txt=foo-bar

  # plain files just like kubectl create secret --from-file=myfile.txt=./myfile.txt
  -s file=myfile.txt=./myfile.txt

  # extract according to schema into a single property 'config-env.json'
  -s sema-schema-to-file=config-env.json=config-schema.json

  # extract according to schema into environment variable literals
  -s sema-schema-to-literals=config-schema.json

  # extract key value from SeMa into literals
  -s sema-literal=MY_APP_SECRET=MY_APP_SECRET_NEW

Configuration can also be done through YAML in file %q.
`, DefaultFileSecretsConfig)

func init() {
	var err error
	renderCommand, err = parser.AddCommand("render", renderDescription, renderDescriptionLong, renderCommandOpts)
	panicIfErr(err)
}

// Execute of RenderCommand is the 'sema render' command
func (opts *RenderCommand) Execute(args []string) error {
	if len(args) > 0 && args[0] == "test" {
		return nil
	}

	// Load defaults from config file
	configRenderCommand := opts.parseConfigFile()
	opts.mergeCommandOptions(renderCommand, configRenderCommand)
	// Default secret name to folder basename
	if opts.Name == "" {
		cwdpath, err := os.Getwd()
		panicIfErr(err)
		opts.Name = path.Base(cwdpath)
	}

	// Inject SeMa client into handlers:
	var client secretmanager.KVClient
	var err error
	if opts.MockSema {
		client = secretmanager.NewMockClient("mock", "*", "")
	} else {
		if opts.OfflineLookupFile != "" {
			client, err = secretmanager.NewOfflineClient(opts.OfflineLookupFile, opts.Positional.Project)
			panicIfErr(err)
		} else {
			client = prepareSemaClient(opts.Positional.Project)
		}
	}
	opts.Handlers = handlers.InjectSemaClient(opts.Handlers, client, handlers.SecretHandlerOptions{
		Prefix:  opts.Prefix,
		Mock:    opts.MockSema,
		Verbose: len(opts.Verbose) > 0,
	})

	// Give all handlers a go at downloading key-value lists/preparations
	// Give all handlers a go to write annotation data
	fields := make(map[string]bool, 0)
	annotations := make(map[string]string, 0)
	for _, h := range opts.Handlers {
		h.Prepare(fields)
		h.Annotate(func(key, value string) {
			key, value, ok := postProcessAnnotation(key, value)
			if ok {
				annotations[key] = value
			}
		})
	}

	if opts.Format == "files" {
		for _, file := range sortedKeysB(fields) {
			log.Printf("Preparing to write %q", file)
		}
		log.Println("Downloading from key-value sources (use --verbose to preview which)...")
		if len(opts.Verbose) > 0 {
			for _, v := range annotations {
				log.Printf("Source %q", v)
			}
		}
	}

	// Preamble, depending on the format
	if opts.Format == "yaml" {
		yml, err := yaml.Marshal(secretYAML{
			Kind:       "Secret",
			APIVersion: "v1",
			Type:       "Opaque",
			Metadata: secretYAMLMetadata{
				Name:        opts.Name,
				Annotations: annotations,
				Labels: map[string]string{
					"info/generated-by": "sema",
				},
			},
		})
		panicIfErr(err)
		os.Stdout.WriteString(string(yml))
		// Write 'data' separately to unify writing in the different formats
		os.Stdout.WriteString("data:\n")
	}

	// Give all handlers a go to write to the secret data
	data := make(map[string][]byte, 0)
	for _, h := range opts.Handlers {
		h.Populate(data)
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

// Allows storing flags in a config file
func (opts *RenderCommand) parseConfigFile() RenderCommand {
	var configRenderCommand RenderCommand
	if opts.ConfigFile == "" {
		opts.ConfigFile = DefaultFileSecretsConfig
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

func (opts *RenderCommand) mergeCommandOptions(command *flags.Command, configFileOptions RenderCommand) {
	prefixOption := command.FindOptionByLongName("prefix")

	if prefixOption.IsSetDefault() && configFileOptions.Prefix != "" {
		opts.Prefix = configFileOptions.Prefix
	}
	nameOption := command.FindOptionByLongName("name")
	if nameOption.IsSetDefault() && configFileOptions.Name != "" {
		opts.Name = configFileOptions.Name
	}
	dirOption := command.FindOptionByLongName("dir")
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
		fmt.Printf(fmt.Sprintf("You are about to overwrite '%s', are you sure? [y/N]: ", filepath))
		confirmed := askForConfirmation()
		if confirmed {
			err = ioutil.WriteFile(filepath, value, 0755)
			if err != nil {
				panic(fmt.Sprintf("Error writing to file %s\n err: %s", filepath, err.Error()))
			}
		}
	} else {
		err = ioutil.WriteFile(filepath, value, 0755)
		if err != nil {
			panic(fmt.Sprintf("Error writing to file %s\nerr: %s", filepath, err.Error()))
		}
		fmt.Println(fmt.Sprintf("Rendered %s", filepath))
	}
}

// RenderCommand describes how to use the render command
type RenderCommand struct {
	Positional struct {
		Project string `required:"yes" description:"Google Cloud project" positional-arg-name:"project"`
	} `positional-args:"yes"`
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	Format  string `short:"f" long:"format" default:"yaml" description:"How to output: 'yaml' is a fully specified Kubernetes secret, 'env' will generate a *.env file format that can be used for Docker (Compose). 'files' will generate files per secret in the secrets folder"`
	Prefix  string `long:"prefix" description:"A SecretManager prefix that will override non-prefixed keys"`

	Handlers []handlers.ConcreteSecretHandler `short:"s" long:"secrets" description:"The Secret source, this can be specified multiple times"`

	Name       string `long:"name" description:"Name of Kubernetes secret. NB: with Kustomize this will just be the prefix!"`
	Dir        string `short:"d" long:"dir" default:"secrets" description:"Specify output directory when writing out to files, only used in combination with --format=files"`
	ConfigFile string `short:"c" long:"config" description:"We read flags from this file, when present. Default location: .secrets-config.yml."`
	// Debugging/offline usage
	OfflineLookupFile string ` env:"OFFLINE" long:"offline" description:"You might want to run sema as an unprivileged user, for testing/validation purposes for example. Use this to provide fake/real/offline secrets."`
	MockSema          bool   ` env:"MOCK_SEMA" long:"mock-sema" description:"If you want to run without having Secret-Manager access"`
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

	// Do it
	_, err := parser.ParseArgs(args)
	if err != nil {
		os.Exit(1)
	}
	return opts
}

// Parse a yaml bytearray into a RenderCommand for easy testing
func parseConfigFileData(data []byte) RenderCommand {
	opts := RenderCommand{}
	var parsed RenderConfigYAML
	err := yaml.Unmarshal([]byte(data), &parsed)
	if err != nil {
		panic(err)
	}
	opts.Name = valueOrEmpty(parsed.Name)
	opts.Prefix = valueOrEmpty(parsed.Prefix)
	opts.Dir = valueOrEmpty(parsed.Dir)
	opts.Handlers = []handlers.ConcreteSecretHandler{}
	for _, val := range parsed.Secrets {
		if _, ok := val["type"]; ok {
			handler, err := handlers.ParseSecretHandler(val)
			if err == nil {
				opts.Handlers = append(opts.Handlers, handlers.ConcreteSecretHandler{SecretHandler: handler})
			} else {
				panic(err)
			}

		}
	}
	return opts
}

// UnknownOptionHandler parses all --from-... arguments into opts.Handlers
func cliParseFromHandlers(commandOptions *RenderCommand, option string, arg flags.SplitArgument, args []string) (nextArgs []string, outErr error) {
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
			handler, err := handlers.MakeSecretHandler(matchedKey[1], matchedValue[1], matchedValue[3])
			commandOptions.Handlers = append(commandOptions.Handlers, handlers.ConcreteSecretHandler{SecretHandler: handler})
			return args, err
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

func sortedKeysB(mp map[string]bool) (keys []string) {
	for v := range mp {
		keys = append(keys, v)
	}
	sort.Strings(keys)
	return
}

func valueOrEmpty(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

type secretYAML struct {
	Kind       string             `yaml:"kind"`
	APIVersion string             `yaml:"apiVersion"`
	Metadata   secretYAMLMetadata `yaml:"metadata"`
	Type       string             `yaml:"type"`
}

type secretYAMLMetadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations"`
	Labels      map[string]string `yaml:"labels"`
}
