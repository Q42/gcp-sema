package secretmanager

import (
	"github.com/pkg/errors"
)

// CatchAllClient -
type CatchAllClient struct{}

// CatchAllFlexibleKVValue -
type CatchAllFlexibleKVValue struct{}

/* check that types implement the interfaces */

var (
	_ KVClient = &CatchAllClient{}
	_ KVValue  = &CatchAllFlexibleKVValue{}
)

/* interface implementations */

func (*CatchAllClient) ListKeys() ([]KVValue, error) {
	return []KVValue{&CatchAllFlexibleKVValue{}}, nil
}
func (*CatchAllClient) Get(name string) (KVValue, error) {
	return &CatchAllFlexibleKVValue{}, nil
}
func (*CatchAllClient) New(name string, labels map[string]string) (KVValue, error) {
	return nil, errors.New("Not implemented")
}

func (*CatchAllFlexibleKVValue) GetFullName() string          { return "fullname-fake" }
func (*CatchAllFlexibleKVValue) GetShortName() string         { return "short-fake" }
func (*CatchAllFlexibleKVValue) GetValue() ([]byte, error)    { return nil, nil }
func (*CatchAllFlexibleKVValue) GetLabels() map[string]string { panic(errors.New("Not implemented")) }
func (*CatchAllFlexibleKVValue) SetLabels(labels map[string]string) error {
	return errors.New("Not implemented")
}
func (*CatchAllFlexibleKVValue) SetValue([]byte) (string, error) {
	return "", errors.New("Not implemented")
}
