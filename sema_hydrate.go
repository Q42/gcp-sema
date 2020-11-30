package main

import (
	"github.com/Q42/gcp-sema/pkg/dynamic"
	"github.com/Q42/gcp-sema/pkg/schema"
)

func hydrateSecretTree(tree *schema.ConvictJSONTree, resolved map[string]dynamic.ResolvedSecret) (outerResult interface{}, outerErr error) {
	if tree == nil {
		return nil, nil
	}
	if tree.Leaf != nil {
		resolved := resolved[tree.Leaf.Key()]
		if resolved == nil {
			return nil, nil // unresolved, TODO err?
		}
		val, err := resolved.GetSecretValue()
		return val, err
	}
	result := make(map[string]interface{}, 0)
	for key, c := range tree.Children {
		nested, err := hydrateSecretTree(c, resolved)
		if nested != nil {
			result[key] = nested
		}
		outerErr = multiAppend(outerErr, err)
	}
	if len(result) == 0 {
		return nil, outerErr
	}
	return result, outerErr
}

func injectSemaClient(handlers []concreteSecretHandler, schemaResolver schema.SchemaResolver) (returned []concreteSecretHandler) {
	for _, h := range handlers {
		returned = append(returned, h.injectSemaClient(schemaResolver))
	}
	return
}

func (c concreteSecretHandler) injectSemaClient(schemaResolver schema.SchemaResolver) concreteSecretHandler {
	switch s := c.SecretHandler.(type) {
	case *semaHandlerEnvironmentVariables:
		s.resolver = schemaResolver
		c.SecretHandler = s
	case *semaHandlerLiteral:
		s.resolver = schemaResolver
		c.SecretHandler = s
	case *semaHandlerSingleKey:
		s.resolver = schemaResolver
		c.SecretHandler = s
	}
	return c
}
