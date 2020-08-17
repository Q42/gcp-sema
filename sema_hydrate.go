package main

import (
	"strings"
)

func hydrateSecretTree(tree *convictJSONTree, resolved map[string]resolvedSecret) (outerResult interface{}, outerErr error) {
	if tree == nil {
		return nil, nil
	}
	if tree.Leaf != nil {
		resolved := resolved[strings.Join(tree.Leaf.Path, ".")]
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
