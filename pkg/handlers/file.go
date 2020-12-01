package handlers

import (
	"fmt"
	"io/ioutil"
)

type fileHandler struct {
	key  string
	file string
	data []byte
}

func (h *fileHandler) Prepare(bucket map[string]bool) {
	var err error
	h.data, err = ioutil.ReadFile(h.file)
	panicIfErr(err)
	bucket[h.key] = true
}
func (h *fileHandler) Populate(bucket map[string][]byte) {
	bucket[h.key] = h.data
}
func (h *fileHandler) Annotate(annotate func(key string, value string)) {
	annotate(h.key, fmt.Sprintf("type=file,file=%s", h.file))
}
