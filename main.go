package main

import (
	// Secret Manager API from Google

	loglib "log"
	"os"

	secretmanager "github.com/Q42/gcp-sema/pkg/secretmanager"
	flags "github.com/jessevdk/go-flags"
)

var client secretmanager.KVClient
var log *loglib.Logger = loglib.New(os.Stderr, "", 0)
var parser = flags.NewParser(&struct{}{}, flags.Default)

func prepareSemaClient(project string) secretmanager.KVClient {
	// Get Secret Manager client
	var err error
	client, err = secretmanager.NewClient(project)
	panicIfErr(err)
	return client
}

func main() {
	// Subcommands are added in cmd-*.go files
	_, err := parser.Parse()
	if err != nil {
		flagsErr, ok := err.(*flags.Error)
		if ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}
