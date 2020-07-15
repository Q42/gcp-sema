package main

import (
	// Secret Manager API from Google
	"context"
	loglib "log"
	"os"

	sema "cloud.google.com/go/secretmanager/apiv1"
	flags "github.com/jessevdk/go-flags"
)

var client *sema.Client
var ctx context.Context = context.Background()
var log *loglib.Logger = loglib.New(os.Stderr, "", 0)
var parser = flags.NewParser(&struct{}{}, flags.Default)

func prepareSemaClient() {
	// Get Secret Manager client
	var err error
	client, err = sema.NewClient(ctx)
	panicIfErr(err)
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
