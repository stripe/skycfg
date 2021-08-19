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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// newMessageType creates a Starlark value representing a named Protobuf message
// type that can be used for constructing new concrete protobuf objects.
func newMessageType(registry *protoregistry.Types, msg protoreflect.ProtoMessage) starlark.Callable {
	attrs := make(starlark.StringDict)

	descriptor := msg.ProtoReflect().Type().Descriptor()

	// Register child messages, enums as attrs
	for ii := 0; ii < descriptor.Enums().Len(); ii++ {
		child := descriptor.Enums().Get(ii)
		attrs[string(child.Name())] = newEnumType(child)
	}
	for ii := 0; ii < descriptor.Messages().Len(); ii++ {
		child := descriptor.Messages().Get(ii)
		if !child.IsMapEntry() {
			childMsg, err := registry.FindMessageByName(child.FullName())
			if err != nil {
				// Fallback to dynamicpb if nested message is
				// unavailable in registry. This points to the
				// registry having been incompletely
				// constructed, missing nested types
				childMsg = dynamicpb.NewMessageType(child)
			}

			attrs[string(child.Name())] = newMessageType(registry, childMsg.New().Interface())
		}
	}

	emptyMsg := proto.Clone(msg)
	proto.Reset(emptyMsg)

	return &protoMessageType{
		descriptor: descriptor,
		attrs:      attrs,
		emptyMsg:   emptyMsg,
	}
}

type protoMessageType struct {
	descriptor protoreflect.MessageDescriptor
	attrs      starlark.StringDict

	// An empty protobuf message of the appropriate type.
	emptyMsg protoreflect.ProtoMessage
}

var _ starlark.HasAttrs = (*protoMessageType)(nil)
var _ starlark.Callable = (*protoMessageType)(nil)
var _ skyProtoMessageType = (*protoMessageType)(nil)

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

	// Parse the kwarg set into a map[string]starlark.Value, containing one
	// entry for each provided kwarg. Keys are the original protobuf field names.
	// This lets the starlark kwarg parser handle most of the error reporting,
	// except type errors which are deferred until later.
	var parserPairs []interface{}
	parsedKwargs := make(map[string]*starlark.Value, len(kwargs))

	fields := t.descriptor.Fields()
	for ii := 0; ii < fields.Len(); ii++ {
		fieldName := string(fields.Get(ii).Name())
		v := new(starlark.Value)
		parsedKwargs[fieldName] = v
		parserPairs = append(parserPairs, fieldName+"?", v)
	}

	if err := starlark.UnpackArgs(t.Name(), nil, kwargs, parserPairs...); err != nil {
		return nil, err
	}

	// Instantiate a new message and populate the fields
	out, err := NewMessage(t.emptyMsg)
	if err != nil {
		return nil, err
	}
	for fieldName, starlarkValue := range parsedKwargs {
		if *starlarkValue == nil {
			continue
		}

		if err := out.SetField(fieldName, *starlarkValue); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (t *protoMessageType) NewMessage() protoreflect.ProtoMessage {
	return proto.Clone(t.emptyMsg)
}
