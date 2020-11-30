package main

import (
	"github.com/Q42/gcp-sema/pkg/schema"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
)

// This thing returns void data for ANY requested secret!
func prepareMockClient() (secretmanager.KVClient, schema.SchemaResolver) {
	client := secretmanager.NewMockClient("projects/mock/secrets", "*", "")
	return client, schema.CatchAllResolver{}
}
