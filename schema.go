package main

import (
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/flynn/json5"
)

func parseSchemaFile(schemaFile string) convictConfigSchema {
	data, err := ioutil.ReadFile(schemaFile)
	panicIfErr(err)
	return parseSchema(data)
}

func parseSchema(data []byte) convictConfigSchema {
	dest := &convictJSONTree{}
	err := json5.Unmarshal(data, &dest)
	panicIfErr(err)

	return convictConfigSchema{
		// rawJSONSchema:      data,
		flatConfigurations: convictRecursiveResolve(dest),
	}
}

type convictJSONTree struct {
	Leaf     *convictConfiguration
	Children []*convictJSONTree
}

func (tree *convictJSONTree) Nest(key string) {
	if tree.Leaf != nil {
		tree.Leaf.Path = append([]string{key}, tree.Leaf.Path...)
	}
	for _, c := range tree.Children {
		c.Nest(key)
	}
}

func (tree *convictJSONTree) UnmarshalJSON(data []byte) error {
	convict := struct {
		Default jsonValue `json:"default"`
	}{}

	// Parse to decide if it is a convict propery
	err := json5.Unmarshal(data, &convict)
	if err != nil {
		return err
	}

	// This is a convict configuration property: parse as map[string]interface{}
	if convict.Default.Set {
		obj := map[string]interface{}{}
		err := json5.Unmarshal(data, &obj)
		if err != nil {
			return err
		}
		_, format := isConvictLeaf(obj)
		tree.Leaf = &convictConfiguration{
			Format:       format,
			DefaultValue: convict.Default.Value,
			Doc:          toString(obj["doc"]),
			Env:          toString(obj["env"]),
		}
		return nil
	}

	// Else, this is a nested tree, parse items as nested things
	childs := map[string]*convictJSONTree{}
	err = json5.Unmarshal(data, &childs)
	if err != nil {
		return err
	}

	// Apply stable ordering of tree.Children,
	// otherwise testing is a nightmare.
	keys := []string{}
	for key := range childs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		v := childs[key]
		v.Nest(key)
		tree.Children = append(tree.Children, v)
	}

	return nil
}

type convictConfigSchema struct {
	rawJSONSchema      []byte
	flatConfigurations []convictConfiguration
}

type convictConfiguration struct {
	Path         []string
	Format       convictFormat
	DefaultValue interface{} `json:"default"`
	Doc          string      `json:"doc"`
	Env          string      `json:"env"`
}

// Convict supports nested properties. Everything with a "default" property
func isConvictLeaf(data map[string]interface{}) (hasFormat bool, format convictFormat) {
	switch v := data["format"].(type) {
	case string:
		hasFormat = true
		switch v {
		case "port":
			format = convictFormatPort{}
		case "Boolean":
			format = convictFormatBoolean{}
		case "Number":
			fallthrough
		case "int":
			format = convictFormatInt{v}
		case "Array":
			format = convictFormatArray{}
		case "String":
			fallthrough
		case "string-file-exists":
			fallthrough
		case "string-optional":
			fallthrough
		case "string-optional-locally":
			format = convictFormatString{actualFormat: v}
		default:
			panic(fmt.Errorf("Unknown format %s", v))
		}
		format = convictFormatString{actualFormat: v}
	case []interface{}:
		if strs, isAllString := allStrings(v); isAllString {
			hasFormat = true
			format = convictFormatString{
				actualFormat:   v,
				possibleValues: strs,
			}
		}
	default:
		hasFormat = false
	}

	return hasFormat, format
}

func convictRecursiveResolve(data *convictJSONTree) []convictConfiguration {
	configs := []convictConfiguration{}
	if data.Leaf != nil {
		configs = append(configs, *data.Leaf)
	}
	for _, v := range data.Children {
		configs = append(configs, convictRecursiveResolve(v)...)
	}
	return configs
}

func asString(input interface{}) (string, bool) {
	if input != nil {
		str, isStr := input.(string)
		if isStr {
			return str, true
		}
	}
	return "", false
}

func toString(input interface{}) string {
	str, _ := asString(input)
	return str
}

// ToSecretManagerKeys

func allStrings(input []interface{}) (out []string, allString bool) {
	allString = true
	for _, v := range input {
		if str, isStr := v.(string); isStr {
			out = append(out, str)
		} else {
			allString = false
		}
	}
	return
}

type jsonValue struct {
	Value interface{}
	Valid bool
	Set   bool
}

func (i *jsonValue) UnmarshalJSON(data []byte) error {
	// If this method was called, the value was set.
	i.Set = true

	if string(data) == "null" {
		// The key was set to null
		i.Valid = false
		return nil
	}

	// The key isn't set to null
	var temp interface{}
	if err := json5.Unmarshal(data, &temp); err != nil {
		return err
	}
	i.Value = temp
	i.Valid = true
	return nil
}
