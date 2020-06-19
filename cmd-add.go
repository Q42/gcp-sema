package main

func init() {
	parser.AddCommand("add", "Add a secret value to Secret Manager from the command-line", "", &addCommand{})
}

type addCommand struct {
}

func (*addCommand) Execute(args []string) error {
	log.Println("TODO")
	return nil
}
