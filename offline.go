package main

import (
	"os"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/joho/godotenv"
)

func prepareOfflineClient(file string) (secretmanager.KVClient, Matcher) {
	if file == "*" {
		return secretmanager.NewMockClient("fake", "*", "na"), catchAllMatcher
	}

	fileReader, err := os.Open(file)
	panicIfErr(err)
	defer fileReader.Close()
	env, err := godotenv.Parse(fileReader)
	panicIfErr(err)
	var flatList []string
	for k, v := range env {
		flatList = append(flatList, k, v)
	}
	return secretmanager.NewMockClient(file, flatList...), defaultMatcher
}

type catchAllClient struct{}

var _ secretmanager.KVClient = &catchAllClient{}

func (*catchAllClient) ListKeys() ([]secretmanager.KVValue, error) {
	return []secretmanager.KVValue{&catchAllFlexibleKVValue{}}, nil
}
func (*catchAllClient) Get(name string) (secretmanager.KVValue, error) { return nil, nil }
func (*catchAllClient) New(name string, labels map[string]string) (secretmanager.KVValue, error) {
	return nil, nil
}

type catchAllFlexibleKVValue struct{}

var _ secretmanager.KVValue = &catchAllFlexibleKVValue{}

func (*catchAllFlexibleKVValue) GetFullName() string                      { return "fullname-fake" }
func (*catchAllFlexibleKVValue) GetShortName() string                     { return "short-fake" }
func (*catchAllFlexibleKVValue) GetValue() ([]byte, error)                { return nil, nil }
func (*catchAllFlexibleKVValue) GetLabels() map[string]string             { return nil }
func (*catchAllFlexibleKVValue) SetLabels(labels map[string]string) error { return nil }
func (*catchAllFlexibleKVValue) SetValue([]byte) (string, error)          { return "", nil }

func catchAllMatcher(conf convictConfiguration, s secretmanager.KVValue, key string) bool {
	if conf.Format.IsOptional() || conf.Env != "" || conf.DefaultValue != nil {
		return false
	}
	return !conf.Format.IsOptional()
}
