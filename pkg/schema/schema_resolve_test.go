package schema

import (
	"testing"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
)

const exampleSchema2 = `{
  "log": {
    "level": { "format": "String", "default": "info", "env": "LOG_LEVEL" },
  },
  "redis": {
    "shards": { "default": null, "format": "Array", "doc": "bla", "env": "REDIS_SHARDS" },
  },
  "encryption": {
	"ssh_key": { "default": null, "format": "string-optional" },
	"opt_int": { "default": null, "format": "int-optional" },
  }
}`

func TestSchemaResolving(t *testing.T) {
	config, err := parseSchema([]byte(exampleSchema2))
	assert.Equal(t, nil, err)

	secretManagerNonprefixed := secretmanager.NewMockClient("my-project", "redis_shards", "1,2,3,4,5")
	secretManagerPrefixed := secretmanager.NewMockClient("my-project", "myapp4_redis_shards", "a,b,c,d,e")

	resolved := schemaResolver{Client: secretmanager.NewMockClient("my-project")}.Resolve(config)
	assert.IsType(t, resolvedSecretRuntime{}, resolved["log.level"])
	assert.IsType(t, resolvedSecretRuntime{}, resolved["redis.shards"])

	logConfig := resolved["log.level"].(resolvedSecretRuntime).conf
	shardConfig := resolved["redis.shards"].(resolvedSecretRuntime).conf
	encryptionConfig := resolved["encryption.ssh_key"].(resolvedSecretRuntime).conf
	optIntConfig := resolved["encryption.opt_int"].(resolvedSecretRuntime).conf

	////////////////
	// One by one //
	////////////////

	// Non prefixed
	keys, _ := secretManagerNonprefixed.ListKeys()
	result, _, err := schemaResolver{Prefix: ""}.resolveConf(shardConfig, keys)
	assert.IsType(t, ResolvedSecretSema{}, result)
	assert.Equal(t, "redis_shards", result.(ResolvedSecretSema).Key)
	assert.Equal(t, keys[0], result.(ResolvedSecretSema).KV)
	assert.Equal(t, nil, err)

	// Prefixed
	keys, _ = secretManagerPrefixed.ListKeys()
	result, _, err = schemaResolver{Prefix: "myapp4"}.resolveConf(shardConfig, keys)
	assert.IsType(t, ResolvedSecretSema{}, result)
	assert.Equal(t, "myapp4_redis_shards", result.(ResolvedSecretSema).Key)
	assert.Equal(t, keys[0], result.(ResolvedSecretSema).KV)
	assert.Equal(t, nil, err)

	//////////////////////
	// Joined in 1 call //
	//////////////////////

	// Non prefixed
	resolved = schemaResolver{Client: secretManagerNonprefixed, Prefix: ""}.Resolve(config)
	assert.IsType(t, resolvedSecretRuntime{}, resolved["log.level"])
	assert.IsType(t, resolvedSecretRuntime{}, resolved["encryption.ssh_key"])
	assert.IsType(t, resolvedSecretRuntime{}, resolved["encryption.opt_int"])
	assert.IsType(t, ResolvedSecretSema{}, resolved["redis.shards"])
	assert.EqualValues(t, resolvedSecretRuntime{conf: logConfig}, resolved["log.level"])
	assert.EqualValues(t, resolvedSecretRuntime{conf: encryptionConfig}, resolved["encryption.ssh_key"])
	assert.EqualValues(t, resolvedSecretRuntime{conf: optIntConfig}, resolved["encryption.opt_int"])
	assert.IsType(t, ResolvedSecretSema{}, resolved["redis.shards"])
	assert.EqualValues(t, "redis_shards", resolved["redis.shards"].(ResolvedSecretSema).Key)
	assert.EqualValues(t, secretManagerNonprefixed, resolved["redis.shards"].(ResolvedSecretSema).Client)

	// Prefixed
	resolved = schemaResolver{Client: secretManagerPrefixed, Prefix: "myapp4"}.Resolve(config)
	assert.IsType(t, ResolvedSecretSema{}, resolved["redis.shards"])
	assert.EqualValues(t, "myapp4_redis_shards", resolved["redis.shards"].(ResolvedSecretSema).Key)
	assert.EqualValues(t, secretManagerPrefixed, resolved["redis.shards"].(ResolvedSecretSema).Client)
}
