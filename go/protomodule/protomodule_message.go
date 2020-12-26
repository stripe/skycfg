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

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// NewProtoMessage returns a Starlark value representing the given Protobuf
// message. It can be returned back to a proto.Message() via AsProtoMessage().
func NewMessage(msg proto.Message) starlark.Value {
	// TODO
	return nil
}

// AsProtoMessage returns a Protobuf message underlying the given Starlark
// value, which must have been created by NewProtoMessage(). Returns
// (_, false) if the value is not a valid message.
func AsProtoMessage(v starlark.Value) (proto.Message, bool) {
	// TODO
	return nil, false
}

type protoMessage struct {
	msg    protoreflect.Message
	frozen bool
}

var _ starlark.HasAttrs = (*protoMessage)(nil)
var _ starlark.HasSetField = (*protoMessage)(nil)
var _ starlark.Comparable = (*protoMessage)(nil)

func (msg *protoMessage) String() string {
	return fmt.Sprintf("<%s %s>", msg.Type(), prototext.Format(msg.msg.Interface()))
}
func (msg *protoMessage) Type() string         { return string(msg.msg.Descriptor().FullName()) }
func (msg *protoMessage) Truth() starlark.Bool { return starlark.True }
func (msg *protoMessage) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", msg.Type())
}

func (msg *protoMessage) Freeze() {
	/*
		if !msg.frozen {
			msg.frozen = true
			for _, attr := range msg.attrCache {
				attr.Freeze()
			}
		}
	*/
}

func (msg *protoMessage) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	/*
		other, ok := y.(*skyProtoMessage)
		if !ok {
			return false, nil
		}

		switch op {
		case syntax.EQL:
			eql := proto.Equal(msg.msg, other.msg)
			return eql, nil
		case syntax.NEQ:
			eql := proto.Equal(msg.msg, other.msg)
			return !eql, nil
		default:
			return false, fmt.Errorf("Only == and != operations are supported on protobufs, found %s", op.String())
		}
	*/
	return false, nil // TODO
}

func (msg *protoMessage) MarshalJSON() ([]byte, error) {
	/*
		if msg.looksLikeKubernetesGogo() {
			return json.Marshal(msg.msg)
		}

		jsonData, err := (protojson.MarshalOptions{
			UseProtoNames: true,
		}).Marshal(proto.MessageV2(msg.msg))
		if err != nil {
			return nil, err
		}
		return []byte(jsonData), nil
	*/
	return nil, nil // TODO
}

func (msg *protoMessage) Attr(name string) (starlark.Value, error) {
	return nil, nil // TODO
}

func (msg *protoMessage) AttrNames() []string {
	return nil // TODO
}

func (msg *protoMessage) SetField(name string, sky starlark.Value) error {
	return nil // TODO
}
