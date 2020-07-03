package main

import "strings"

func hydrateSecretTree(tree *convictJSONTree, resolved map[string]resolvedSecret) interface{} {
	if tree == nil {
		return nil
	}
	if tree.Leaf != nil {
		resolved := resolved[strings.Join(tree.Leaf.Path, ".")]
		if resolved == nil {
			return nil
		}
		val := resolved.GetSecretValue()
		if val == nil {
			return nil
		}
		return val
	}
	result := make(map[string]interface{}, 0)
	for key, c := range tree.Children {
		nested := hydrateSecretTree(c, resolved)
		if nested != nil {
			result[key] = nested
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
