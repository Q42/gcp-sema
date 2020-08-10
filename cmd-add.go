package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"syscall"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/go-errors/errors"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	parser.AddCommand("add", "Add a secret value to Secret Manager from the command-line", "", &addCommand{})
}

type addCommandPositional struct {
	Project string `required:"yes" description:"Google Cloud project" positional-arg-name:"project"`
	Name    string `long:"name" default:"mysecretkey" description:"Name of secret key"`
}

type addCommand struct {
	Positional addCommandPositional `positional-args:"yes"`
	Labels     map[string]string    `short:"l" long:"label" description:"set labels using --label=foo:bar"`
	Force      []bool               `short:"f" long:"force" description:"force overwrite value/labels"`
	Verbose    []bool               `short:"v" long:"verbose" description:"Show verbose debug information"`
	Data       string               `hidden:"yes"`
}

func (opts *addCommand) Execute(args []string) (err error) {
	prepareSemaClient(opts.Positional.Project)

	if opts.Data == "" {
		opts.Data = readStringSilently("Enter secret value: ")
	}

	// Upsert "Secret" (the container)
	var secret secretmanager.KVValue
	secret, err = client.Get(opts.Positional.Name)
	var isExistingSecret = secret != nil

	if secret == nil || status.Convert(err).Code() == codes.NotFound {
		_, err := client.New(opts.Positional.Name, opts.Labels)
		if err != nil {
			return err
		}
	} else if !reflect.DeepEqual(secret.GetLabels(), opts.Labels) {
		if len(opts.Force) == 0 {
			log.Println("Existing labels:", formatLabels(secret.GetLabels()))
			return errors.New("Please set the same labels, or use --force to update the already existing secret")
		}
		err = secret.SetLabels(opts.Labels)
		if err != nil {
			return err
		}
	}

	// Upsert "Secret Version" (the actual data)
	if isExistingSecret && len(opts.Force) == 0 {
		return errors.New("Please use --force to update value of existing secret")
	}

	version, err := secret.SetValue([]byte(opts.Data))
	if err != nil {
		return err
	}

	log.Println("Written", version)
	return nil
}

// readStringSilently will ensure that if you type the password on the commandline,
// the value is not copied to the output framebuffer, by using `terminal.ReadPassword`.
func readStringSilently(prompt string) (secret string) {
	if terminal.IsTerminal(syscall.Stdin) {
		log.Printf(prompt)
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal(err)
		}
		secret = string(bytePassword)
	} else {
		password, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		secret = string(password)
	}
	return
}

func formatLabels(mp map[string]string) string {
	labels := make([]string, 0)
	for k, v := range mp {
		labels = append(labels, fmt.Sprintf("-l %s:%s", k, v))
	}
	return strings.Join(labels, " ")
}
