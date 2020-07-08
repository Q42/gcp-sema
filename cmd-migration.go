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
	"strconv"
	"strings"

	"github.com/fatih/color"
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
	path, err := os.Getwd()
	panicIfErr(err)

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

	legacy, err := getLegacySecretConfiguration()
	panicIfErr(err)
	if opts.KubernetesSecretName == "" && legacy.Name != "" {
		opts.KubernetesSecretName = legacy.Name
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

`, opts.Positional.Project, opts.KubernetesContext, opts.KubernetesSecretName, opts.KubernetesSecretCmd, schemas[0], opts.Prefix, opts.Mode)

	if !strings.HasPrefix(opts.KubernetesContext, "gke_"+opts.Positional.Project) {
		color.Red("Cowardly refusing to migrate secret from cluster %q to GCP project %q.\nAre you sure this is the desired kubectl cluster context and project?\nYou can override the command (-c) if you really need to.", opts.KubernetesContext, opts.Positional.Project)
		os.Exit(3)
	}

	// Get all secret names that are available
	availableSecrets := getAllSecretsInProject()

	// Get secret from Kubernetes
	k8sSecretData, err := getCommandOutput("sh", "-c", opts.KubernetesSecretCmd)
	panicIfErr(err)
	var k8sSecret struct {
		Metadata struct {
			Name        string
			Annotations map[string]string
			Labels      map[string]string
		}
		Data map[string]string
	}
	err = json.Unmarshal([]byte(k8sSecretData), &k8sSecret)
	panicIfErr(err)
	log.Printf(`Found secret %q
  deployer: %q
  updated:  %q

`,
		k8sSecret.Metadata.Name,
		k8sSecret.Metadata.Labels["deployer"],
		k8sSecret.Metadata.Labels["updated"],
	)

	actions := make([]ProposedAction, 0)

	switch opts.Mode {
	case "literal":
		cmd := fmt.Sprintf(`sema create %s --name="%s"`, opts.Positional.Project, opts.KubernetesSecretName)
		for _, secret := range legacy.Secrets {
			if k8sSecret.Data[secret.Name] == "" {
				fmt.Printf("Missing secret with key %q\n", secret.Name)
				continue
			}
			data, err := base64.StdEncoding.DecodeString(k8sSecret.Data[secret.Name])
			panicIfErr(err)
			secretUploadName := strings.Map(allowedCharacters, fmt.Sprintf("%s_%s", opts.KubernetesSecretName, secret.Name))
			actions = append(actions, &addCommand{
				Positional: addCommandPositional{Project: opts.Positional.Project, Name: secretUploadName},
				Data:       string(data),
				Labels:     map[string]string{"source": k8sSecret.Metadata.Name, "prefix": opts.Prefix},
			})
			cmd = cmd + fmt.Sprintf(` --from-sema-literal="%s=%s"`, secret.Name, secretUploadName)
		}

		actions = append(actions, manualAction{
			Action: fmt.Sprintf(`Manual: update config to run: %s`, cmd),
		})

	case "multi":
		// Show all configuration options, suggested SecretManager keys
		// and which are already set.

		for idx, conf := range parseSchemaFile(schemas[0]).flatConfigurations {
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
			for _, suggestion := range convictToSemaKey(opts.Prefix, conf.Path) {
				log.Println("\t- ", colorBasedOnAvailability(availableSecrets, suggestion))
			}
		}
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
			strings.Join(path, "_"),
			strings.Join(append([]string{prefix}, path...), "_"),
		}
	}
	return []string{
		strings.Join(path, "_"),
	}
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
