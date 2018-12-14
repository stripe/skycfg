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

	"go.starlark.net/starlark"
	yaml "gopkg.in/yaml.v2"
)

// YamlModule returns a Starlark module for YAML helpers.
func YamlModule() starlark.Value {
	return &Module{
		Name: "yaml",
		Attrs: starlark.StringDict{
			"marshal": yamlMarshal(),
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
