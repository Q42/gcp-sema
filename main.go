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

	// log.Println("Called with $0", os.Args[1:])
	// log.Println("Usage:", "$0 [project] --format=yaml --from-[handler]=[key]=[source]")
	// log.Println("Usage:", "$0 [project] --format=env --from-[handler]=[key]=[source]")
	// log.Printf("Parsed arguments to %+v", )
	// log.Println("")
	if os.Args[1] == "add" {
		log.Println("TODO")
	} else if os.Args[1] == "create" {
		opts := parseArgs(os.Args[2:])

		// Preamble, depending on the format
		if opts.Format == "" || opts.Format == "yaml" {
			log.Println(`apiVersion: v1
kind: Secret
metadata:
  name: mysecret
type: Opaque
data:`)
		}

		// Give all handlers a go to write to the secret data
		data := make(map[string][]byte, 0)
		for _, h := range opts.Handlers {
			h.Populate(data)
		}

		// Print all values in the correct format
		for key, value := range data {
			switch opts.Format {
			case "env":
				log.Printf("%s=%s", key, string(value))
			default:
				log.Printf("  %s: %s", key, string(value))
			}
		}
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
