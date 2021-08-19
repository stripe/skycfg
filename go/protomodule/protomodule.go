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

// Package protomodule defines a Starlark module of Protobuf-related functions.
package protomodule

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	any_pb "google.golang.org/protobuf/types/known/anypb"
)

// NewModule returns a Starlark module of Protobuf-related functions.
//
//  proto = module(
//    clear,
//    clone,
//    decode_any,
//    decode_json,
//    decode_text,
//    encode_any,
//    encode_json,
//    encode_text,
//    merge,
//    set_defaults,
//  )
//
// See `docs/modules.asciidoc` for details on the API of each function.
func NewModule(registry *protoregistry.Types) *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "proto",
		Members: starlark.StringDict{
			"clear":        starlarkClear,
			"clone":        starlarkClone,
			"decode_any":   decodeAny(registry),
			"decode_json":  decodeJSON(registry),
			"decode_text":  decodeText(registry),
			"encode_any":   starlarkEncodeAny,
			"encode_json":  encodeJSON(registry),
			"encode_text":  encodeText(registry),
			"merge":        starlarkMerge,
			"package":      starlarkPackageFn(registry),
			"set_defaults": starlarkSetDefaults,
		},
	}
}

var starlarkClear = starlark.NewBuiltin("proto.clear", func(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	_, skyProtoMsg, err := wantSingleProtoMessage(fn, args, kwargs)
	if err != nil {
		return nil, err
	}

	err = skyProtoMsg.Clear()
	if err != nil {
		return nil, err
	}

	return skyProtoMsg, nil
})

var starlarkClone = starlark.NewBuiltin("proto.clone", func(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	msg, _, err := wantSingleProtoMessage(fn, args, kwargs)
	if err != nil {
		return nil, err
	}
	return NewMessage(proto.Clone(msg))
})

func decodeAny(registry *protoregistry.Types) starlark.Callable {
	return starlark.NewBuiltin("proto.decode_any", func(
		t *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		protoMsg, skyProtoMsg, err := wantSingleProtoMessage(fn, args, kwargs)
		if err != nil {
			return nil, err
		}
		anyMsg, ok := protoMsg.(*any_pb.Any)
		if !ok {
			return nil, fmt.Errorf("%s: for parameter 1: got %s, want google.protobuf.Any", fn.Name(), skyProtoMsg.Type())
		}

		decoded, err := any_pb.UnmarshalNew(anyMsg, proto.UnmarshalOptions{
			Resolver: registry,
		})
		if err != nil {
			return nil, err
		}
		return NewMessage(decoded)
	})
}

func decodeJSON(registry *protoregistry.Types) starlark.Callable {
	return starlark.NewBuiltin("proto.decode_json", func(
		t *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		var msgType starlark.Value
		var value starlark.String
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &msgType, &value); err != nil {
			return nil, err
		}
		protoMsgType, ok := msgType.(skyProtoMessageType)
		if !ok {
			return nil, fmt.Errorf("%s: for parameter 1: got %s, want proto.MessageType", fn.Name(), msgType.Type())
		}

		unmarshal := protojson.UnmarshalOptions{
			Resolver: registry,
		}
		decoded := protoMsgType.NewMessage()
		if err := unmarshal.Unmarshal([]byte(value), decoded); err != nil {
			return nil, err
		}
		return NewMessage(decoded)
	})
}

func decodeText(registry *protoregistry.Types) starlark.Callable {
	return starlark.NewBuiltin("proto.decode_text", func(
		t *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		var msgType starlark.Value
		var value starlark.String
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &msgType, &value); err != nil {
			return nil, err
		}
		protoMsgType, ok := msgType.(skyProtoMessageType)
		if !ok {
			return nil, fmt.Errorf("%s: for parameter 1: got %s, want proto.MessageType", fn.Name(), msgType.Type())
		}

		unmarshal := prototext.UnmarshalOptions{
			Resolver: registry,
		}
		decoded := protoMsgType.NewMessage()
		if err := unmarshal.Unmarshal([]byte(value), decoded); err != nil {
			return nil, err
		}
		return NewMessage(decoded)
	})
}

var starlarkEncodeAny = starlark.NewBuiltin("proto.encode_any", func(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	protoMsg, _, err := wantSingleProtoMessage(fn, args, kwargs)
	if err != nil {
		return nil, err
	}
	any := &any_pb.Any{}
	if err := any_pb.MarshalFrom(any, protoMsg, proto.MarshalOptions{
		Deterministic: true,
	}); err != nil {
		return nil, err
	}

	return NewMessage(any)
})

func encodeJSON(registry *protoregistry.Types) starlark.Callable {
	return starlark.NewBuiltin("proto.encode_json", func(
		t *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		protoMsg, _, err := wantSingleProtoMessage(fn, args, nil)
		if err != nil {
			return nil, err
		}

		marshal := protojson.MarshalOptions{
			UseProtoNames: true,
			Resolver:      registry,
		}

		if len(kwargs) > 0 {
			compact := true
			if err := starlark.UnpackArgs(fn.Name(), nil, kwargs, "compact", &compact); err != nil {
				return nil, err
			}
			if !compact {
				marshal.Multiline = true
			}
		}
		jsonData, err := marshal.Marshal(protoMsg)
		if err != nil {
			return nil, err
		}
		return starlark.String(jsonData), nil
	})
}

func encodeText(registry *protoregistry.Types) starlark.Callable {
	return starlark.NewBuiltin("proto.encode_text", func(
		t *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		protoMsg, _, err := wantSingleProtoMessage(fn, args, nil)
		if err != nil {
			return nil, err
		}

		marshal := prototext.MarshalOptions{
			Resolver: registry,
		}

		if len(kwargs) > 0 {
			compact := true
			if err := starlark.UnpackArgs(fn.Name(), nil, kwargs, "compact", &compact); err != nil {
				return nil, err
			}
			if !compact {
				marshal.Multiline = true
			}
		}
		text, err := marshal.Marshal(protoMsg)
		if err != nil {
			return nil, err
		}
		return starlark.String(text), nil
	})
}

var starlarkMerge = starlark.NewBuiltin("proto.merge", func(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var val1, val2 starlark.Value
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &val1, &val2); err != nil {
		return nil, err
	}

	dst := val1.(*protoMessage)
	src := val2.(*protoMessage)
	if src.Type() != dst.Type() {
		return nil, fmt.Errorf("%s: types are not the same: got %s and %s", fn.Name(), src.Type(), dst.Type())
	}

	err := dst.Merge(src)
	if err != nil {
		return nil, err
	}

	return dst, nil
})

var starlarkSetDefaults = starlark.NewBuiltin("proto.set_defaults", func(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	_, skyProtoMsg, err := wantSingleProtoMessage(fn, args, kwargs)
	if err != nil {
		return nil, err
	}

	err = skyProtoMsg.SetDefaults()
	if err != nil {
		return nil, err
	}

	return skyProtoMsg, nil
})

type skyProtoMessageType interface {
	NewMessage() protoreflect.ProtoMessage
}

type skyProtoMessage interface {
	starlark.Value
	MarshalJSON() ([]byte, error)
	Clear() error
	Merge(*protoMessage) error
	SetDefaults() error
	CheckMutable(string) error
}

func wantSingleProtoMessage(
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (proto.Message, skyProtoMessage, error) {
	var val starlark.Value
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &val); err != nil {
		return nil, nil, err
	}
	gotMsg, ok := AsProtoMessage(val)
	if !ok {
		return nil, nil, fmt.Errorf("%s: for parameter 1: got %s, want proto.Message", fn.Name(), val.Type())
	}
	return gotMsg, val.(skyProtoMessage), nil
}
