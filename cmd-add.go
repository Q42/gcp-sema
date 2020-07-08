package main

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"
	"syscall"

	"github.com/go-errors/errors"
	"golang.org/x/crypto/ssh/terminal"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/genproto/protobuf/field_mask"
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
	Force      []bool               `short:"f" long:"force" description:"force overwrite labels"`
	Verbose    []bool               `short:"v" long:"verbose" description:"Show verbose debug information"`
	Data       string               `hidden:"yes"`
}

func (opts *addCommand) Execute(args []string) (err error) {
	if opts.Data == "" {
		opts.Data = readStringSilently("Enter secret value: ")
	}

	// Upsert "Secret" (the container)
	var secret *secretmanagerpb.Secret
	var version string
	secret, err = client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", opts.Positional.Project, opts.Positional.Name),
	})
	var isExistingSecret = secret != nil

	if secret == nil || status.Convert(err).Code() == codes.NotFound {
		_, err := client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
			Parent:   fmt.Sprintf("projects/%s", opts.Positional.Project),
			SecretId: opts.Positional.Name,
			Secret: &secretmanagerpb.Secret{
				Labels: opts.Labels, Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{},
				}},
		})
		if err != nil {
			return err
		}
	} else if !reflect.DeepEqual(secret.Labels, opts.Labels) {
		if len(opts.Force) > 0 {
			secret.Labels = opts.Labels
			secret, err = client.UpdateSecret(ctx, &secretmanagerpb.UpdateSecretRequest{
				Secret:     secret,
				UpdateMask: &field_mask.FieldMask{Paths: []string{"labels"}},
			})
			if err != nil {
				return err
			}
		} else {
			return errors.New("Please use --force to update labels of already existing secret")
		}
	}

	// Upsert "Secret Version" (the actual data)
	if isExistingSecret && len(opts.Force) == 0 {
		return errors.New("Please use --force to update value of existing secret")
	}

	version, err = writeSecretVersion(opts.Positional.Project, opts.Positional.Name, opts.Data)
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
		reader := bufio.NewReader(os.Stdin)
		password, _ := reader.ReadString('\n')
		secret = strings.Trim(password, "\n\r")
	}
	return
}
