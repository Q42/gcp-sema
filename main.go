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

	log.Println("Called with $0", os.Args[1:])
	log.Println("Usage:", "$0 [project] --format=yaml --from-[handler]=[key]=[source]")
	log.Println("Usage:", "$0 [project] --format=env --from-[handler]=[key]=[source]")
	log.Printf("Parsed arguments to %+v", parseArgs(os.Args[1:]))
	log.Println("")

	// Dummy:
	secrets := getAllSecretsInProject("my-project")
	for _, name := range secrets {
		log.Println("Secret", name)
		version := getLastSecretVersion(name)
		value := getSecretValue(version).Data
		log.Println("Secret", version, "secret data length =", len(value))
	}
}
