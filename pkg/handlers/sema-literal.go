package handlers

import (
	"fmt"
	"regexp"

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
	key    string
	secret string
	client secretmanager.KVClient
	//private
	cacheResolved ResolvedSecretSema
}

/* Test it conforms to interfaces */
var _ SecretHandler = &semaHandlerLiteral{}
var _ SecretHandlerWithSema = &semaHandlerLiteral{}

/* Implemented methods */
func (h *semaHandlerLiteral) InjectSemaClient(client secretmanager.KVClient, opts SecretHandlerOptions) {
	if opts.Mock {
		h.cacheResolved = ResolvedSecretSema{Key: h.secret, Client: h.client, KV: &secretmanager.CatchAllFlexibleKVValue{}}
		return
	}
	h.client = client
}

func (h *semaHandlerLiteral) Prepare(bucket map[string]bool) {
	if h.cacheResolved.KV == nil {
		secret, err := h.client.Get(h.secret)
		panicIfErr(err)
		h.cacheResolved = ResolvedSecretSema{Key: h.secret, Client: h.client, KV: secret}
	}
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
