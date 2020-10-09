package secretmanager

import (
	"fmt"
)

type mockKVClient struct {
	prefix string
	data   map[string]*mockKVValue
}

type mockKVValue struct {
	client *mockKVClient
	key    string
	path   string
	values [][]byte
	labels map[string]string
}

var _ KVClient = &mockKVClient{}
var _ KVValue = &mockKVValue{}

// NewMockClient creates a handy mock stand-in for Secret Manager
func NewMockClient(project string, keyValues ...string) KVClient {
	prefix := fmt.Sprintf("project/%s/secrets", project)
	if len(keyValues)%2 != 0 {
		panic("Specify key & values in pairs!")
	}
	c := mockKVClient{prefix: prefix, data: make(map[string]*mockKVValue, 0)}
	for i := 0; i < len(keyValues); i += 2 {
		c.data[keyValues[i]] = &mockKVValue{
			client: &c,
			key:    keyValues[i],
			path:   fmt.Sprintf("%s/%s", c.prefix, keyValues[i]),
			values: [][]byte{[]byte(keyValues[i+1])},
			labels: make(map[string]string)}
	}
	return &c
}

func (c *mockKVClient) ListKeys() ([]KVValue, error) {
	list := []KVValue{}
	for _, v := range c.data {
		list = append(list, KVValue(v))
	}
	return list, nil
}

func (c *mockKVClient) Get(name string) (KVValue, error) {
	val, ok := c.data[name]
	if ok {
		return KVValue(val), nil
	}
	return nil, fmt.Errorf("404: %q", name)
}

func (c *mockKVClient) New(name string, labels map[string]string) (KVValue, error) {
	v := mockKVValue{
		client: c,
		key:    name,
		path:   fmt.Sprintf("%s/%s", c.prefix, name),
		values: [][]byte{},
		labels: labels}
	c.data[name] = &v
	return KVValue(&v), nil
}

func (v *mockKVValue) GetFullName() string {
	return v.path
}
func (v *mockKVValue) GetShortName() string {
	return v.key
}
func (v *mockKVValue) GetValue() ([]byte, error) {
	return v.values[len(v.values)-1], nil

}
func (v *mockKVValue) GetLabels() map[string]string {
	return v.labels
}
func (v *mockKVValue) SetLabels(l map[string]string) error {
	v.labels = l
	return nil
}
func (v *mockKVValue) SetValue(data []byte) (string, error) {
	v.values = append(v.values, data)
	return fmt.Sprintf("%s/%d", v.GetFullName(), len(v.values)), nil
}
