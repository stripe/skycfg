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

	tests := []struct {
		expr    string
		want    string
		wantErr error
	}{
		{
			expr: `proto.package("skycfg.test_proto")`,
			want: `<proto.Package "skycfg.test_proto">`,
		},
		{
			expr: `dir(proto.package("skycfg.test_proto"))`,
			want: `["MessageV2", "MessageV3", "ToplevelEnumV2", "ToplevelEnumV3"]`,
		},
		{
			expr: `proto.package("skycfg.test_proto").MessageV2`,
			want: `<proto.MessageType "skycfg.test_proto.MessageV2">`,
		},
		{
			expr: `proto.package("skycfg.test_proto").ToplevelEnumV2`,
			want: `<proto.EnumType "skycfg.test_proto.ToplevelEnumV2">`,
		},
		{
			expr:    `proto.package("skycfg.test_proto").NoExist`,
			wantErr: errors.New(`Protobuf type "skycfg.test_proto.NoExist" not found`),
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			val, err := starlark.Eval(&starlark.Thread{}, "", test.expr, globals)

			if test.wantErr != nil {
				if !checkError(err, test.wantErr) {
					t.Fatalf("eval(%q): expected error %v, got %v", test.expr, test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("eval(%q): %v", test.expr, err)
			}
			if test.want != val.String() {
				t.Errorf("eval(%q): expected value %q, got %q", test.expr, test.want, val.String())
			}
		})
	}
}

func TestMessageType(t *testing.T) {
	globals := starlark.StringDict{
		"pb": NewProtoPackage(newRegistry(), "skycfg.test_proto"),
	}

	tests := []struct {
		expr    string
		want    string
		wantErr error
	}{
		{
			expr: `pb.MessageV2`,
			want: `<proto.MessageType "skycfg.test_proto.MessageV2">`,
		},
		{
			expr: `dir(pb.MessageV2)`,
			want: `["NestedEnum", "NestedMessage"]`,
		},
		{
			expr: `pb.MessageV2.NestedMessage`,
			want: `<proto.MessageType "skycfg.test_proto.MessageV2.NestedMessage">`,
		},
		{
			expr: `pb.MessageV2.NestedEnum`,
			want: `<proto.EnumType "skycfg.test_proto.MessageV2.NestedEnum">`,
		},
		{
			expr:    `pb.MessageV2.NoExist`,
			wantErr: errors.New(`Protobuf type "skycfg.test_proto.MessageV2.NoExist" not found`),
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			val, err := starlark.Eval(&starlark.Thread{}, "", test.expr, globals)

			if test.wantErr != nil {
				if !checkError(err, test.wantErr) {
					t.Fatalf("eval(%q): expected error %v, got %v", test.expr, test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("eval(%q): %v", test.expr, err)
			}
			if test.want != val.String() {
				t.Errorf("eval(%q): expected value %q, got %q", test.expr, test.want, val.String())
			}
		})
	}
}

func TestEnumType(t *testing.T) {
	globals := starlark.StringDict{
		"pb": NewProtoPackage(newRegistry(), "skycfg.test_proto"),
	}

	tests := []struct {
		expr    string
		want    string
		wantErr error
	}{
		{
			expr: `pb.ToplevelEnumV2`,
			want: `<proto.EnumType "skycfg.test_proto.ToplevelEnumV2">`,
		},
		{
			expr: `dir(pb.ToplevelEnumV2)`,
			want: `["TOPLEVEL_ENUM_V2_A", "TOPLEVEL_ENUM_V2_B"]`,
		},
		{
			expr: `pb.MessageV2.NestedEnum`,
			want: `<proto.EnumType "skycfg.test_proto.MessageV2.NestedEnum">`,
		},
		{
			expr: `dir(pb.MessageV2.NestedEnum)`,
			want: `["NESTED_ENUM_A", "NESTED_ENUM_B"]`,
		},
		{
			expr:    `pb.ToplevelEnumV2.NoExist`,
			wantErr: errors.New(`proto.EnumType has no .NoExist field or method`),
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			val, err := starlark.Eval(&starlark.Thread{}, "", test.expr, globals)

			if test.wantErr != nil {
				if !checkError(err, test.wantErr) {
					t.Fatalf("eval(%q): expected error %v, got %v", test.expr, test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("eval(%q): %v", test.expr, err)
			}
			if test.want != val.String() {
				t.Errorf("eval(%q): expected value %q, got %q", test.expr, test.want, val.String())
			}
		})
	}
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

	tests := []struct {
		name    string
		expr    string
		exprFun string
		want    string
		wantErr error
	}{
		{
			name: "new list",
			expr: `list()`,
			want: `[]`,
		},
		{
			name: "list AttrNames",
			expr: `dir(list())`,
			want: `["append", "clear", "extend", "index", "insert", "pop", "remove"]`,
		},
		// List methods
		{
			name: "list.Append",
			exprFun: `
def fun():
    l = list()
    l.append("some string")
    return l
`,
			want: `["some string"]`,
		},
		{
			name: "list.Extend",
			exprFun: `
def fun():
    l = list()
    l.extend(["a", "b"])
    return l
`,
			want: `["a", "b"]`,
		},
		{
			name: "list.Clear",
			exprFun: `
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
			exprFun: `
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
			exprFun: `
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
			expr:    `list().append(1)`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "list extend typchecks",
			expr:    `list().extend([1,2])`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name: "list set index typchecks",
			exprFun: `
def fun():
    l = list()
    l.extend(["a", "b"])
    l[1] = 1
    return l
`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var val starlark.Value
			var err error
			if test.expr != "" {
				val, err = starlark.Eval(&starlark.Thread{}, "", test.expr, globals)
			} else {
				val, err = evalFunc(test.exprFun, globals)
			}

			if test.wantErr != nil {
				if !checkError(err, test.wantErr) {
					t.Fatalf("eval(%q): expected error %v, got %v", test.expr, test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("eval(%q): %v", test.expr, err)
			}
			if test.want != val.String() {
				t.Errorf("eval(%q): expected value %q, got %q", test.expr, test.want, val.String())
			}
		})
	}
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

	tests := []struct {
		name    string
		expr    string
		exprFun string
		want    string
		wantErr error
	}{
		{
			name: "new map",
			expr: `map()`,
			want: `{}`,
		},
		{
			name: "map AttrNames",
			expr: `dir(map())`,
			want: `["clear", "get", "items", "keys", "pop", "popitem", "setdefault", "update", "values"]`,
		},
		// Map methods
		{
			name: "map.SetDefault",
			exprFun: `
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
			exprFun: `
def fun():
    m = map()
    m["a"] = "some string"
    return m
`,
			want: `{"a": "some string"}`,
		},
		{
			name: "map.Update",
			exprFun: `
def fun():
    m = map()
    m.update([("a", "a_string"), ("b", "b_string")])
    return m
`,
			want: `{"a": "a_string", "b": "b_string"}`,
		},
		{
			name: "map.Clear",
			exprFun: `
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
			exprFun: `
def fun():
    m = map()
    m["a"] = 1
    return m
`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "map.Update typechecks",
			expr:    `map().update([("a", 1)])`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "map.SetDefault typechecks",
			expr:    `map().setdefault("a", 1)`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
		{
			name:    "map.SetDefault typechecks key",
			expr:    `map().setdefault(1, "a")`,
			wantErr: errors.New(`TypeError: value 1 (type "int") can't be assigned to type "string".`),
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var val starlark.Value
			var err error
			if test.expr != "" {
				val, err = starlark.Eval(&starlark.Thread{}, "", test.expr, globals)
			} else {
				val, err = evalFunc(test.exprFun, globals)
			}

			if test.wantErr != nil {
				if !checkError(err, test.wantErr) {
					t.Fatalf("eval(%q): expected error %v, got %v", test.expr, test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("eval(%q): %v", test.expr, err)
			}
			if test.want != val.String() {
				t.Errorf("eval(%q): expected value %q, got %q", test.expr, test.want, val.String())
			}
		})
	}
}

func TestRepeatedProtoFieldMutation(t *testing.T) {
	val, err := evalFunc(`
def fun():
    pkg = proto.package("skycfg.test_proto")
    msg = pkg.MessageV3()
    msg.r_submsg.append(pkg.MessageV3())
    msg.r_submsg[0].f_string = "foo"
    msg.r_submsg.extend([pkg.MessageV3()])
    msg.r_submsg[1].f_string = "bar"
    return msg`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := removeRandomSpace(val.String())
	want := `<skycfg.test_proto.MessageV3 r_submsg:{f_string:"foo"} r_submsg:{f_string:"bar"}>`
	if want != got {
		t.Fatalf("skyProtoMessage.String(): wanted %q, got %q", want, got)
	}
}

func TestProtoEnumEqual(t *testing.T) {
	val, err := eval(`proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A == proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := val.(starlark.Bool)
	if !bool(got) {
		t.Error("Expected equal enums")
	}

	val, err = eval(`proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A == proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got = val.(starlark.Bool)
	if bool(got) {
		t.Error("Expected unequal enums")
	}

	val, err = eval(`proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A != proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got = val.(starlark.Bool)
	if bool(got) {
		t.Error("Expected equal enums")
	}

	val, err = eval(`proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_A != proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got = val.(starlark.Bool)
	if !bool(got) {
		t.Error("Expected unequal enums")
	}
}

func TestProtoToText(t *testing.T) {
	val, err := eval(`proto.encode_text(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := string(val.(starlark.String))
	want := "f_string:\"some string\""
	if want != got {
		t.Fatalf("encode_text: wanted %q, got %q", want, got)
	}
}

func TestProtoToTextCompact(t *testing.T) {
	val, err := eval(`proto.encode_text(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=True)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := string(val.(starlark.String))
	want := "f_string:\"some string\""
	if want != got {
		t.Fatalf("encode_text_compact: wanted %q, got %q", want, got)
	}
}

func TestProtoToTextFull(t *testing.T) {
	val, err := eval(`proto.encode_text(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=False)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := removeRandomSpace(string(val.(starlark.String)))
	want := "f_string: \"some string\"\n"
	if want != got {
		t.Fatalf("encode_text_full: wanted %q, got %q", want, got)
	}
}

func TestProtoFromText(t *testing.T) {
	val, err := eval(`proto.decode_text(proto.package("skycfg.test_proto").MessageV3, "f_int32: 1010").f_int32`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := val.String()
	want := "1010"
	if want != got {
		t.Fatalf("decode_text: wanted %q, got %q", want, got)
	}
}

func TestProtoToJson(t *testing.T) {
	val, err := eval(`proto.encode_json(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := string(val.(starlark.String))
	want := `{"f_string":"some string"}`
	if want != got {
		t.Fatalf("encode_json: wanted %q, got %q", want, got)
	}
}

func TestProtoToJsonCompact(t *testing.T) {
	val, err := eval(`proto.encode_json(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=True)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := string(val.(starlark.String))
	want := `{"f_string":"some string"}`
	if want != got {
		t.Fatalf("encode_json_compact: wanted %q, got %q", want, got)
	}
}

func TestProtoToJsonFull(t *testing.T) {
	val, err := eval(`proto.encode_json(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=False)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := removeRandomSpace(string(val.(starlark.String)))
	want := "{\n \"f_string\": \"some string\"\n}"
	if want != got {
		t.Fatalf("encode_json_full: wanted %q, got %q", want, got)
	}
}

func TestProtoFromJson(t *testing.T) {
	val, err := eval(`proto.decode_json(proto.package("skycfg.test_proto").MessageV3, "{\"f_int32\": 1010}").f_int32`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := val.String()
	want := "1010"
	if want != got {
		t.Fatalf("decode_json: wanted %q, got %q", want, got)
	}
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

func checkError(got, want error) bool {
	if got == nil {
		return false
	}
	return got.Error() == want.Error()
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
