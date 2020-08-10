package main

import (
	flags "github.com/jessevdk/go-flags"
)

var pullCommandOpts = &pullCommand{}
var pullCommandInst *flags.Command
var pullDescription = `Pull combines the Secret Manager data, and saves them on disk like how they would be available in Kubernetes, use this for local development.`

// pullCommand is an alias for RenderCommand with the difference that it writes to files on disk
// instead of rendering YAMLs by default.
type pullCommand struct{ RenderCommand }

func init() {
	var err error
	pullCommandInst, err = parser.AddCommand("pull", pullDescription, pullDescription+" See 'render' help for more information.", pullCommandOpts)
	panicIfErr(err)
	parser.UnknownOptionHandler = cliParseFromHandlers
}

func (opts *pullCommand) Execute(args []string) error {
	// if the render default ("yaml") is used, use "files" instead
	formatOpt := pullCommandInst.FindOptionByLongName("format")
	if formatOpt.IsSetDefault() {
		opts.Format = "files"
	}
	return opts.RenderCommand.Execute(args)
}
