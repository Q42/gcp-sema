package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

func init() {
	parser.AddCommand("migration", "Migrate to Secret Manager", "", &migrateCommand{})
}

type migrateCommand struct {
	Positional struct {
		Project string `required:"yes" description:"Google Cloud project" positional-arg-name:"project"`
	} `positional-args:"yes"`
	Prefix string `long:"prefix" description:"A SecretManager prefix that will override non-prefixed keys"`
}

// Execute runs the migration command
func (opts *migrateCommand) Execute(args []string) error {
	GcloudProject = opts.Positional.Project
	path, err := os.Getwd()
	panicIfErr(err)

	// Collect all config-schema.json
	cmd := exec.CommandContext(context.Background(), "find", path, "-name", "config-schema.json", "-maxdepth", "2", "-print")
	data := bytes.NewBuffer([]byte{})
	cmd.Stdout = data
	cmd.Run()
	files := strings.Split(string(data.Bytes()), "\n")

	// Get all secret names that are available
	_ = getAllSecretsInProject()

	// Show all configuration options, suggested SecretManager keys
	// and which are already set.

	for _, file := range files {
		if strings.TrimSpace(file) != "" {
			for idx, conf := range parseSchemaFile(file).flatConfigurations {
				log.Printf("%d:\t%s: %s\n", idx, strings.Join(conf.Path, "."), conf.Format)
				if conf.Doc != "" {
					log.Println(" ", conf.Doc)
				}
				for _, suggestion := range convictToSemaKey(opts.Prefix, conf.Path) {
					log.Println("\t- ", suggestion)
				}
			}
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
