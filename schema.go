package main

import (
	"io/ioutil"

	"github.com/flynn/json5"
)

func parseSchemaFile(schemaFile string) convictConfigSchema {
	data, err := ioutil.ReadFile(schemaFile)
	panicIfErr(err)
	return parseSchema(data)
}

func parseSchema(data []byte) convictConfigSchema {
	dest := map[string]interface{}{}
	err := json5.Unmarshal(data, &dest)
	panicIfErr(err)

	return convictConfigSchema{
		// rawJSONSchema:      data,
		flatConfigurations: convictRecursiveResolve(dest),
	}
}

type convictConfigSchema struct {
	rawJSONSchema      []byte
	flatConfigurations []convictConfiguration
}

type convictConfiguration struct {
	path         []string
	format       string
	defaultValue string
	doc          string
	env          string
}

func convictRecursiveResolve(data map[string]interface{}, path ...string) []convictConfiguration {
	format, hasFormat := asString(data["format"])
	defaultValue, hasDefault := asString(data["default"])

	if hasFormat || hasDefault {
		return []convictConfiguration{{
			path:         path,
			format:       format,
			defaultValue: defaultValue,
			doc:          toString(data["doc"]),
			env:          toString(data["env"]),
		}}
	}

	configs := []convictConfiguration{}
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			configs = append(configs, convictRecursiveResolve(v, append(path, key)...)...)
		}
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
