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

	proto_v1 "github.com/golang/protobuf/proto"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	any_pb "google.golang.org/protobuf/types/known/anypb"

	impl "github.com/stripe/skycfg/internal/go/skycfg"
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
	protoMsg, skyProtoMsg, err := wantSingleProtoMessage(fn, args, kwargs)
	if err != nil {
		return nil, err
	}
	if err := skyProtoMsg.CheckMutable("clear"); err != nil {
		return nil, err
	}
	proto.Reset(protoMsg)
	skyProtoMsg.ResetAttrCache()
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
	return impl.NewSkyProtoMessage(proto_v1.MessageV1(proto.Clone(msg))), nil
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
		return impl.NewSkyProtoMessage(proto_v1.MessageV1(decoded)), nil
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
		return impl.NewSkyProtoMessage(proto_v1.MessageV1(decoded)), nil
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
		return impl.NewSkyProtoMessage(proto_v1.MessageV1(decoded)), nil
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
	return impl.NewSkyProtoMessage(proto_v1.MessageV1(any)), nil
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
	dstMsg, ok := impl.ToProtoMessage(val1)
	if !ok {
		return nil, fmt.Errorf("%s: for parameter 1: got %s, want proto.Message", fn.Name(), val1.Type())
	}
	srcMsg, ok := impl.ToProtoMessage(val2)
	if !ok {
		return nil, fmt.Errorf("%s: for parameter 2: got %s, want proto.Message", fn.Name(), val2.Type())
	}
	dst := val1.(skyProtoMessage)
	src := val2.(skyProtoMessage)
	if src.Type() != dst.Type() {
		return nil, fmt.Errorf("%s: types are not the same: got %s and %s", fn.Name(), src.Type(), dst.Type())
	}
	if err := dst.CheckMutable("merge into"); err != nil {
		return nil, err
	}
	proto.Merge(proto_v1.MessageV2(dstMsg), proto_v1.MessageV2(srcMsg))
	dst.ResetAttrCache()
	return dst, nil
})

var starlarkSetDefaults = starlark.NewBuiltin("proto.set_defaults", func(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	protoMsg, skyProtoMsg, err := wantSingleProtoMessage(fn, args, kwargs)
	if err != nil {
		return nil, err
	}
	if err := skyProtoMsg.CheckMutable("set field defaults of"); err != nil {
		return nil, err
	}
	proto_v1.SetDefaults(proto_v1.MessageV1(protoMsg))
	skyProtoMsg.ResetAttrCache()
	return skyProtoMsg, nil
})

type skyProtoMessageType interface {
	NewMessage() proto.Message
}

type skyProtoMessage interface {
	starlark.Value
	MarshalJSON() ([]byte, error)
	ResetAttrCache()
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
	gotMsg, ok := impl.ToProtoMessage(val)
	if !ok {
		return nil, nil, fmt.Errorf("%s: for parameter 1: got %s, want proto.Message", fn.Name(), val.Type())
	}
	return proto_v1.MessageV2(gotMsg), val.(skyProtoMessage), nil
}
