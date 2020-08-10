package secretmanager

// KVClient is a generic interface implemented by SecretManager and a mock
type KVClient interface {
	ListKeys() ([]KVValue, error)
	Get(name string) (KVValue, error)
	New(name string, labels map[string]string) (KVValue, error)
}

// KVValue represents a versions secret data storage
type KVValue interface {
	GetFullName() string
	GetShortName() string
	GetValue() ([]byte, error)
	GetLabels() map[string]string
	SetLabels(labels map[string]string) error
	SetValue([]byte) (string, error)
}
