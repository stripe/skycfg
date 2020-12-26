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
	"google.golang.org/protobuf/reflect/protoreflect"
)

func newMessageType(descriptor protoreflect.MessageDescriptor) starlark.Callable {
	attrs := make(starlark.StringDict)

	for ii := 0; ii < descriptor.Enums().Len(); ii++ {
		child := descriptor.Enums().Get(ii)
		attrs[string(child.Name())] = newEnumType(child)
	}
	for ii := 0; ii < descriptor.Messages().Len(); ii++ {
		child := descriptor.Messages().Get(ii)
		if !child.IsMapEntry() {
			attrs[string(child.Name())] = newMessageType(child)
		}
	}

	return &protoMessageType{
		descriptor: descriptor,
		attrs:      attrs,
	}
}

type protoMessageType struct {
	descriptor protoreflect.MessageDescriptor
	attrs      starlark.StringDict
}

var _ starlark.HasAttrs = (*protoMessageType)(nil)
var _ starlark.Callable = (*protoMessageType)(nil)

func (t *protoMessageType) String() string {
	return fmt.Sprintf("<proto.MessageType %q>", t.Name())
}
func (t *protoMessageType) Type() string         { return "proto.MessageType" }
func (t *protoMessageType) Freeze()              {}
func (t *protoMessageType) Truth() starlark.Bool { return starlark.True }
func (t *protoMessageType) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", t.Type())
}

func (t *protoMessageType) Name() string {
	return string(t.descriptor.FullName())
}

func (t *protoMessageType) AttrNames() []string {
	names := make([]string, 0, len(t.attrs))
	for name := range t.attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (t *protoMessageType) Attr(attrName string) (starlark.Value, error) {
	if attr, ok := t.attrs[attrName]; ok {
		return attr, nil
	}
	fullName := t.descriptor.FullName().Append(protoreflect.Name(attrName))
	return nil, fmt.Errorf("Protobuf type %q not found", fullName)
}

func (t *protoMessageType) CallInternal(
	thread *starlark.Thread,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	// This is semantically the constructor of a protobuf message, and we
	// want it to accept only kwargs (where keys are protobuf field names).
	// Inject a useful error message if a user tries to pass positional args.
	if err := starlark.UnpackPositionalArgs(t.Name(), args, nil, 0); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("protoMessageType.CallInternal: not implemented yet")
}
