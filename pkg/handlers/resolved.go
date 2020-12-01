package handlers

import (
	"fmt"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
)

// ResolvedSecret is something that does not necessarily have a value at the start; it is a placeholder that can be resolved later.
type ResolvedSecret interface {
	String() string
	Annotation() string
	GetSecretValue() (interface{}, error)
}

// ResolvedSecretSema -
type ResolvedSecretSema struct {
	Key    string
	Client secretmanager.KVClient
	KV     secretmanager.KVValue
}

var _ ResolvedSecret = &ResolvedSecretSema{} // test interface adherence

func (r ResolvedSecretSema) Annotation() string {
	if r.KV != nil {
		return fmt.Sprintf("secretmanager(fullname: %s)", r.KV.GetFullName())
	}
	return r.String()
}

func (r ResolvedSecretSema) String() string {
	return fmt.Sprintf("secretmanager(key: %s)", r.Key)
}

func (r ResolvedSecretSema) GetSecretValue() (interface{}, error) {
	var err error
	secret := r.KV
	if secret == nil {
		secret, err = r.Client.Get(r.Key)
		if err != nil {
			return nil, err
		}
	}
	val, err := secret.GetValue()
	if err != nil {
		return nil, err
	}
	stringValue := string(val)
	return &stringValue, nil
}
