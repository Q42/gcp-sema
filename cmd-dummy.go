package main

func init() {
	parser.AddCommand("dummy", "Testing only", "", &dummyCommand{})
}

type dummyCommand struct{}

// Execute runs the dummy command
func (*dummyCommand) Execute(args []string) error {
	client := prepareSemaClient("my-project")

	// Dummy:
	secrets, err := client.ListKeys()
	panicIfErr(err)
	for _, secret := range secrets {
		log.Println("Secret", secret)
		value, err := secret.GetValue()
		panicIfErr(err)
		log.Println("secret data length =", len(value))
	}
	return nil
}
