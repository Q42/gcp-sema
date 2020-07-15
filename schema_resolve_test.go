package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const exampleSchema2 = `{
  "log": {
    "level": { "format": "String", "default": "info", "env": "LOG_LEVEL" },
  },
  "redis": {
    "shards": { "default": null, "format": "Array", "doc": "bla", "env": "REDIS_SHARDS" },
  }
}`

func TestSchemaResolving(t *testing.T) {
	config, err := parseSchema([]byte(exampleSchema2))
	assert.Equal(t, nil, err)

	secretManagerNonprefixed := []string{"projects/foobar/secrets/redis_shards"}
	secretManagerPrefixed := []string{"projects/foobar/secrets/myapp4_redis_shards"}

	resolved := schemaResolveSecrets(config, nil)
	assert.IsType(t, resolvedSecretRuntime{}, resolved["log.level"])
	assert.IsType(t, resolvedSecretRuntime{}, resolved["redis.shards"])

	logConfig := resolved["log.level"].(resolvedSecretRuntime).conf
	shardConfig := resolved["redis.shards"].(resolvedSecretRuntime).conf

	////////////////
	// One by one //
	////////////////

	// Non prefixed
	RenderPrefix = ""
	result, _, err := schemaResolveSecret(shardConfig, secretManagerNonprefixed)
	assert.Equal(t, resolvedSecretSema{key: "redis_shards"}, result)
	assert.Equal(t, nil, err)

	// Prefixed
	RenderPrefix = "myapp4"
	result, _, err = schemaResolveSecret(shardConfig, secretManagerPrefixed)
	assert.IsType(t, resolvedSecretSema{}, result)
	assert.Equal(t, resolvedSecretSema{key: "myapp4_redis_shards"}, result)
	assert.Equal(t, nil, err)

	//////////////////////
	// Joined in 1 call //
	//////////////////////

	// Non prefixed
	RenderPrefix = ""
	resolved = schemaResolveSecrets(config, secretManagerNonprefixed)
	assert.IsType(t, resolvedSecretRuntime{}, resolved["log.level"])
	assert.IsType(t, resolvedSecretSema{}, resolved["redis.shards"])
	assert.EqualValues(t, resolvedSecretRuntime{conf: logConfig}, resolved["log.level"])
	assert.EqualValues(t, resolvedSecretSema{key: "redis_shards"}, resolved["redis.shards"])

	// Prefixed
	RenderPrefix = "myapp4"
	resolved = schemaResolveSecrets(config, secretManagerPrefixed)
	assert.IsType(t, resolvedSecretSema{}, resolved["redis.shards"])
	assert.EqualValues(t, resolvedSecretSema{key: "myapp4_redis_shards"}, resolved["redis.shards"])
}
