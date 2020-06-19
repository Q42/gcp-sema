package main

import (
	// Secret Manager API from Google
	"context"
	"flag"
	loglib "log"
	"os"
	"regexp"

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

func parseArgs(args []string) arguments {
	set := flag.NewFlagSet("", flag.CommandLine.ErrorHandling())
	format := set.String("format", "yaml", "How to output: 'yaml' is a fully specified Kubernetes secret, 'env' will generate a *.env file format that can be used for Docker (Compose).")
	set.Parse(args)

	// Dummy
	args = []string{
		"--from-[handler]=[key]=[source]",
		"--from-literal=myfile.txt=literal-value",
		// plain files just like kubectl create secret
		"--from-file=myfile.txt=myfile.txt",
		// extract according to schema into a single property 'config-env.json'
		"--from-sema-schema-to-file=config-env.json=config-schema.json",
		// extract according to schema into environment variable literals
		"--from-sema-schema-to-literals=config-schema.json",
		// extract key value from SeMa into literals
		"--from-sema-literal=MY_APP_SECRET=MY_APP_SECRET_NEW",
	}

	var handlers = []argumentSecret{}
	re := regexp.MustCompile(`--from-([^=]+)=([^=]+)=([^=]+)`)
	for _, arg := range args {
		if matched := re.FindStringSubmatch(arg); len(matched) > 3 {
			handlers = append(handlers, argumentSecret{
				matched[1],
				matched[2],
				matched[3],
			})
		}
	}

	return arguments{
		format:   *format,
		handlers: handlers,
	}
}

type arguments struct {
	format   string
	handlers []argumentSecret
}

type argumentSecret struct {
	handler string
	key     string
	value   string
}
