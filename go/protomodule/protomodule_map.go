// Copyright 2021 The Skycfg Authors.
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

package protomodule

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var allowedDictMethods = map[string]func(*protoMap) starlark.Value{
	"clear":      nil,
	"get":        nil,
	"items":      nil,
	"keys":       nil,
	"setdefault": (*protoMap).wrapSetDefault,
	"update":     (*protoMap).wrapUpdate,
	"values":     nil,
}

// protoMap wraps an underlying starlark.Dict to enforce typechecking
type protoMap struct {
	mapKey   protoreflect.FieldDescriptor
	mapValue protoreflect.FieldDescriptor
	dict     *starlark.Dict
}

var _ starlark.Value = (*protoMap)(nil)
var _ starlark.Iterable = (*protoMap)(nil)
var _ starlark.Sequence = (*protoMap)(nil)
var _ starlark.HasAttrs = (*protoMap)(nil)
var _ starlark.HasSetKey = (*protoMap)(nil)
var _ starlark.Comparable = (*protoMap)(nil)

func newProtoMap(mapKey protoreflect.FieldDescriptor, mapValue protoreflect.FieldDescriptor) *protoMap {
	return &protoMap{
		mapKey:   mapKey,
		mapValue: mapValue,
		dict:     starlark.NewDict(0),
	}
}

func newProtoMapFromDict(mapKey protoreflect.FieldDescriptor, mapValue protoreflect.FieldDescriptor, d *starlark.Dict) (*protoMap, error) {
	out := &protoMap{
		mapKey:   mapKey,
		mapValue: mapValue,
		dict:     d,
	}

	for _, item := range d.Items() {
		err := out.typeCheck(item[0], item[1])
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (m *protoMap) Attr(name string) (starlark.Value, error) {
	wrapper, ok := allowedDictMethods[name]
	if !ok {
		return nil, nil
	}
	if wrapper != nil {
		return wrapper(m), nil
	}
	return m.dict.Attr(name)
}

func (m *protoMap) AttrNames() []string                                { return m.dict.AttrNames() }
func (m *protoMap) Freeze()                                            { m.dict.Freeze() }
func (m *protoMap) Hash() (uint32, error)                              { return m.dict.Hash() }
func (m *protoMap) Get(k starlark.Value) (starlark.Value, bool, error) { return m.dict.Get(k) }
func (m *protoMap) Iterate() starlark.Iterator                         { return m.dict.Iterate() }
func (m *protoMap) Len() int                                           { return m.dict.Len() }
func (m *protoMap) String() string                                     { return m.dict.String() }
func (m *protoMap) Truth() starlark.Bool                               { return m.dict.Truth() }
func (m *protoMap) Items() []starlark.Tuple                            { return m.dict.Items() }

func (m *protoMap) Type() string {
	return fmt.Sprintf("map<%s, %s>", typeName(m.mapKey), typeName(m.mapValue))
}

func (m *protoMap) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	other, ok := y.(*protoMap)
	if !ok {
		return false, nil
	}

	return starlark.CompareDepth(op, m.dict, other.dict, depth)
}

func (m *protoMap) wrapSetDefault() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var key, defaultValue starlark.Value = nil, starlark.None
		if err := starlark.UnpackPositionalArgs("setdefault", args, kwargs, 1, &key, &defaultValue); err != nil {
			return nil, err
		}
		if val, ok, err := m.dict.Get(key); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return defaultValue, m.SetKey(key, defaultValue)
	}
	return starlark.NewBuiltin("setdefault", impl).BindReceiver(m)
}

func (m *protoMap) wrapUpdate() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// Use the underlying starlark `dict.update()` to get a Dict containing
		// all the new values, so we don't have to recreate the API here. After
		// the temp dict is constructed, type check.
		tempDict := &starlark.Dict{}
		tempUpdate, _ := tempDict.Attr("update")
		if _, err := starlark.Call(thread, tempUpdate, args, kwargs); err != nil {
			return nil, err
		}
		for _, item := range tempDict.Items() {
			if err := m.SetKey(item[0], item[1]); err != nil {
				return nil, err
			}
		}

		return starlark.None, nil
	}
	return starlark.NewBuiltin("update", impl).BindReceiver(m)
}

func (m *protoMap) SetKey(k, v starlark.Value) error {
	err := m.typeCheck(k, v)
	if err != nil {
		return err
	}

	err = m.dict.SetKey(k, v)
	if err != nil {
		return err
	}

	return nil
}

func (m *protoMap) typeCheck(k, v starlark.Value) error {
	err := scalarTypeCheck(m.mapKey, k)
	if err != nil {
		return err
	}

	err = scalarTypeCheck(m.mapValue, v)
	if err != nil {
		return err
	}

	return nil
}
