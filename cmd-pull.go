package main

import (
	"fmt"

	flags "github.com/jessevdk/go-flags"
)

var pullCommandOpts = &PullCommand{}
var pullCommand *flags.Command

type PullCommand struct {
	RenderCommand
}

func init() {
	var err error
	pullCommand, err = parser.AddCommand("pull", renderDescription, renderDescriptionLong, pullCommandOpts)
	panicIfErr(err)
	parser.UnknownOptionHandler = cliParseFromHandlers
}

// PullCommand is an alias for RenderCommand with the difference that it writes to files on disk
// instead of rendering YAMLs by default.
func (opts *PullCommand) Execute(args []string) error {
	fmt.Println(opts, args)
	formatOpt := pullCommand.FindOptionByLongName("format")
	if formatOpt.IsSetDefault() {
		opts.Format = "files"
	}
	return opts.RenderCommand.Execute(args)
}
