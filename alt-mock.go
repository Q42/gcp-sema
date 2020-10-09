package main

import (
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/go-errors/errors"
)

// This thing returns void data for ANY requested secret!
func prepareMockClient() (secretmanager.KVClient, SchemaResolver) {
	client := secretmanager.NewMockClient("projects/mock/secrets", "*", "")
	return client, catchAllResolver{}
}

type catchAllClient struct{}
type catchAllResolver struct{}
type catchAllFlexibleKVValue struct{}

/* check that types implement the interfaces */

var (
	_ secretmanager.KVClient = &catchAllClient{}
	_ secretmanager.KVValue  = &catchAllFlexibleKVValue{}
	_ SchemaResolver         = catchAllResolver{}
)

/* interface implementations */

func (*catchAllClient) ListKeys() ([]secretmanager.KVValue, error) {
	return []secretmanager.KVValue{&catchAllFlexibleKVValue{}}, nil
}
func (*catchAllClient) Get(name string) (secretmanager.KVValue, error) {
	return &catchAllFlexibleKVValue{}, nil
}
func (*catchAllClient) New(name string, labels map[string]string) (secretmanager.KVValue, error) {
	return nil, errors.New("Not implemented")
}

func (*catchAllFlexibleKVValue) GetFullName() string          { return "fullname-fake" }
func (*catchAllFlexibleKVValue) GetShortName() string         { return "short-fake" }
func (*catchAllFlexibleKVValue) GetValue() ([]byte, error)    { return nil, nil }
func (*catchAllFlexibleKVValue) GetLabels() map[string]string { panic(errors.New("Not implemented")) }
func (*catchAllFlexibleKVValue) SetLabels(labels map[string]string) error {
	return errors.New("Not implemented")
}
func (*catchAllFlexibleKVValue) SetValue([]byte) (string, error) {
	return "", errors.New("Not implemented")
}

func (catchAllResolver) Resolve(schema convictConfigSchema) map[string]resolvedSecret {
	allResolved := make(map[string]resolvedSecret, 0)
	for _, conf := range schema.flatConfigurations {
		if conf.DefaultValue != nil || conf.Env != "" || conf.Format.IsOptional() {
			continue
		}
		allResolved[conf.Key()] = resolvedSecretSema{
			key: convictToSemaKey("", conf.Path)[0],
			kv:  &catchAllFlexibleKVValue{},
		}
	}
	return allResolved
}
func (catchAllResolver) IsVerbose() bool                   { return false }
func (catchAllResolver) GetClient() secretmanager.KVClient { return &catchAllClient{} }
