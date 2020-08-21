package main

import (
	"encoding/base64"
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
	// private
	client secretmanager.KVClient
}

func (opts *addCommand) FormatCmd() string {
	cmd := fmt.Sprintf(`sema add %q %q %s`, opts.Positional.Project, opts.Positional.Name, formatLabels(opts.Labels))
	if len(opts.Force) > 0 && opts.Force[0] {
		cmd += " -f"
	}
	if len(opts.Force) > 0 && opts.Force[0] {
		cmd += " -v"
	}
	return fmt.Sprintf(`echo "%s" | base64 -D | %s`, base64.StdEncoding.EncodeToString([]byte(opts.Data)), cmd)
}

func (opts *addCommand) Execute(args []string) (err error) {
	if opts.client == nil {
		opts.client = prepareSemaClient(opts.Positional.Project)
	}

	if opts.Data == "" {
		opts.Data = readStringSilently("Enter secret value: ")
	}

	// Upsert "Secret" (the container)
	var secret secretmanager.KVValue
	secret, err = opts.client.Get(opts.Positional.Name)
	var isExistingSecret = secret != nil

	if secret == nil || status.Convert(err).Code() == codes.NotFound {
		secret, err = opts.client.New(opts.Positional.Name, opts.Labels)
		if err != nil {
			return err
		}
	} else if existingLabels := secret.GetLabels(); !equalLabels(existingLabels, opts.Labels) {
		if len(opts.Force) == 0 {
			return fmt.Errorf(`Please set the same labels, or use --force to update the already existing secret
  Existing labels: %s
  New labels:      %s`, formatLabels(existingLabels), formatLabels(opts.Labels))
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

func equalLabels(mpA map[string]string, mpB map[string]string) bool {
	// Unset labels will make regular reflect.DeepEqual insufficient, since:
	// reflect.DeepEqual(make(map[string]string, 0), nil) == false
	// So we allow empty maps to be equal with nil values
	if mpA == nil || mpB == nil || len(mpA) == 0 || len(mpB) == 0 {
		return len(mpA) == len(mpB)
	}
	return reflect.DeepEqual(mpA, mpB)
}
