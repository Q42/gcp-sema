package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/flynn/json5"
	"github.com/go-errors/errors"
	"gopkg.in/yaml.v2"
)

func init() {
	parser.AddCommand("migration", "Migrate to Secret Manager", "", &migrateCommand{})
}

var allowedCharacters = func(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return r
	case r >= 'a' && r <= 'z':
		return r
	case r >= '0' && r <= '9':
		return r
	case r == '_':
		return r
	default:
		return '_'
	}
}

type migrateCommand struct {
	Positional struct {
		Project string `description:"Google Cloud project" positional-arg-name:"project"`
	} `positional-args:"yes"`
	Dir                  string `long:"dir" description:"Use this if the config-schema.json is in $dir relative to another directory; example --dir=server"`
	Prefix               string `long:"prefix" description:"A SecretManager prefix that will override non-prefixed keys"`
	KubernetesContext    string `long:"context" description:"Explicitly specify which kubectl context to run in to get the k8s secret."`
	KubernetesSecretName string `short:"s" long:"kubernetesSecretName" description:"Explicitly specify which k8s secret to migrate to SeMa."`
	KubernetesSecretCmd  string `short:"c" long:"kubernetesSecretCommand" description:"Explicitly specify which kubectl command to run to get the k8s secret."`
	Mode                 string `short:"m" long:"mode" default:"literal" choice:"literal" choice:"multi" description:"Defines how secret data is divided over Secret Manager secrets"`
	Force                []bool `short:"f" long:"force" description:"Overwrite existing secret & labels"`
}

// Execute runs the migration command
func (opts *migrateCommand) Execute(args []string) error {
	var heading = color.New(color.Bold, color.Underline)
	heading.Println("Migration")
	fmt.Println(`This command will:
- formulate the sema-plugin command that needs to be run as part of the deploy
- upload the existing kubernetes secret to Secret Manager
- has two modes,
  literal) which uploads the whole config-env.json as 1 secret and
  multi)   which splits the config-env.json into secrets like environment variables`)
	fmt.Println()

	GcloudProject = opts.getProject()
	workingDir, err := os.Getwd()
	panicIfErr(err)
	path := workingDir
	if opts.Dir != "" {
		path = filepath.Join(path, opts.Dir)
	}

	// Collect all config-schema.json
	deepSchemas := listFilesMatching(path, "config-schema.json", 2)
	schemas := listFilesMatching(path, "config-schema.json", 1)
	if len(deepSchemas) > len(schemas) {
		color.Blue("These are the available config-schema.json files in this tree:")
		for _, schema := range deepSchemas {
			color.Blue("- %s", schema)
		}
	}
	if len(schemas) < 1 {
		color.Red("No config-schema.json in this directory")
		os.Exit(1)
	}
	schemaPath, _ := filepath.Rel(workingDir, schemas[0])

	legacy, err := getLegacySecretConfiguration()
	panicIfErr(err)
	if opts.KubernetesSecretName == "" && legacy.Name != "" {
		opts.KubernetesSecretName = legacy.Name
	}

	// Use the secret name as the prefix in literal mode, unless it is explicitly set
	if opts.Prefix == "" && opts.Mode == "literal" {
		opts.Prefix = opts.KubernetesSecretName
	}

	if opts.KubernetesSecretCmd == "" {
		if opts.KubernetesContext == "" {
			opts.KubernetesContext, err = getCommandOutput("kubectl", "config", "current-context")
			panicIfErr(err)
		}
		opts.KubernetesSecretCmd = fmt.Sprintf(`kubectl get secret "%s" -o="json" --context="%s"`, opts.KubernetesSecretName, opts.KubernetesContext)
	}

	heading.Printf("Settings")
	fmt.Printf(`
- gcp project: %s
- k8s context: %s
- k8s secret:  %s
- k8s cmd:     %s
- schema:      %s
- sema prefix: %q
- mode:        %s

`, opts.Positional.Project, opts.KubernetesContext, opts.KubernetesSecretName, opts.KubernetesSecretCmd, schemaPath, opts.Prefix, opts.Mode)

	if !strings.HasPrefix(opts.KubernetesContext, "gke_"+opts.Positional.Project) {
		color.Red("Cowardly refusing to migrate secret from cluster %q to GCP project %q.\nAre you sure this is the desired kubectl cluster context and project?\nYou can override the command (-c) if you really need to.", opts.KubernetesContext, opts.Positional.Project)
		os.Exit(3)
	}

	// Get secret from Kubernetes
	k8sSecret := opts.getKubernetesSecret()
	log.Printf(`Found secret %q
  deployer: %q
  updated:  %q

`,
		k8sSecret.Metadata.Name,
		k8sSecret.Metadata.Labels["deployer"],
		k8sSecret.Metadata.Labels["updated"],
	)

	// Gather all actions so we can give an overview later
	actions := make([]ProposedAction, 0)

	switch opts.Mode {
	case "literal":
		// In literal mode we literally take each Kubernetes secret key/value pair and put it into Secret Manager.
		// For applications using config-env.json, this means that whole file is stored in 1 single value.

		manualCommand := fmt.Sprintf(`sema render %s --name="%s" `, opts.Positional.Project, opts.KubernetesSecretName)
		for _, secret := range legacy.Secrets {
			var data []byte
			if base64data, hasKey := k8sSecret.Data[secret.Name]; hasKey {
				data, err = base64.StdEncoding.DecodeString(base64data)
				if err != nil {
					return errors.WrapPrefix(err, fmt.Sprintf("failed to parse %q", secret.Name), 0)
				}
			} else {
				return fmt.Errorf("Configuration .secrets-config.yml contains %q but in Kubernetes this is unavailable", secret.Name)
			}
			// How it is called in Secret Manager. Note this can be an ugly name, but that is fine. This is migration only!
			semaName := strings.Map(allowedCharacters, fmt.Sprintf("%s_%s", opts.Prefix, secret.Name))
			actions = append(actions, &addCommand{
				Positional: addCommandPositional{Project: opts.Positional.Project, Name: semaName},
				Data:       string(data),
				Labels:     map[string]string{"source": k8sSecret.Metadata.Name, "prefix": opts.Prefix},
			})
			manualCommand += fmt.Sprintf("\\\n --from-sema-literal=\"%s=%s\"", secret.Name, semaName)
		}

		actions = append(actions, manualAction{
			Action: fmt.Sprintf("Manual: update config to run:\n%s", color.BlueString(manualCommand)),
		})

	case "multi":
		// Get all secret names that are available
		availableSecrets := getAllSecretsInProject()

		// List all configuration options, including existing values in config-env.json
		// and the suggested SecretManager keys and which of those are already set.
		for idx, conf := range parseSchemaFile(schemaPath).flatConfigurations {
			// print: 1: LOGLEVEL (format: [none,debug,info,warn,error], env: LOGLEVEL)
			infos := make([]string, 0)
			if conf.Format != nil {
				infos = append(infos, "format: "+conf.Format.String())
			}
			if conf.Env != "" {
				infos = append(infos, "env: "+conf.Env)
			}
			if conf.DefaultValue != nil {
				data, _ := json.Marshal(conf.DefaultValue)
				infos = append(infos, fmt.Sprintf("default: %s", string(data)))
			}
			log.Printf("%d:\t%s (%s)\n", idx, color.CyanString(strings.Join(conf.Path, ".")), strings.Join(infos, ", "))
			if conf.Doc != "" {
				log.Printf("\t%s\n", color.BlueString(conf.Doc))
			}

			// print all possible keys we'll look for later
			usedConfigEnvValue := false
			usedSemaKey := false

			configEnvName := fmt.Sprintf("secret %q at key %q", k8sSecret.Metadata.Name, strings.Join(conf.Path, "."))
			if node, exists := k8sSecret.Lookup(conf); exists == nil {
				ok, err := isSafeCoercible(node, conf)
				if ok {
					log.Println("\t- ", color.GreenString(configEnvName))
					usedConfigEnvValue = true
					semaName := convictToSemaKey(opts.Prefix, conf.Path)[0]
					data, _ := conf.Format.Flatten(node)
					actions = append(actions, &addCommand{
						Positional: addCommandPositional{Project: opts.Positional.Project, Name: semaName},
						Data:       data,
						Labels:     map[string]string{"source": k8sSecret.Metadata.Name, "prefix": opts.Prefix},
					})
				} else {
					log.Println("\t- ", color.RedString(configEnvName), errors.WrapPrefix(err, "value not safe to convert to string", 0))
				}
			} else {
				log.Println("\t- ", color.RedString(configEnvName))
			}

			for _, suggestion := range convictToSemaKey(opts.Prefix, conf.Path) {
				if isListElement(availableSecrets, suggestion) {
					if !usedConfigEnvValue && !usedSemaKey {
						usedSemaKey = true
						actions = append(actions, &manualAction{
							Action: fmt.Sprintf("validate Secret Manager key %q", suggestion),
						})
					}
					log.Println("\t- ", color.GreenString(suggestion))
				} else {
					log.Println("\t- ", color.RedString(suggestion))
				}
			}
		}

		manualCommand := fmt.Sprintf(`sema render %s --name="%s" --prefix="%s"`, opts.Positional.Project, opts.KubernetesSecretName, opts.Prefix)
		manualCommand += fmt.Sprintf("\\\n  --from-sema-schema-to-file=config-env.json=%s", schemaPath)
		actions = append(actions, manualAction{
			Action: fmt.Sprintf("Manual: update config to run:\n%s", color.BlueString(manualCommand)),
		})
	default:
		log.Fatalf("Invalid mode %s", opts.Mode)
	}

	heading.Println("Plan:")
	for _, action := range actions {
		log.Println("-", action.Explainer())
	}
	log.Println()

	if prompt("Continue? [y/N] ") != "y" {
		color.Red("Aborted")
		os.Exit(127)
	}

	for _, action := range actions {
		err := action.Func()()
		if err == nil {
			log.Println("- ✅", action.Explainer())
		} else {
			log.Println("- 🚨", err)
		}
	}

	// // Dummy:
	// GcloudProject = "my-project"
	// secrets := getAllSecretsInProject()
	// for _, name := range secrets {
	// 	log.Println("Secret", name)
	// 	version := getLastSecretVersion(name)
	// 	value := getSecretValue(version).Data
	// 	log.Println("Secret", version, "secret data length =", len(value))
	// }
	return nil
}

type kubernetesSecret struct {
	Metadata struct {
		Name        string
		Annotations map[string]string
		Labels      map[string]string
	}
	Data           map[string]string
	configEnvCache map[string]interface{}
}

func (opts *migrateCommand) getKubernetesSecret() *kubernetesSecret {
	k8sSecretData, err := getCommandOutput("sh", "-c", opts.KubernetesSecretCmd)
	panicIfErr(err)

	var k8sSecret kubernetesSecret
	err = json.Unmarshal([]byte(k8sSecretData), &k8sSecret)
	panicIfErr(err)
	return &k8sSecret
}

func (s *kubernetesSecret) Lookup(conf convictConfiguration) (interface{}, error) {
	if s.configEnvCache == nil {
		s.configEnvCache = make(map[string]interface{})
		// naive: this can be renamed inside apps!
		if data, isSet := s.Data["config-env.json"]; isSet {
			var bytes []byte
			bytes, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, err
			}
			err = json5.Unmarshal(bytes, &s.configEnvCache)
			if err != nil {
				return nil, err
			}
		}
	}
	var node interface{} = s.configEnvCache
	for _, key := range conf.Path {
		switch v := node.(type) {
		case map[string]interface{}:
			node = v[key]
		default:
			return nil, fmt.Errorf("config-env.json contains no property at %q", conf.Path)
		}
	}
	return node, nil
}

func (opts *migrateCommand) getProject() string {
	if opts.Positional.Project != "" {
		return opts.Positional.Project
	}
	// if cli-argument [project] is not set:
	project := prompt("Which project? [my-project]: ")
	// if user typed Enter directly, use default:
	if project == "" {
		project = "my-project"
	}
	opts.Positional.Project = project
	return opts.Positional.Project
}

func prompt(name string) string {
	fmt.Printf(name)
	reader := bufio.NewReader(os.Stdin)
	value, _ := reader.ReadString('\n')
	value = strings.Trim(value, "\n\r")
	return value
}

func convictToSemaKey(prefix string, path []string) []string {
	if prefix != "" {
		return []string{
			strings.Join(append([]string{prefix}, path...), "_"),
			strings.Join(path, "_"),
		}
	}
	return []string{
		strings.Join(path, "_"),
	}
}

func isListElement(availables []string, suggestion string) bool {
	for _, available := range availables {
		if suggestion == available {
			return true
		}
	}
	return false
}

func colorBasedOnAvailability(availables []string, suggestion string) string {
	for _, available := range availables {
		if suggestion == available {
			return color.GreenString(suggestion)
		}
	}
	return color.RedString(suggestion)
}

func listFilesMatching(path, namePattern string, maxDepth int) (files []string) {
	cmd := exec.CommandContext(context.Background(), "find", path, "-name", namePattern, "-maxdepth", strconv.Itoa(maxDepth), "-print")
	data := bytes.NewBuffer([]byte{})
	cmd.Stdout = data
	cmd.Run()
	for _, file := range strings.Split(string(data.Bytes()), "\n") {
		if file != "" {
			files = append(files, file)
		}
	}
	return
}

func getCommandOutput(command string, args ...string) (output string, err error) {
	cmd := exec.CommandContext(context.Background(), command, args...)
	data := bytes.NewBuffer([]byte{})
	cmd.Stdout = data
	err = cmd.Run()
	output = strings.TrimSpace(string(data.Bytes()))
	return
}

type legacySecretConfig struct {
	Name    string `yaml:"name"`
	Secrets []struct {
		Path string `yaml:"path,omitempty"`
		Name string `yaml:"name,omitempty"`
	}
}

func getLegacySecretConfiguration() (config *legacySecretConfig, err error) {
	fileData, err := ioutil.ReadFile(".secrets-config.yml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(fileData, &config)
	return
}

// ProposedAction allows us to ask for confirmation and THEN execute
type ProposedAction interface {
	Explainer() string
	Func() func() error
}

type manualAction struct {
	Action string
}

func (a manualAction) Explainer() string {
	return a.Action
}
func (a manualAction) Func() func() error {
	return func() error { return nil }
}

func (a *addCommand) Explainer() string {
	length := len(a.Data)
	h := sha1.New()
	h.Write([]byte(a.Data))
	shasum := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("Upload secret %q to SecretManager (length=%d, sha1sum=%s)", a.Positional.Name, length, shasum)
}

func (a *addCommand) Func() func() error {
	return func() error {
		return a.Execute([]string{})
	}
}

// Detects if a configuration value will survive being serialized to string
func isSafeCoercible(node interface{}, conf convictConfiguration) (bool, error) {
	value, err := conf.Format.Flatten(node)
	if err != nil {
		return false, err
	}
	nodeConverted, err := conf.Format.Coerce(value)
	if err != nil {
		return false, err
	}
	nextValue, err := conf.Format.Flatten(nodeConverted)
	if err != nil {
		return false, err
	}
	return nextValue == value, nil
}
