package main

import (
	"regexp"

	flags "github.com/jessevdk/go-flags"
)

// Usage describes how to use this module
type Usage struct {
	Verbose  []bool          `short:"v" long:"verbose" description:"Show verbose debug information"`
	Format   string          `short:"o" long:"format" description:"How to output: 'yaml' is a fully specified Kubernetes secret, 'env' will generate a *.env file format that can be used for Docker (Compose)."`
	Handlers []SecretHandler `no-flag:"y" description:"multiple ways to specify a secret source, the format is --from-[handler]=[key]=[source/value]"`
}

func parseArgs(args []string) Usage {
	opts := Usage{}
	reArgName := regexp.MustCompile(`from-([^=]+)`)
	reArgValue := regexp.MustCompile(`([^=]+)=([^=]+)`)

	parser := flags.NewParser(&opts, flags.Default) // flags.IgnoreUnknown
	parser.UnknownOptionHandler = func(option string, arg flags.SplitArgument, args []string) ([]string, error) {
		log.Println("UnknownOptionHandler", option, arg, args)
		value, hasValue := arg.Value()
		if matchedKey := reArgName.FindStringSubmatch(option); len(matchedKey) == 2 && hasValue {
			if matchedValue := reArgValue.FindStringSubmatch(value); len(matchedValue) == 3 {
				handler := MakeSecretHandler(matchedKey[1], matchedValue[1], matchedValue[2])
				opts.Handlers = append(opts.Handlers, handler)
				return args, nil
			}
		}
		return args, nil
	}

	_, err := parser.ParseArgs(args)
	panicIfErr(err)
	return opts
}
