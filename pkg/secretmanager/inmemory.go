package secretmanager

import (
	"fmt"
)

type memoryKVClient struct {
	prefix string
	data   map[string]*memoryKVValue
}

type memoryKVValue struct {
	client *memoryKVClient
	key    string
	path   string
	values [][]byte
	labels map[string]string
}

var _ KVClient = &memoryKVClient{}
var _ KVValue = &memoryKVValue{}

// NewInMemoryClient creates a handy stand-in for Secret Manager (for example for mocking)
func NewInMemoryClient(project string, keyValues ...string) KVClient {
	prefix := fmt.Sprintf("project/%s/secrets", project)
	if len(keyValues)%2 != 0 {
		panic("Specify key & values in pairs!")
	}
	c := memoryKVClient{prefix: prefix, data: make(map[string]*memoryKVValue, 0)}
	for i := 0; i < len(keyValues); i += 2 {
		c.data[keyValues[i]] = &memoryKVValue{
			client: &c,
			key:    keyValues[i],
			path:   fmt.Sprintf("%s/%s", c.prefix, keyValues[i]),
			values: [][]byte{[]byte(keyValues[i+1])},
			labels: make(map[string]string)}
	}
	return &c
}

func (c *memoryKVClient) ListKeys() ([]KVValue, error) {
	list := []KVValue{}
	for _, v := range c.data {
		list = append(list, KVValue(v))
	}
	return list, nil
}

func (c *memoryKVClient) Get(name string) (KVValue, error) {
	val, ok := c.data[name]
	if ok {
		return KVValue(val), nil
	}
	return nil, fmt.Errorf("404: %q", name)
}

func (c *memoryKVClient) New(name string, labels map[string]string) (KVValue, error) {
	v := memoryKVValue{
		client: c,
		key:    name,
		path:   fmt.Sprintf("%s/%s", c.prefix, name),
		values: [][]byte{},
		labels: labels}
	c.data[name] = &v
	return KVValue(&v), nil
}

func (v *memoryKVValue) GetFullName() string {
	return v.path
}
func (v *memoryKVValue) GetShortName() string {
	return v.key
}
func (v *memoryKVValue) GetValue() ([]byte, error) {
	return v.values[len(v.values)-1], nil

}
func (v *memoryKVValue) GetLabels() map[string]string {
	return v.labels
}
func (v *memoryKVValue) SetLabels(l map[string]string) error {
	v.labels = l
	return nil
}
func (v *memoryKVValue) SetValue(data []byte) (string, error) {
	v.values = append(v.values, data)
	return fmt.Sprintf("%s/%d", v.GetFullName(), len(v.values)), nil
}
