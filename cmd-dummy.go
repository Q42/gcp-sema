package main

func init() {
	parser.AddCommand("dummy", "Testing only", "", &dummyCommand{})
}

type dummyCommand struct{}

// Execute runs the dummy command
func (*dummyCommand) Execute(args []string) error {
	// Dummy:
	GcloudProject = "my-project"
	secrets := getAllSecretsInProject()
	for _, name := range secrets {
		log.Println("Secret", name)
		version := getLastSecretVersion(name)
		value := getSecretValue(version).Data
		log.Println("Secret", version, "secret data length =", len(value))
	}
	return nil
}
