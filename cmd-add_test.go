package main

import (
	"testing"

	secretmanager "github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
)

func TestAddForceOverwrite(t *testing.T) {
	kv := secretmanager.NewMockClient("cl-test", "withlabels", "bar", "withoutlabels", "zulu")

	// Update secret with labels
	secret, _ := kv.Get("withlabels")
	secret.SetLabels(map[string]string{"a": "b"})
	cmdOpts := addCommand{Positional: addCommandPositional{"cl-test", "withlabels"}, Data: "baz1", Labels: map[string]string{"a": "b"}, client: kv, Force: []bool{true}}
	err := cmdOpts.Execute([]string{})
	assert.NoError(t, err)
	secretData, err := secret.GetValue()
	assert.Equal(t, "baz1", string(secretData))

	// Update secret with changed labels
	secret, _ = kv.Get("withlabels")
	secret.SetLabels(map[string]string{"a": "b"})
	cmdOpts = addCommand{Positional: addCommandPositional{"cl-test", "withlabels"}, Data: "baz1", Labels: map[string]string{"a": "different"}, client: kv, Force: []bool{true}}
	err = cmdOpts.Execute([]string{})
	assert.NoError(t, err)
	secretData, err = secret.GetValue()
	assert.Equal(t, "baz1", string(secretData))
	assert.Equal(t, map[string]string{"a": "different"}, secret.GetLabels())

	// Update secret without labels
	secret, _ = kv.Get("withoutlabels")
	cmdOpts = addCommand{Positional: addCommandPositional{"cl-test", "withoutlabels"}, Data: "baz2", client: kv, Force: []bool{true}}
	err = cmdOpts.Execute([]string{})
	assert.NoError(t, err)
	secretData, err = secret.GetValue()
	assert.Equal(t, "baz2", string(secretData))
}

func TestAddOverwriteFail(t *testing.T) {
	kv := secretmanager.NewMockClient("cl-test", "withlabels", "bar")

	// Update secret without specifying the same labels
	secret, _ := kv.Get("withlabels")
	secret.SetLabels(map[string]string{"a": "b"})
	cmdOpts := addCommand{Positional: addCommandPositional{"cl-test", "withlabels"}, Data: "baz1", client: kv}
	err := cmdOpts.Execute([]string{})
	assert.Error(t, err, "should throw an error about that labels must be the same")
	assert.Equal(t, err.Error(), `Please set the same labels, or use --force to update the already existing secret
  Existing labels: -l a:b
  New labels:      `)

	cmdOpts = addCommand{Positional: addCommandPositional{"cl-test", "withlabels"}, Labels: map[string]string{"a": "different"}, Data: "baz1", client: kv}
	err = cmdOpts.Execute([]string{})
	assert.Error(t, err, "should throw an error about that labels must be the same")
	assert.Equal(t, err.Error(), `Please set the same labels, or use --force to update the already existing secret
  Existing labels: -l a:b
  New labels:      -l a:different`)
}

func TestAddNotExisting(t *testing.T) {
	kv := secretmanager.NewMockClient("cl-test", "notfoo", "bar")
	cmdOpts := addCommand{Positional: addCommandPositional{"cl-test", "foo"}, Data: "baz", Labels: map[string]string{"a": "b"}, client: kv}
	err := cmdOpts.Execute([]string{})
	assert.NoError(t, err)
	secret, _ := kv.Get("foo")
	secretData, err := secret.GetValue()
	assert.Equal(t, "baz", string(secretData))
}

func TestEqualLabels(t *testing.T) {
	assert.Equal(t, true, equalLabels(make(map[string]string, 0), nil))
	assert.Equal(t, true, equalLabels(nil, make(map[string]string, 0)))
	assert.Equal(t, true, equalLabels(map[string]string{"foo": "bar"}, map[string]string{"foo": "bar"}))
	assert.Equal(t, false, equalLabels(map[string]string{"foo": "bar"}, map[string]string{"john": "doe"}))
}
