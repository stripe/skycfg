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
	"errors"
	"strings"
	"testing"

	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	any "google.golang.org/protobuf/types/known/anypb"

	pb "github.com/stripe/skycfg/internal/testdata/test_proto"
)

func init() {
	resolve.AllowFloat = true
}

func newRegistry() *protoregistry.Types {
	registry := &protoregistry.Types{}
	registry.RegisterMessage((&pb.MessageV2{}).ProtoReflect().Type())
	registry.RegisterMessage((&pb.MessageV2_NestedMessage{}).ProtoReflect().Type())
	registry.RegisterMessage((&pb.MessageV3{}).ProtoReflect().Type())
	registry.RegisterMessage((&pb.MessageV3_NestedMessage{}).ProtoReflect().Type())
	registry.RegisterEnum((pb.ToplevelEnumV2)(0).Type())
	registry.RegisterEnum((pb.ToplevelEnumV3)(0).Type())
	return registry
}

func TestProtoPackage(t *testing.T) {
	globals := starlark.StringDict{
		//"proto": NewModule(newRegistry()),
		"proto": &starlarkstruct.Module{
			Name: "proto",
			Members: starlark.StringDict{
				"package": starlarkPackageFn(newRegistry()),
			},
		},
	}

	runSkycfgTests(t, []skycfgTest{
		{
			src:  `proto.package("skycfg.test_proto")`,
			want: `<proto.Package "skycfg.test_proto">`,
		},
		{
			src:  `dir(proto.package("skycfg.test_proto"))`,
			want: `["MessageV2", "MessageV3", "ToplevelEnumV2", "ToplevelEnumV3"]`,
		},
		{
			src:  `proto.package("skycfg.test_proto").MessageV2`,
			want: `<proto.MessageType "skycfg.test_proto.MessageV2">`,
		},
		{
			src:  `proto.package("skycfg.test_proto").ToplevelEnumV2`,
			want: `<proto.EnumType "skycfg.test_proto.ToplevelEnumV2">`,
		},
		{
			src:     `proto.package("skycfg.test_proto").NoExist`,
			wantErr: errors.New(`Protobuf type "skycfg.test_proto.NoExist" not found`),
		},
	}, withGlobals(globals))
}

func TestMessageType(t *testing.T) {
	globals := starlark.StringDict{
		"pb": NewProtoPackage(newRegistry(), "skycfg.test_proto"),
	}

	runSkycfgTests(t, []skycfgTest{
		{
			src:  `pb.MessageV2`,
			want: `<proto.MessageType "skycfg.test_proto.MessageV2">`,
		},
		{
			src:  `dir(pb.MessageV2)`,
			want: `["NestedEnum", "NestedMessage"]`,
		},
		{
			src:  `pb.MessageV2.NestedMessage`,
			want: `<proto.MessageType "skycfg.test_proto.MessageV2.NestedMessage">`,
		},
		{
			src:  `pb.MessageV2.NestedEnum`,
			want: `<proto.EnumType "skycfg.test_proto.MessageV2.NestedEnum">`,
		},
		{
			src:     `pb.MessageV2.NoExist`,
			wantErr: errors.New(`Protobuf type "skycfg.test_proto.MessageV2.NoExist" not found`),
		},
	}, withGlobals(globals))
}

func TestEnumType(t *testing.T) {
	globals := starlark.StringDict{
		"pb": NewProtoPackage(newRegistry(), "skycfg.test_proto"),
	}

	runSkycfgTests(t, []skycfgTest{
		{
			src:  `pb.ToplevelEnumV2`,
			want: `<proto.EnumType "skycfg.test_proto.ToplevelEnumV2">`,
		},
		{
			src:  `dir(pb.ToplevelEnumV2)`,
			want: `["TOPLEVEL_ENUM_V2_A", "TOPLEVEL_ENUM_V2_B"]`,
		},
		{
			src:  `pb.MessageV2.NestedEnum`,
			want: `<proto.EnumType "skycfg.test_proto.MessageV2.NestedEnum">`,
		},
		{
			src:  `dir(pb.MessageV2.NestedEnum)`,
			want: `["NESTED_ENUM_A", "NESTED_ENUM_B"]`,
		},
		{
			src:     `pb.ToplevelEnumV2.NoExist`,
			wantErr: errors.New(`proto.EnumType has no .NoExist field or method`),
		},
	}, withGlobals(globals))
}

func TestListType(t *testing.T) {
	var listFieldDesc protoreflect.FieldDescriptor
	msg := (&pb.MessageV3{}).ProtoReflect().Descriptor()
	listFieldDesc = msg.Fields().ByName("r_string")

	globals := starlark.StringDict{
		"list": starlark.NewBuiltin("list", func(
			t *starlark.Thread,
			fn *starlark.Builtin,
			args starlark.Tuple,
			kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			return newProtoRepeated(listFieldDesc), nil
		}),
	}

	runSkycfgTests(t, []skycfgTest{
		{
			name: "new list",
			src:  `list()`,
			want: `[]`,
		},
		{
			name: "list AttrNames",
			src:  `dir(list())`,
			want: `["append", "clear", "extend", "index", "insert", "pop", "remove"]`,
		},
		// List methods
		{
			name: "list.Append",
			srcFunc: `
def fun():
    l = list()
    l.append("some string")
    return l
`,
			want: `["some string"]`,
		},
		{
			name: "list.Extend",
			srcFunc: `
def fun():
    l = list()
    l.extend(["a", "b"])
    return l
`,
			want: `["a", "b"]`,
		},
		{
			name: "list.Clear",
			srcFunc: `
def fun():
    l = list()
    l.extend(["a", "b"])
    l.clear()
    return l
`,
			want: `[]`,
		},
		{
			name: "list.SetIndex",
			srcFunc: `
def fun():
    l = list()
    l.extend(["a", "b"])
    l[1] = "c"
    return l
`,
			want: `["a", "c"]`,
		},
		{
			name: "list binary add operation",
			srcFunc: `
def fun():
    l = list()
    l2 = list()
    l2.extend(["a", "b"])
    l += l2
    l += ["c", "d"]
    return l
`,
			want: `["a", "b", "c", "d"]`,
		},

		// List typechecking
		{
			name:    "list append typchecks",
			src:     `list().append(1)`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "list extend typchecks",
			src:     `list().extend([1,2])`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name: "list set index typchecks",
			srcFunc: `
def fun():
    l = list()
    l.extend(["a", "b"])
    l[1] = 1
    return l
`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
	}, withGlobals(globals))
}

func TestMapType(t *testing.T) {
	var mapFieldDesc protoreflect.FieldDescriptor
	msg := (&pb.MessageV3{}).ProtoReflect().Descriptor()
	mapFieldDesc = msg.Fields().ByName("map_string")

	globals := starlark.StringDict{
		"map": starlark.NewBuiltin("map", func(
			t *starlark.Thread,
			fn *starlark.Builtin,
			args starlark.Tuple,
			kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			return newProtoMap(mapFieldDesc.MapKey(), mapFieldDesc.MapValue()), nil
		}),
	}

	runSkycfgTests(t, []skycfgTest{
		{
			name: "new map",
			src:  `map()`,
			want: `{}`,
		},
		{
			name: "map AttrNames",
			src:  `dir(map())`,
			want: `["clear", "get", "items", "keys", "pop", "popitem", "setdefault", "update", "values"]`,
		},
		// Map methods
		{
			name: "map.SetDefault",
			srcFunc: `
def fun():
    m = map()
    m["a"] = "A"
    m.setdefault('a', 'Z')
    m.setdefault('b', 'Z')
    return m
`,
			want: `{"a": "A", "b": "Z"}`,
		},
		{
			name: "map.SetKey",
			srcFunc: `
def fun():
    m = map()
    m["a"] = "some string"
    return m
`,
			want: `{"a": "some string"}`,
		},
		{
			name: "map.Update",
			srcFunc: `
def fun():
    m = map()
    m.update([("a", "a_string"), ("b", "b_string")])
    return m
`,
			want: `{"a": "a_string", "b": "b_string"}`,
		},
		{
			name: "map.Clear",
			srcFunc: `
def fun():
    m = map()
    m["a"] = "some string"
    m.clear()
    return m
`,
			want: `{}`,
		},

		// Map typechecking
		{
			name: "map.SetKey typechecks",
			srcFunc: `
def fun():
    m = map()
    m["a"] = 1
    return m
`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "map.Update typechecks",
			src:     `map().update([("a", 1)])`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "map.SetDefault typechecks",
			src:     `map().setdefault("a", 1)`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "map.SetDefault typechecks key",
			src:     `map().setdefault(1, "a")`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
	}, withGlobals(globals))
}

func TestRepeatedProtoFieldMutation(t *testing.T) {
	runSkycfgTests(t, []skycfgTest{
		{
			srcFunc: `
def fun():
    pkg = proto.package("skycfg.test_proto")
    msg = pkg.MessageV3()
    msg.r_submsg.append(pkg.MessageV3())
    msg.r_submsg[0].f_string = "foo"
    msg.r_submsg.extend([pkg.MessageV3()])
    msg.r_submsg[1].f_string = "bar"
    return msg`,
			want:              `<skycfg.test_proto.MessageV3 r_submsg:{f_string:"foo"} r_submsg:{f_string:"bar"}>`,
			removeRandomSpace: true,
		},
	})
}

// Skycfg has had inconsistent copy on assignment behavior
// Test that Skycfg does not copy lists/maps on assignment, matching Starlark/Python's behavior
func TestNoCopyOnAssignment(t *testing.T) {
	runSkycfgTests(t, []skycfgTest{
		{
			name: "list does not copy on assignment, *protoRepeated",
			srcFunc: `
def fun():
    pkg = proto.package("skycfg.test_proto")
    msg1 = pkg.MessageV3()
    msg2 = pkg.MessageV3()
    msg1.r_string = ["a","b"]
    a = msg1.r_string
    msg2.r_string = msg1.r_string
    a.append("c")
    return [msg1.r_string, msg2.r_string, a]
`,
			want: `[["a", "b", "c"], ["a", "b", "c"], ["a", "b", "c"]]`,
		},
		{
			name: "list does not copy on assignment, *stalark.List",
			srcFunc: `
def fun():
    pkg = proto.package("skycfg.test_proto")
    a = ["a","b"]
    msg1 = pkg.MessageV3()
    msg1.r_string = a
    a.append("c")
    msg1.r_string.append("d")
    return [msg1.r_string, a]
`,
			want: `[["a", "b", "c", "d"], ["a", "b", "c", "d"]]`,
		},
		{
			name: "map does not copy on assignment, *protoMap",
			srcFunc: `
def fun():
    pkg = proto.package("skycfg.test_proto")
    msg1 = pkg.MessageV3()
    msg2 = pkg.MessageV3()
    msg1.map_string = {
        "ka": "va",
        "kb": "vb",
    }
    a = msg1.map_string
    msg2.map_string = msg1.map_string
    a["kc"] = "vc"
    return [msg1.map_string, msg2.map_string, a]
`,
			want: `[{"ka": "va", "kb": "vb", "kc": "vc"}, {"ka": "va", "kb": "vb", "kc": "vc"}, {"ka": "va", "kb": "vb", "kc": "vc"}]`,
		},
		{
			name: "map does not copy on assignment, *stalark.Dict",
			srcFunc: `
def fun():
    pkg = proto.package("skycfg.test_proto")
    msg1 = pkg.MessageV3()
    a = {
        "ka": "va",
        "kb": "vb",
    }
    msg1.map_string = a
    a["kc"] = "vc"
    msg1.map_string["kd"] = "vd"
    return [msg1.map_string, a]
`,
			want: `[{"ka": "va", "kb": "vb", "kc": "vc", "kd": "vd"}, {"ka": "va", "kb": "vb", "kc": "vc", "kd": "vd"}]`,
		},
		{
			name: "message does not copy on assignment",
			srcFunc: `
def fun():
    pkg = proto.package("skycfg.test_proto")
    msg1 = pkg.MessageV3()
    msg2 = pkg.MessageV3()
    msg1.f_submsg = pkg.MessageV3()
    msg2.f_submsg = msg1.f_submsg
    a = msg1.f_submsg
    a.f_string = "a"
    return [msg1.f_submsg.f_string, msg2.f_submsg.f_string, a.f_string]
`,
			want: `["a", "a", "a"]`,
		},
	})
}

func TestProtoEnumEqual(t *testing.T) {
	runSkycfgTests(t, []skycfgTest{
		{
			src:  `proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A == proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A`,
			want: true,
		},
		{
			src:  `proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A == proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B`,
			want: false,
		},
		{
			src:  `proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A != proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A`,
			want: false,
		},
		{
			src:  `proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A != proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B`,
			want: true,
		},
	})
}

func TestProtoText(t *testing.T) {
	runSkycfgTests(t, []skycfgTest{
		{
			name: "proto.encode_text",
			src: `
proto.encode_text(proto.package("skycfg.test_proto").MessageV3(
    f_string = "some string",
))`,
			want:     `"f_string:\"some string\""`,
			wantType: "string",
		},
		{
			name: "proto.encode_text compact",
			src: `
proto.encode_text(proto.package("skycfg.test_proto").MessageV3(
    f_string = "some string",
), compact=True)`,
			want:     `"f_string:\"some string\""`,
			wantType: "string",
		},
		{
			name: "proto.encode_text full",
			src: `
proto.encode_text(proto.package("skycfg.test_proto").MessageV3(
    f_string = "some string",
), compact=False)`,
			want:              `"f_string: \"some string\"\n"`,
			wantType:          "string",
			removeRandomSpace: true,
		},
		{
			name: "proto.decode_text",
			src:  `proto.decode_text(proto.package("skycfg.test_proto").MessageV3, "f_int32: 1010").f_int32`,
			want: "1010",
		},
	})
}

func TestProtoJson(t *testing.T) {
	runSkycfgTests(t, []skycfgTest{
		{
			name: "proto.encode_json",
			src: `proto.encode_json(proto.package("skycfg.test_proto").MessageV3(
				f_string = "some string",
			))`,
			want:     `"{\"f_string\":\"some string\"}"`,
			wantType: "string",
		},
		{
			name: "proto.encode_json compact",
			src: `proto.encode_json(proto.package("skycfg.test_proto").MessageV3(
				f_string = "some string",
			), compact=True)`,
			want:     `"{\"f_string\":\"some string\"}"`,
			wantType: "string",
		},
		{
			name: "proto.encode_json full",
			src: `proto.encode_json(proto.package("skycfg.test_proto").MessageV3(
				f_string = "some string",
			), compact=False)`,
			want:              `"{\n \"f_string\": \"some string\"\n}"`,
			wantType:          "string",
			removeRandomSpace: true,
		},
		{
			name: "proto.decode_json",
			src:  `proto.decode_json(proto.package("skycfg.test_proto").MessageV3, "{\"f_int32\": 1010}").f_int32`,
			want: "1010",
		},
		{
			// This is a bit of a weird test. Protobuf's have complex behavior around whether a field is present.
			// Reference: https://github.com/protocolbuffers/protobuf/blob/main/docs/field_presence.md
			// This is particularly relevant for JSON encoding and decoding.
			// This test specifically checks two things:
			//   1. skycfg does not track presence for default scalar values (i.e., the empty field may be "not present" after a serialization round-trip)
			//   2. skycfg _does_ track presence for `oneof` fields
			// Without this behavior, working with "json-like" protos, e.g., google.protobuf.Value, becomes challenging.
			name: "proto.decode_json oneof field presence",
			src: `proto.encode_json(proto.decode_json(
				proto.package("skycfg.test_proto").MessageV3,
				"{\"f_string\":\"\",\"f_oneof_a\":\"\"}",
			))`,
			want: `"{\"f_oneof_a\":\"\"}"`,
		},
	})
}

func TestProtoToAnyV2(t *testing.T) {
	val, err := eval(`proto.encode_any(proto.package("skycfg.test_proto").MessageV2(
		f_string = "some string",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	myProto := mustProtoMessage(t, val)
	myAny := myProto.(*anypb.Any)

	want := "type.googleapis.com/skycfg.test_proto.MessageV2"
	if want != myAny.GetTypeUrl() {
		t.Fatalf("encode_any: wanted %q, got %q", want, myAny.GetTypeUrl())
	}

	msg := pb.MessageV2{}
	err = myAny.UnmarshalTo(&msg)
	if err != nil {
		t.Fatalf("encode_any: could not unmarshal: %v", err)
	}

	want = "some string"
	if want != msg.GetFString() {
		t.Fatalf("encode_any: wanted %q, got %q", want, msg.GetFString())
	}
}

func TestProtoToAnyV3(t *testing.T) {
	val, err := eval(`proto.encode_any(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	myAny := mustProtoMessage(t, val).(*any.Any)

	want := "type.googleapis.com/skycfg.test_proto.MessageV3"
	if want != myAny.GetTypeUrl() {
		t.Fatalf("encode_any: wanted %q, got %q", want, myAny.GetTypeUrl())
	}

	msg := pb.MessageV3{}
	err = myAny.UnmarshalTo(&msg)
	if err != nil {
		t.Fatalf("encode_any: could not unmarshal: %v", err)
	}

	want = "some string"
	if want != msg.GetFString() {
		t.Fatalf("encode_any: wanted %q, got %q", want, msg.GetFString())
	}
}

type skycfgTest struct {
	name     string
	src      string
	srcFunc  string
	wantErr  error
	want     interface{}
	wantType string

	// Options
	globals           starlark.StringDict
	removeRandomSpace bool
}

// Mutates all tests
type globalTestOption func(*skycfgTest)

func withGlobals(globals starlark.StringDict) globalTestOption {
	return func(test *skycfgTest) {
		test.globals = globals
	}
}

func runSkycfgTests(t *testing.T, tests []skycfgTest, opts ...globalTestOption) {
	t.Helper()
	for _, test := range tests {
		for _, opt := range opts {
			opt(&test)
		}

		name := test.name
		if name == "" {
			name = test.src
		}
		t.Run(name, func(t *testing.T) {
			var val starlark.Value
			var err error
			if test.src != "" {
				val, err = eval(test.src, test.globals)
			} else if test.srcFunc != "" {
				val, err = evalFunc(test.srcFunc, test.globals)
			} else {
				t.Fatal("Test has no src or srcFunc")
			}

			checkError(t, err, test.wantErr)

			// Only check values if evaluation is not expected to error
			if test.wantErr != nil {
				return
			}

			switch want := test.want.(type) {
			case proto.Message:
				got := mustProtoMessage(t, val)
				checkProtoEqual(t, want, got)
			case string:
				got := val.String()
				if test.removeRandomSpace {
					got = removeRandomSpace(got)
				}
				if want != got {
					t.Fatalf("wanted: %s\ngot   : %s", want, got)
				}
			case bool:
				got := val.(starlark.Bool)
				if bool(got) != want {
					t.Fatalf("wanted: %t\ngot   : %t", want, got)
				}
			default:
				t.Fatalf("runSkycfgTests does not support comparing %T yet", want)
			}

			if test.wantType != "" {
				if val.Type() != test.wantType {
					t.Fatalf("Expected type\nwanted: %t\ngot   : %t", test.wantType, val.Type())
				}
			}
		})
	}
}

func eval(src string, globals starlark.StringDict) (starlark.Value, error) {
	if globals == nil {
		globals = starlark.StringDict{
			"proto": NewModule(newRegistry()),
		}
	}

	return starlark.Eval(&starlark.Thread{}, "", src, globals)
}

func evalFunc(src string, globals starlark.StringDict) (starlark.Value, error) {
	if globals == nil {
		globals = starlark.StringDict{
			"proto": NewModule(newRegistry()),
		}
	}

	globals, err := starlark.ExecFile(&starlark.Thread{}, "", src, globals)
	if err != nil {
		return nil, err
	}
	v, ok := globals["fun"]
	if !ok {
		return nil, errors.New(`Expected function "fun", not found`)
	}
	fun, ok := v.(starlark.Callable)
	if !ok {
		return nil, errors.New("Fun not callable")
	}
	return starlark.Call(&starlark.Thread{}, fun, nil, nil)
}

func mustProtoMessage(t *testing.T, v starlark.Value) proto.Message {
	t.Helper()
	if msg, ok := AsProtoMessage(v); ok {
		return msg
	}
	t.Fatalf("Expected *protoMessage value, got %T", v)
	return nil
}

func mustMarshalAny(t *testing.T, v proto.Message) *anypb.Any {
	t.Helper()
	any, err := anypb.New(v)
	if err != nil {
		t.Fatalf("Expected *protoMessage value, got %T", v)
	}
	return any
}

func checkError(t *testing.T, got, want error) {
	t.Helper()

	if want == nil && got != nil {
		t.Fatalf("Expected no error, got: %v\n", got)
	} else if want != nil && got == nil {
		t.Fatalf("Expected error got nil\nwanted: %q", want.Error())
	} else if want != nil && got.Error() != want.Error() {
		t.Fatalf("Expected error\nwanted: %q\ngot   : %q", want.Error(), got.Error())
	}
}

// Generate a diff of two structs, which may contain protobuf messages.
func checkProtoEqual(t *testing.T, want, got proto.Message) {
	t.Helper()
	if proto.Equal(want, got) {
		return
	}

	t.Fatalf(
		"Protobuf messages differ\n--- WANT:\n%v\n--- GOT:\n%v\n",
		(prototext.MarshalOptions{Multiline: true}).Format(want),
		(prototext.MarshalOptions{Multiline: true}).Format(got),
	)
}

// https://github.com/protocolbuffers/protobuf-go/commit/c3f4d486298baf0f69057fc51d4b5194f8dfbfdd
func removeRandomSpace(s string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(s, "  ", " "),
		" >", ">",
	)
}
