package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func init() {
	parser.AddCommand("add", "Add a secret value to Secret Manager from the command-line", "", &addCommand{})
}

type addCommand struct {
	Positional struct {
		Project string `required:"yes" description:"Google Cloud project" positional-arg-name:"project"`
		Name    string `long:"name" default:"mysecretkey" description:"Name of secret key"`
	} `positional-args:"yes"`
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	Prefix  string `long:"prefix" description:"A SecretManager prefix that will override non-prefixed keys"`
}

func (*addCommand) Execute(args []string) error {
	var secret string
	if terminal.IsTerminal(syscall.Stdin) {
		fmt.Print("Enter secret value: ")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal(err)
		}
		secret = string(bytePassword)
	} else {
		reader := bufio.NewReader(os.Stdin)
		password, _ := reader.ReadString('\n')
		secret = strings.Trim(password, "\n\r")
	}
	log.Println(secret)

	return nil
}
