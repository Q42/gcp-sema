package main

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/flynn/json5"
	"github.com/go-errors/errors"
)

func parseSchemaFile(schemaFile string) convictConfigSchema {
	data, err := ioutil.ReadFile(schemaFile)
	panicIfErr(err)
	var schema convictConfigSchema
	schema, err = parseSchema(data)
	if err != nil {
		panic(errors.WrapPrefix(err, fmt.Sprintf("cannot parse schema '%s'", schemaFile), 0))
	}
	return schema
}

func parseSchema(data []byte) (result convictConfigSchema, err error) {
	dest := &convictJSONTree{}
	err = json5.Unmarshal(data, &dest)
	result = convictConfigSchema{
		// rawJSONSchema:      data,
		tree:               dest,
		flatConfigurations: convictRecursiveResolve(dest),
	}
	return
}

type convictJSONTree struct {
	Leaf     *convictConfiguration
	Children map[string]*convictJSONTree
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
		// Ignore errors like this!
		// A map could contain an additional key like `doc: "some explanation"`,
		// where "some explanation" does not fit the struct above
		return nil
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
	tree.Children = map[string]*convictJSONTree{}
	err = json5.Unmarshal(data, &tree.Children)
	for key, c := range tree.Children {
		if c != nil {
			c.Nest(key)
		} else {
			delete(tree.Children, key) // prevents issues down the line
		}
	}
	return err
}

type convictConfigSchema struct {
	rawJSONSchema      []byte
	tree               *convictJSONTree
	flatConfigurations []convictConfiguration
}

type convictConfiguration struct {
	Path         []string
	Format       convictFormat
	DefaultValue interface{} `json:"default"`
	Doc          string      `json:"doc"`
	Env          string      `json:"env"`
	Optional     bool
}

// Key is the standardized way of serializing a convictConfiguration.Path
func (conf *convictConfiguration) Key() string {
	return strings.Join(conf.Path, ".")
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
		case "int-optional":
			format = convictFormatInt{v}
		case "int":
			format = convictFormatInt{v}
		case "Array":
			format = convictFormatArray{}
		case "email":
			fallthrough
		case "url":
			fallthrough
		case "String":
			fallthrough
		case "string-file-exists":
			fallthrough
		case "string-optional":
			fallthrough
		case "string-optional-locally":
			format = convictFormatString{actualFormat: v}
		case "*":
			format = convictFormatAny{}
		default:
			panic(fmt.Errorf("Unknown format %s", v))
		}
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
	if data == nil {
		return nil
	}

	configs := []convictConfiguration{}
	if data.Leaf != nil {
		configs = append(configs, *data.Leaf)
	}
	// Apply stable ordering of tree.Children,
	// otherwise testing is a nightmare.
	var keys = make([]string, 0)
	for key := range data.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		v := data.Children[key]
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
