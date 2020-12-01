package handlers

type literalHandler struct {
	key   string
	value string
}

func (h *literalHandler) Prepare(bucket map[string]bool) {
	bucket[h.key] = true
}
func (h *literalHandler) Populate(bucket map[string][]byte) {
	bucket[h.key] = []byte(h.value)
}
func (h *literalHandler) Annotate(annotate func(key string, value string)) {
	annotate(h.key, "type=literal")
}
