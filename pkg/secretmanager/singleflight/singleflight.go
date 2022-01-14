// Singleflight is a small wrapper around KVClient that prevents concurrent requests on the same entities.
// Both listing, getting secrets and getting the values runs using golang.org/x/sync/singleflight.
package singleflight

import (
	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"golang.org/x/sync/singleflight"
)

func New(c secretmanager.KVClient) secretmanager.KVClient {
	return &semaSingleFlightClient{KVClient: c, sf: &singleflight.Group{}}
}

type semaSingleFlightClient struct {
	secretmanager.KVClient
	sf *singleflight.Group
}

var _ secretmanager.KVClient = &semaSingleFlightClient{}

func (c *semaSingleFlightClient) ListKeys() ([]secretmanager.KVValue, error) {
	result, err, _ := c.sf.Do("list", func() (interface{}, error) {
		return c.KVClient.ListKeys()
	})
	if err != nil {
		return nil, err
	}
	// Wrap returned kv's
	castedResult := result.([]secretmanager.KVValue)
	for i := range castedResult {
		castedResult[i] = &semaSingleFlightClientKeyValue{client: c, KVValue: castedResult[i]}
	}
	return castedResult, nil
}

func (c *semaSingleFlightClient) Get(name string) (secretmanager.KVValue, error) {
	result, err, _ := c.sf.Do(name, func() (interface{}, error) {
		return c.KVClient.Get(name)
	})
	if err != nil {
		return nil, err
	}
	// Wrap returned kv
	return &semaSingleFlightClientKeyValue{KVValue: result.(secretmanager.KVValue), client: c}, nil
}

func (c *semaSingleFlightClient) New(name string, labels map[string]string) (secretmanager.KVValue, error) {
	v, err := c.KVClient.New(name, labels)
	if err != nil {
		return nil, err
	}
	// Wrap returned kv
	return &semaSingleFlightClientKeyValue{KVValue: v, client: c}, nil
}

type semaSingleFlightClientKeyValue struct {
	secretmanager.KVValue
	client *semaSingleFlightClient
}

func (sf *semaSingleFlightClientKeyValue) GetValue() ([]byte, error) {
	dataInterface, err, _ := sf.client.sf.Do(sf.KVValue.GetFullName(), func() (interface{}, error) {
		return sf.KVValue.GetValue()
	})
	if err != nil {
		return nil, err
	}
	return dataInterface.([]byte), nil
}
