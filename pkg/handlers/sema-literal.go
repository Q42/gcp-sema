package handlers

import (
	"fmt"
	"regexp"

	"github.com/Q42/gcp-sema/pkg/schema"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
)

// TODO deduplicate
var qnameExtCharFmtExcluded *regexp.Regexp = regexp.MustCompile("[^-a-zA-Z0-9_.]+")

// TODO deduplicate
func alfanum(inp string) string {
	return qnameExtCharFmtExcluded.ReplaceAllString(inp, "")
}

// TODO deduplicate
// prints stack
func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

type semaHandlerLiteral struct {
	key      string
	secret   string
	resolver schema.SchemaResolver
	//private
	cacheResolved schema.ResolvedSecretSema
}

func (h *semaHandlerLiteral) Prepare(bucket map[string]bool) {
	secret, err := h.resolver.GetClient().Get(h.secret)
	panicIfErr(err)
	h.cacheResolved = schema.ResolvedSecretSema{Key: h.secret, Client: h.resolver.GetClient(), KV: secret}
	bucket[h.key] = true
}
func (h *semaHandlerLiteral) Populate(bucket map[string][]byte) {
	val, err := h.cacheResolved.GetSecretValue()
	panicIfErr(err)
	if stringVal, ok := val.(*string); ok {
		bucket[h.key] = []byte(*stringVal)
	}
}
func (h *semaHandlerLiteral) Annotate(annotate func(key string, value string)) {
	annotate(h.key, fmt.Sprintf("type=sema-literal,secret=%s", h.secret))
	annotate(fmt.Sprintf("%s.%s", h.key, alfanum(h.secret)), h.cacheResolved.Annotation())
}
func (h *semaHandlerLiteral) InjectClient(c secretmanager.KVClient) {
	// TODO
}
