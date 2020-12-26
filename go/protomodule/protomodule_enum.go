// Copyright 2020 The Skycfg Authors.
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
	"sort"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func newEnumType(descriptor protoreflect.EnumDescriptor) starlark.Value {
	values := descriptor.Values()
	attrs := make(map[string]starlark.Value, values.Len())
	for ii := 0; ii < values.Len(); ii++ {
		value := values.Get(ii)
		attrs[string(value.Name())] = &protoEnumValue{
			typeName: descriptor.FullName(),
			value:    value,
		}
	}

	return &protoEnumType{
		name:  descriptor.FullName(),
		attrs: attrs,
	}
}

type protoEnumType struct {
	name  protoreflect.FullName
	attrs starlark.StringDict
}

var _ starlark.HasAttrs = (*protoEnumType)(nil)

func (t *protoEnumType) String() string {
	return fmt.Sprintf("<proto.EnumType %q>", t.name)
}
func (t *protoEnumType) Type() string         { return "proto.EnumType" }
func (t *protoEnumType) Freeze()              {}
func (t *protoEnumType) Truth() starlark.Bool { return starlark.True }
func (t *protoEnumType) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", t.Type())
}

func (t *protoEnumType) Attr(attrName string) (starlark.Value, error) {
	if attr, ok := t.attrs[attrName]; ok {
		return attr, nil
	}
	// TODO: better error message
	return nil, nil
}

func (t *protoEnumType) AttrNames() []string {
	names := make([]string, 0, len(t.attrs))
	for name := range t.attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type protoEnumValue struct {
	typeName protoreflect.FullName
	value    protoreflect.EnumValueDescriptor
}

var _ starlark.Comparable = (*protoEnumValue)(nil)

func (v *protoEnumValue) String() string {
	return fmt.Sprintf("<%s %s=%d>", v.typeName, v.value.Name(), v.value.Number())
}
func (v *protoEnumValue) Type() string         { return string(v.typeName) }
func (v *protoEnumValue) Freeze()              {}
func (v *protoEnumValue) Truth() starlark.Bool { return starlark.True }
func (v *protoEnumValue) Hash() (uint32, error) {
	return starlark.MakeInt64(int64(v.value.Number())).Hash()
}

func (v *protoEnumValue) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	other := y.(*protoEnumValue)
	switch op {
	case syntax.EQL:
		return v.value == other.value, nil
	case syntax.NEQ:
		return v.value != other.value, nil
	default:
		return false, fmt.Errorf("enums support only `==' and `!=' comparisons, got: %#v", op)
	}
}
