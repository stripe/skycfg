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
	"sort"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// NewMessage returns a Starlark value representing the given Protobuf
// message. It can be returned back to a proto.Message() via AsProtoMessage().
//
// NewMessage copies the input proto.Message and therefore does not modify it
func NewMessage(msg proto.Message) (*protoMessage, error) {
	msgReflect := msg.ProtoReflect()

	fields := make(map[string]starlark.Value)

	// Copy any existing set fields
	var rangeErr error
	msgReflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		// Protobuf field presense is complex: https://github.com/protocolbuffers/protobuf/blob/d67f921f90e3aa03f65c3e1e507ca7017c8327a6/docs/field_presence.md
		// There are two different manifestations of presence for protobufs: no
		// presence, where the generated message API stores field values (only),
		// and explicit presence, where the API also stores whether or not a
		// field has been set.
		// For fields with explicit presence, we trust `Range`: if it iterates over them, we keep the field.
		// For fields with no presence, we manually double-check whether the field is equal to the default value. If so, we omit it.
		if fd.HasPresence() || isFieldSet(v, fd) {
			starlarkValue, err := valueToStarlark(v, fd)
			if err != nil {
				rangeErr = err
				return false
			}
			fields[string(fd.Name())] = starlarkValue
		}
		return true
	})
	if rangeErr != nil {
		return nil, rangeErr
	}

	// Clone and reset the input msg to ensure no mutations
	cloned := proto.Clone(msg)
	proto.Reset(cloned)

	return &protoMessage{
		msg:     cloned,
		msgDesc: msgReflect.Descriptor(),
		fields:  fields,
		frozen:  false,
	}, nil
}

// AsProtoMessage returns a Protobuf message underlying the given Starlark
// value, which must have been created by NewProtoMessage(). Returns
// (_, false) if the value is not a valid message.
func AsProtoMessage(v starlark.Value) (proto.Message, bool) {
	if msg, ok := v.(*protoMessage); ok {
		return msg.toProtoMessage(), true
	}
	return nil, false
}

// protoMessage exposes an underlying protobuf message as a starlark.Value
//
// Internally protoMessage tracks the message state on the `fields` map. Values
// are stored as starlark.Value through execution and only converted into a
// proto.Message through AsProtoMessage. Any fields set to starlark.None or the
// default field value will be ignored when returning to a protobuf.Message
type protoMessage struct {
	// A copy of the underlying is stored so AsProtoMessage can construct a new object
	msg     proto.Message
	msgDesc protoreflect.MessageDescriptor
	fields  map[string]starlark.Value
	frozen  bool
}

var _ starlark.Value = (*protoMessage)(nil)
var _ starlark.HasAttrs = (*protoMessage)(nil)
var _ starlark.HasSetField = (*protoMessage)(nil)
var _ starlark.Comparable = (*protoMessage)(nil)

func (msg *protoMessage) String() string {
	return fmt.Sprintf("<%s %s>", msg.Type(), (prototext.MarshalOptions{Multiline: false}).Format(msg.toProtoMessage()))
}
func (msg *protoMessage) Type() string         { return string(msg.msgDesc.FullName()) }
func (msg *protoMessage) Truth() starlark.Bool { return starlark.True }
func (msg *protoMessage) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", msg.Type())
}

func (msg *protoMessage) Freeze() {
	if !msg.frozen {
		msg.frozen = true
		for _, field := range msg.fields {
			field.Freeze()
		}
	}
}

// Clear all fields to unset
func (msg *protoMessage) Clear() error {
	if err := msg.CheckMutable("clear"); err != nil {
		return err
	}

	msg.fields = make(map[string]starlark.Value)

	return nil
}

func (msg *protoMessage) CheckMutable(verb string) error {
	if msg.frozen {
		return fmt.Errorf("cannot %s frozen message", verb)
	}
	return nil
}

func (msg *protoMessage) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	other, ok := y.(*protoMessage)
	if !ok {
		return false, nil
	}

	switch op {
	case syntax.EQL:
		eql := proto.Equal(msg.toProtoMessage(), other.toProtoMessage())
		return eql, nil
	case syntax.NEQ:
		eql := proto.Equal(msg.toProtoMessage(), other.toProtoMessage())
		return !eql, nil
	default:
		return false, fmt.Errorf("Only == and != operations are supported on protobufs, found %s", op.String())
	}
}

func (msg *protoMessage) MarshalJSON() ([]byte, error) {
	jsonData, err := (protojson.MarshalOptions{
		UseProtoNames: true,
	}).Marshal(msg.toProtoMessage())
	if err != nil {
		return nil, err
	}
	return []byte(jsonData), nil
}

func (msg *protoMessage) Attr(name string) (starlark.Value, error) {
	// If a value has already been set on msg, return it
	if val, ok := msg.fields[name]; ok {
		return val, nil
	}

	fieldDesc := getFieldDescriptor(msg.msgDesc, name)
	if fieldDesc == nil {
		return starlark.None, fmt.Errorf("AttributeError: `%s' value has no field %q", msg.Type(), name)
	}

	// Given field name exists but has not been set, return a default value
	val := fieldDesc.Default()

	starlarkValue, err := valueToStarlark(val, fieldDesc)
	if err != nil {
		return starlark.None, err
	}

	// For non-scalar values, set the value on access even if it is unset so
	// use without initialization works.
	//
	// Example:
	//   msg = MyProtoMessage()
	//   msg.repeated_field.append("a")
	//   # msg.repeated_field should be ["a"]
	if fieldDesc.IsList() || fieldDesc.IsMap() || fieldDesc.Kind() == protoreflect.MessageKind {
		msg.SetField(name, starlarkValue)
	}

	return starlarkValue, nil
}

func (msg *protoMessage) AttrNames() []string {
	return fieldNames(msg.msgDesc)
}

func fieldNames(msgDesc protoreflect.MessageDescriptor) []string {
	out := []string{}
	fields := msgDesc.Fields()
	for i := 0; i < fields.Len(); i++ {
		fieldName := string(fields.Get(i).Name())
		out = append(out, fieldName)
	}
	sort.Strings(out)
	return out
}

func (msg *protoMessage) SetField(name string, val starlark.Value) error {
	fieldDesc := getFieldDescriptor(msg.msgDesc, name)
	if fieldDesc == nil {
		return fmt.Errorf("AttributeError: `%s' value has no field %q", msg.Type(), name)
	}

	if err := msg.CheckMutable("set field of"); err != nil {
		return err
	}

	// Autoconvert starlark.List, starlark.Dict, wrapperspb on assignment
	if fieldDesc.IsList() {
		if starlarkListVal, ok := val.(*starlark.List); ok {
			// To support repeated StringValue support autoboxing
			// if relevant conversion, mutate incoming list
			if fieldDesc.Kind() == protoreflect.MessageKind {
				for i := 0; i < starlarkListVal.Len(); i++ {
					msg, err := maybeConvertToWrapper(fieldDesc, starlarkListVal.Index(i))
					if err != nil {
						return err
					}
					if msg != nil {
						starlarkListVal.SetIndex(i, msg)
					}
				}
			}

			// Convert starlark.List to protoRepeated
			list, err := newProtoRepeatedFromList(fieldDesc, starlarkListVal)
			if err != nil {
				return err
			}

			val = list
		}
	} else if fieldDesc.IsMap() {
		if starlarkDictVal, ok := val.(*starlark.Dict); ok {
			// Convert stalark.Map into protoMap
			mapVal, err := newProtoMapFromDict(fieldDesc.MapKey(), fieldDesc.MapValue(), starlarkDictVal)
			if err != nil {
				return err
			}

			val = mapVal
		}
	} else if fieldDesc.Kind() == protoreflect.MessageKind {
		msg, err := maybeConvertToWrapper(fieldDesc, val)
		if err != nil {
			return err
		}
		if msg != nil {
			val = msg
		}
	}

	// Allow using msg_field = None to unset a scalar message field
	if fieldAllowsNone(fieldDesc) && val == starlark.None {
		delete(msg.fields, name)
		return nil
	}

	// If valueFromStarlark returns an error, the val cannot be assigned to the field
	_, err := valueFromStarlark(msg.msg.ProtoReflect(), fieldDesc, val)
	if err != nil {
		return err
	}

	// Clear other oneof
	oneof := fieldDesc.ContainingOneof()
	if oneof != nil {
		fields := oneof.Fields()
		for i := 0; i < fields.Len(); i++ {
			delete(msg.fields, string(fields.Get(i).Name()))
		}
	}

	msg.fields[name] = val

	return nil
}

// Applies default values for proto2 fields with defaults that are not already set
func (msg *protoMessage) SetDefaults() error {
	if err := msg.CheckMutable("set field defaults of"); err != nil {
		return err
	}

	for _, fieldName := range fieldNames(msg.msgDesc) {
		fieldDesc := getFieldDescriptor(msg.msgDesc, fieldName)
		if fieldDesc == nil {
			continue
		}

		// Already set, do not set default
		if _, ok := msg.fields[fieldName]; ok {
			continue
		}

		if !fieldDesc.HasDefault() {
			continue
		}

		val, err := valueToStarlark(fieldDesc.Default(), fieldDesc)
		if err != nil {
			return err
		}

		msg.SetField(fieldName, val)
	}

	return nil
}

// Merges values from other into msg following proto.Merge logic
func (msg *protoMessage) Merge(other *protoMessage) error {
	if msg.Type() != other.Type() {
		return fmt.Errorf("Cannot merge protobufs of different types: Merge(%s, %s) ", msg.Type(), other.Type())
	}

	if err := msg.CheckMutable("merge"); err != nil {
		return err
	}

	for fieldName, val := range other.fields {
		fieldDesc := getFieldDescriptor(msg.msgDesc, fieldName)
		if fieldDesc == nil {
			continue
		}

		merged, err := mergeField(msg.fields[fieldName], val)
		if err != nil {
			return err
		}

		msg.SetField(fieldName, merged)
	}

	return nil
}

// Construct a new instance of msg.msg and set each field on msg.fields
func (msg *protoMessage) toProtoMessage() proto.Message {
	out := proto.Clone(msg.msg)
	proto.Reset(out)

	// All entries in msg.fields should exist as fields on the message, and
	// the values be the corresponding type, checked in SetField
	for fieldName, val := range msg.fields {
		fieldDesc := getFieldDescriptor(msg.msgDesc, fieldName)
		if fieldDesc == nil {
			continue
		}

		protoValue, err := valueFromStarlark(msg.msg.ProtoReflect(), fieldDesc, val)
		if err != nil {
			continue
		}

		out.ProtoReflect().Set(fieldDesc, protoValue)
	}

	return out
}

func getFieldDescriptor(msgDesc protoreflect.MessageDescriptor, fieldName string) protoreflect.FieldDescriptor {
	fields := msgDesc.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if fieldName == string(field.Name()) {
			return field
		}
	}

	return nil
}

// Return if a value is not the default
func isFieldSet(v protoreflect.Value, fieldDesc protoreflect.FieldDescriptor) bool {
	switch fieldDesc.Kind() {
	case protoreflect.BytesKind:
		return len(v.Bytes()) != 0
	default:
		return v.Interface() != fieldDesc.Default().Interface()
	}
}

func fieldAllowsNone(fieldDesc protoreflect.FieldDescriptor) bool {
	return (fieldDesc.Kind() == protoreflect.MessageKind && !fieldDesc.IsList() && !fieldDesc.IsMap()) || fieldDesc.Syntax() == protoreflect.Proto2
}
