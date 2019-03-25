// Copyright 2018 The Skycfg Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package skycfg

import (
	"bytes"
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
	yaml "gopkg.in/yaml.v2"
)

// YamlModule returns a Starlark module for YAML helpers.
func YamlModule() starlark.Value {
	return &Module{
		Name: "yaml",
		Attrs: starlark.StringDict{
			"marshal":   yamlMarshal(),
			"unmarshal": yamlUnmarshal(),
		},
	}
}

// yamlMarshal returns a Starlark function for marshaling plain values
// (dicts, lists, etc) to YAML.
//
//  def yaml.marshal(value) -> str
func yamlMarshal() starlark.Callable {
	return starlark.NewBuiltin("yaml.marshal", fnYamlMarshal)
}

func fnYamlMarshal(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var v starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeJSON(&buf, v); err != nil {
		return nil, err
	}
	var jsonObj interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &jsonObj); err != nil {
		return nil, err
	}
	yamlBytes, err := yaml.Marshal(jsonObj)
	if err != nil {
		return nil, err
	}
	return starlark.String(yamlBytes), nil
}

// yamlUnmarshal returns a Starlark function for unmarshaling yaml content to
// to starlark values.
//
// def yaml.unmarshal(yaml_content) -> (dicts, lists, etc)
func yamlUnmarshal() starlark.Callable {
	return starlark.NewBuiltin("yaml.unmarshal", fnYamlUnmarshal)
}

func fnYamlUnmarshal(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var blob string
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, nil, 1, &blob); err != nil {
		return nil, err
	}
	var inflated interface{}
	if err := yaml.Unmarshal([]byte(blob), &inflated); err != nil {
		return nil, err
	}
	return toStarlarkValue(inflated)
}

// toStarlarkValue is a DFS walk to translate the DAG from go to starlark
func toStarlarkValue(obj interface{}) (starlark.Value, error) {
	rt := reflect.TypeOf(obj)
	switch rt.Kind() {
	case reflect.String:
		return starlark.String(obj.(string)), nil
	case reflect.Map:
		ret := &starlark.Dict{}
		for k, v := range obj.(map[interface{}]interface{}) {
			starval, err := toStarlarkValue(v)
			if err != nil {
				return nil, err
			}
			if err = ret.SetKey(starlark.String(k.(string)), starval); err != nil {
				return nil, err
			}
		}
		return ret, nil
	case reflect.Slice:
		slice := obj.([]interface{})
		starvals := make([]starlark.Value, len(slice))
		for i, element := range slice {
			v, err := toStarlarkValue(element)
			if err != nil {
				return nil, err
			}
			starvals[i] = v
		}
		return starlark.NewList(starvals), nil
	default:
		return nil, fmt.Errorf("%v is not a slice, map, or string", obj)
	}
}
