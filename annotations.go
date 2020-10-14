package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

var qnameExtCharFmtExcluded *regexp.Regexp = regexp.MustCompile("[^-a-zA-Z0-9_.]+")
var qnameExtChar = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func alfanum(inp string) string {
	return qnameExtCharFmtExcluded.ReplaceAllString(inp, "")
}

/**
 * Prevent errors like:
 * - metadata.annotations: Invalid value: "sema/source/config-env.json/CACHE.EXPIRATION": a qualified name must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]') with an optional DNS subdomain prefix and '/' (e.g. 'example.com/MyName')
 * - metadata.annotations: Invalid value: "sema/source.config-env.json.FEATURE_REDIS_LEGACY_SUBSCRIBEALLSHARDS_ENABLED": name part must be no more than 63 characters
 **/
func postProcessAnnotation(key, value string) (string, string, bool) {
	if key == "" {
		return "sema/source", value, true
	}

	key = fmt.Sprintf("sema/source.%s", alfanum(key))

	// Handle too long strings
	if len(key) > 63 {
		key = key[0:63]
	}

	// The last rune of the string must be a A-Za-z0-9 rune!
	key = strings.TrimRightFunc(key, func(r rune) bool {
		return !strings.ContainsRune(qnameExtChar, r)
	})

	// Log an error if we didn't succesfully generate it
	if errors := validation.IsQualifiedName(key); len(errors) != 0 {
		for _, v := range errors {
			os.Stderr.WriteString(fmt.Sprintf("Formatting annotation %q failed due to %q", key, v))
		}
		return "", "", false
	}
	return key, value, true
}
