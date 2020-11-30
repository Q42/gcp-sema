package schema

import (
	"github.com/Q42/gcp-sema/pkg/dynamic"
	"github.com/Q42/gcp-sema/pkg/secretmanager"
)

// CatchAllResolver is a mock implementation
type CatchAllResolver struct{}

/* check that types implement the interfaces */

var (
	_ SchemaResolver = CatchAllResolver{}
)

/* interface implementations */

func (CatchAllResolver) Resolve(schema ConvictConfigSchema) map[string]dynamic.ResolvedSecret {
	allResolved := make(map[string]dynamic.ResolvedSecret, 0)
	for _, conf := range schema.FlatConfigurations {
		if conf.DefaultValue != nil || conf.Env != "" || conf.Format.IsOptional() {
			continue
		}
		allResolved[conf.Key()] = ResolvedSecretSema{
			Key: ConvictToSemaKey("", conf.Path)[0],
			KV:  &secretmanager.CatchAllFlexibleKVValue{},
		}
	}
	return allResolved
}
func (CatchAllResolver) IsVerbose() bool                   { return false }
func (CatchAllResolver) GetClient() secretmanager.KVClient { return &secretmanager.CatchAllClient{} }
