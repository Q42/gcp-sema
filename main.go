package main

import (
	// Secret Manager API from Google
	"context"
	loglib "log"
	"os"

	sema "cloud.google.com/go/secretmanager/apiv1"
)

var client *sema.Client
var ctx context.Context = context.Background()
var log *loglib.Logger = loglib.New(os.Stderr, "", 0)

func main() {
	// Get Secret Manager client
	var err error
	client, err = sema.NewClient(ctx)
	panicIfErr(err)

	if os.Args[1] == "add" {
		log.Println("TODO")
	} else if os.Args[1] == "create" {
		create(os.Args[2:])
	} else if os.Args[1] == "dummy" {
		// Dummy:
		secrets := getAllSecretsInProject("my-project")
		for _, name := range secrets {
			log.Println("Secret", name)
			version := getLastSecretVersion(name)
			value := getSecretValue(version).Data
			log.Println("Secret", version, "secret data length =", len(value))
		}
	}
}
