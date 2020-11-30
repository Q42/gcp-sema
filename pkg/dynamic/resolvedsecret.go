package dynamic

// ResolvedSecret is something that does not necessarily have a value at the start; it is a placeholder that can be resolved later.
type ResolvedSecret interface {
	String() string
	Annotation() string
	GetSecretValue() (interface{}, error)
}
