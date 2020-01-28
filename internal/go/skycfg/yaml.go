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
	"strings"

	"go.starlark.net/starlark"
	"gopkg.in/yaml.v3"
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

// starlarkToYAMLNode parses v into *yaml.Node.
func starlarkToYAMLNode(v starlark.Value) (*yaml.Node, error) {
	switch v := v.(type) {
	case starlark.NoneType:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "null",
		}, nil
	case starlark.Bool:
		return &yaml.Node{
			Kind: yaml.ScalarNode,
			// Both "True" and "true" are valid YAML but we'll use
			// lowercase to stay consistent with older API (yaml.v2).
			Value: strings.ToLower(v.String()),
		}, nil
	case starlark.Int:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: v.String(),
		}, nil
	case starlark.Float:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%g", v),
		}, nil
	case starlark.String:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: string(v),
		}, nil
	case starlark.Indexable: // Tuple, List
		var elems []*yaml.Node
		for i, n := 0, starlark.Len(v); i < n; i++ {
			nn, err := starlarkToYAMLNode(v.Index(i))
			if err != nil {
				return nil, fmt.Errorf("failed to convert %d-th element of %v to YAML: %v", i, v, err)
			}
			elems = append(elems, nn)
		}
		return &yaml.Node{
			Kind:    yaml.SequenceNode,
			Value:   v.String(),
			Content: elems,
		}, nil
	case *starlark.Dict:
		var elems []*yaml.Node
		for _, pair := range v.Items() {
			key, err := starlarkToYAMLNode(pair[0])
			if err != nil {
				return nil, fmt.Errorf("failed to convert key %v to YAML: %v", pair[0], err)
			}
			if key.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("key `%v' is not scalar", key.Value)
			}
			elems = append(elems, key)

			val, err := starlarkToYAMLNode(pair[1])
			if err != nil {
				return nil, fmt.Errorf("failed to convert value %v to YAML: %v", pair[1], err)
			}
			elems = append(elems, val)
		}
		return &yaml.Node{
			Kind:    yaml.MappingNode,
			Value:   v.String(),
			Content: elems,
		}, nil
	default:
		return nil, fmt.Errorf("TypeError: value %s (type `%s') can't be converted to YAML", v.String(), v.Type())
	}
}

func fnYamlMarshal(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var v starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &v); err != nil {
		return nil, err
	}

	node, err := starlarkToYAMLNode(v)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{node},
	}); err != nil {
		return nil, err
	}
	enc.Close()

	return starlark.String(buf.String()), nil
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
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(blob), &doc); err != nil {
		return nil, err
	}
	return toStarlarkValue(doc.Content[0])
}

// toStarlarkScalarValue converts a scalar node value to corresponding
// starlark.Value.
func toStarlarkScalarValue(node *yaml.Node) (starlark.Value, error) {
	var obj interface{}
	if err := node.Decode(&obj); err != nil {
		return nil, err
	}

	if obj == nil {
		return starlark.None, nil
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return starlark.MakeInt64(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return starlark.MakeUint64(v.Uint()), nil
	case reflect.Bool:
		return starlark.Bool(v.Bool()), nil
	case reflect.Float32, reflect.Float64:
		return starlark.Float(v.Float()), nil
	case reflect.String:
		return starlark.String(v.String()), nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", t)
	}
}

// toStarlarkValue is a DFS walk to translate the DAG from Go to Starlark.
func toStarlarkValue(node *yaml.Node) (starlark.Value, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return toStarlarkScalarValue(node)
	case yaml.MappingNode:
		out := &starlark.Dict{}
		for ki, vi := 0, 1; vi < len(node.Content); ki, vi = ki+2, vi+2 {
			k, v := node.Content[ki], node.Content[vi]

			kv, err := toStarlarkScalarValue(k)
			if err != nil {
				return nil, fmt.Errorf("`%s' not a supported key type: %v", k.Value, err)
			}

			vv, err := toStarlarkValue(v)
			if err != nil {
				return nil, err
			}

			if err = out.SetKey(kv, vv); err != nil {
				return nil, err
			}
		}
		return out, nil
	case yaml.SequenceNode:
		out := make([]starlark.Value, len(node.Content))
		for i, e := range node.Content {
			vv, err := toStarlarkValue(e)
			if err != nil {
				return nil, err
			}
			out[i] = vv
		}
		return starlark.NewList(out), nil
	default:
		return nil, fmt.Errorf("`%s' not a supported value", node.Value)
	}
}
