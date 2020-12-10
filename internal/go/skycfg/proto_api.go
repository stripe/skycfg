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
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/golang/protobuf/proto"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	yaml "gopkg.in/yaml.v2"
)

// UNSTABLE extension point for configuring how protobuf messages are loaded.
//
// This will be stabilized after the go-protobuf v2 API has reached GA.
type ProtoRegistry interface {
	// UNSTABLE lookup from full protobuf message name to a Go type of the
	// generated message struct.
	UnstableProtoMessageType(name string) (reflect.Type, error)

	// UNSTABLE lookup from go-protobuf enum name to the name->value map.
	UnstableEnumValueMap(name string) map[string]int32
}

func NewProtoModule(registry ProtoRegistry) *ProtoModule {
	mod := &ProtoModule{
		Registry: registry,
		attrs: starlark.StringDict{
			// deprecated functions
			"from_yaml": starlark.NewBuiltin("proto.from_yaml", fnProtoFromYaml),
			"to_yaml":   starlark.NewBuiltin("proto.to_yaml", fnProtoToYaml),
		},
	}
	mod.attrs["package"] = starlark.NewBuiltin("proto.package", mod.fnProtoPackage)
	return mod
}

type ProtoModule struct {
	Registry ProtoRegistry
	attrs    starlark.StringDict
}

var _ starlark.HasAttrs = (*ProtoModule)(nil)

func (mod *ProtoModule) String() string       { return fmt.Sprintf("<module %q>", "proto") }
func (mod *ProtoModule) Type() string         { return "module" }
func (mod *ProtoModule) Freeze()              { mod.attrs.Freeze() }
func (mod *ProtoModule) Truth() starlark.Bool { return starlark.True }
func (mod *ProtoModule) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", mod.Type())
}

func (mod *ProtoModule) Attr(name string) (starlark.Value, error) {
	if val, ok := mod.attrs[name]; ok {
		return val, nil
	}
	return nil, nil
}

func (mod *ProtoModule) AttrNames() []string {
	var names []string
	for name := range mod.attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Implementation of the `proto.package()` built-in function.
//
// Note: doesn't do any sort of input validation, because the go-protobuf
// message registration data isn't currently exported in a useful way
// (see https://github.com/golang/protobuf/issues/623).
func (mod *ProtoModule) fnProtoPackage(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var packageName string
	if err := starlark.UnpackPositionalArgs("proto.package", args, kwargs, 1, &packageName); err != nil {
		return nil, err
	}
	return NewSkyProtoPackage(mod.Registry, packageName), nil
}

func wantSingleProtoMessage(fnName string, args starlark.Tuple, kwargs []starlark.Tuple, msg **skyProtoMessage) error {
	var val starlark.Value
	if err := starlark.UnpackPositionalArgs(fnName, args, kwargs, 1, &val); err != nil {
		return err
	}
	gotMsg, ok := val.(*skyProtoMessage)
	if !ok {
		return fmt.Errorf("%s: for parameter 1: got %s, want proto.Message", fnName, val.Type())
	}
	*msg = gotMsg
	return nil
}

// Implementation of the `proto.to_yaml()` built-in function. Returns the
// YAML-formatted content of a protobuf message.
func fnProtoToYaml(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg *skyProtoMessage
	if err := wantSingleProtoMessage("proto.to_yaml", args, kwargs, &msg); err != nil {
		return nil, err
	}
	jsonData, err := (protojson.MarshalOptions{
		UseProtoNames: true,
	}).Marshal(proto.MessageV2(msg.msg))
	if err != nil {
		return nil, err
	}
	var yamlMap yaml.MapSlice
	if err := yaml.Unmarshal(jsonData, &yamlMap); err != nil {
		return nil, err
	}
	yamlData, err := yaml.Marshal(yamlMap)
	if err != nil {
		return nil, err
	}
	return starlark.String(yamlData), nil
}

// Implementation of the `proto.from_yaml()` built-in function.
// Returns the Protobuf message for YAML-formatted content.
func fnProtoFromYaml(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msgType starlark.Value
	var value starlark.String
	if err := starlark.UnpackPositionalArgs("proto.from_yaml", args, kwargs, 2, &msgType, &value); err != nil {
		return nil, err
	}
	protoMsgType, ok := msgType.(*skyProtoMessageType)
	if !ok {
		return nil, fmt.Errorf("%s: for parameter 2: got %s, want proto.MessageType", "proto.from_yaml", msgType.Type())
	}
	var msgBody interface{}
	if err := yaml.Unmarshal([]byte(value), &msgBody); err != nil {
		return nil, err
	}
	msgBody, err := convertMapStringInterface("proto.from_yaml", msgBody)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(msgBody)
	if err != nil {
		return nil, err
	}
	msg := proto.Clone(protoMsgType.emptyMsg)
	msg.Reset()
	if err := (protojson.UnmarshalOptions{}).Unmarshal(jsonData, proto.MessageV2(msg)); err != nil {
		return nil, err
	}
	return NewSkyProtoMessage(msg), nil
}

// Coverts map[interface{}]interface{} into map[string]interface{} for json.Marshaler
func convertMapStringInterface(fnName string, val interface{}) (interface{}, error) {
	switch items := val.(type) {
	case map[interface{}]interface{}:
		result := map[string]interface{}{}
		for k, v := range items {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("%s: TypeError: value %s (type `%s') can't be assigned to type 'string'.", fnName, k, reflect.TypeOf(k))
			}
			value, err := convertMapStringInterface(fnName, v)
			if err != nil {
				return nil, err
			}
			result[key] = value
		}
		return result, nil
	case []interface{}:
		for k, v := range items {
			value, err := convertMapStringInterface(fnName, v)
			if err != nil {
				return nil, err
			}
			items[k] = value
		}
	}
	return val, nil
}
