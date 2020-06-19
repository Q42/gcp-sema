package main

import (
	"io/ioutil"
)

// SecretHandler is a container for cli arg `--from-[handler]=[key]=[value]`
type SecretHandler interface {
	Populate(bucket map[string][]byte)
}

// MakeSecretHandler resolves the different kinds of handlers
func MakeSecretHandler(handler, name, value string) SecretHandler {
	switch handler {
	case "literal":
		return &literalHandler{key: name, value: value}
	case "file":
		return &fileHandler{key: name, file: value}
	default:
		return &unknownHandler{
			handler: handler,
			key:     name,
			value:   value,
		}

	}
}

type literalHandler struct {
	key   string
	value string
}

func (lh *literalHandler) Populate(bucket map[string][]byte) {
	bucket[lh.key] = []byte(lh.value)
}

type unknownHandler struct {
	handler string
	key     string
	value   string
}

func (lh *unknownHandler) Populate(bucket map[string][]byte) {
	panic("Not Implemented!")
}

type fileHandler struct {
	key  string
	file string
}

func (lh *fileHandler) Populate(bucket map[string][]byte) {
	data, err := ioutil.ReadFile(lh.file)
	panicIfErr(err)
	bucket[lh.key] = data
}
