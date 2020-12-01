package schema

import (
	"github.com/Q42/gcp-sema/pkg/handlers"
	"github.com/Q42/gcp-sema/pkg/multierror"
)

func hydrateSecretTree(tree *ConvictJSONTree, resolved map[string]handlers.ResolvedSecret) (outerResult interface{}, outerErr error) {
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
		outerErr = multierror.MultiAppend(outerErr, err)
	}
	if len(result) == 0 {
		return nil, outerErr
	}
	return result, outerErr
}
