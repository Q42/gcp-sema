package main

import (
	"os"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
)

func init() {
	parser.AddCommand("get", "Get a secret value from Secret Manager from the command-line", "", &getCommand{})
}

type getCommandPositional struct {
	Project string `required:"yes" description:"Google Cloud project" positional-arg-name:"project"`
	Name    string `required:"yes" long:"name" description:"Name of secret key" positional-arg-name:"name"`
}

type getCommand struct {
	Positional getCommandPositional `positional-args:"yes"`
	// private
	client secretmanager.KVClient
}

func (opts *getCommand) Execute(args []string) (err error) {
	if opts.client == nil {
		opts.client = prepareSemaClient(opts.Positional.Project)
	}

	secret, err := opts.client.Get(opts.Positional.Name)
	if err != nil {
		return err
	}

	value, err := secret.GetValue()
	if err != nil {
		return err
	}

	os.Stdout.Write(value)
	os.Stderr.WriteString("\n")
	return nil
}
