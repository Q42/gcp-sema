package handlers

import (
	"fmt"
	"io/ioutil"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
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
func (h *fileHandler) InjectClient(c secretmanager.KVClient) {
	// noop
}
