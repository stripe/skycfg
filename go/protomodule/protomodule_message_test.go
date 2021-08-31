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
	"math"
	"reflect"
	"sort"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/proto"

	pb "github.com/stripe/skycfg/internal/testdata/test_proto"
)

func TestMessageAttrNames(t *testing.T) {
	val, err := eval(`proto.package("skycfg.test_proto").MessageV3()`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := val.(starlark.HasAttrs).AttrNames()
	want := []string{
		"f_int32",
		"f_int64",
		"f_uint32",
		"f_uint64",
		"f_float32",
		"f_float64",
		"f_string",
		"f_bool",
		"f_submsg",
		"r_string",
		"r_submsg",
		"map_string",
		"map_submsg",
		"f_nested_submsg",
		"f_toplevel_enum",
		"f_nested_enum",
		"f_oneof_a",
		"f_oneof_b",
		"f_bytes",
		"f_BoolValue",
		"f_StringValue",
		"f_DoubleValue",
		"f_Int32Value",
		"f_Int64Value",
		"f_BytesValue",
		"f_Uint32Value",
		"f_Uint64Value",
		"r_StringValue",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("protoMessage.AttrNames: wanted %#v, got %#v", want, got)
	}
}

func TestMessageV2(t *testing.T) {
	val, err := eval(`proto.package("skycfg.test_proto").MessageV2(
		f_int32 = 1010,
		f_int64 = 1020,
		f_uint32 = 1030,
		f_uint64 = 1040,
		f_float32 = 10.50,
		f_float64 = 10.60,
		f_string = "some string",
		f_bool = True,
		f_submsg = proto.package("skycfg.test_proto").MessageV2(
			f_string = "string in submsg",
		),
		r_string = ["r_string1", "r_string2"],
		r_submsg = [
			proto.package("skycfg.test_proto").MessageV2(
				f_string = "string in r_submsg",
			),
		],
		map_string = {
			"map_string key": "map_string val",
		},
		map_submsg = {
			"map_submsg key": proto.package("skycfg.test_proto").MessageV2(
				f_string = "map_submsg val",
			),
		},
		f_nested_submsg = proto.package("skycfg.test_proto").MessageV2.NestedMessage(
			f_string = "nested_submsg val",
		),
		f_toplevel_enum = proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B,
		f_nested_enum = proto.package("skycfg.test_proto").MessageV2.NestedEnum.NESTED_ENUM_B,
		f_oneof_a = "string in oneof",
		f_bytes = "also some string",
	)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotMsg := mustProtoMessage(t, val)
	wantMsg := &pb.MessageV2{
		FInt32:   proto.Int32(1010),
		FInt64:   proto.Int64(1020),
		FUint32:  proto.Uint32(1030),
		FUint64:  proto.Uint64(1040),
		FFloat32: proto.Float32(10.50),
		FFloat64: proto.Float64(10.60),
		FString:  proto.String("some string"),
		FBool:    proto.Bool(true),
		FSubmsg: &pb.MessageV2{
			FString: proto.String("string in submsg"),
		},
		RString: []string{"r_string1", "r_string2"},
		RSubmsg: []*pb.MessageV2{{
			FString: proto.String("string in r_submsg"),
		}},
		MapString: map[string]string{
			"map_string key": "map_string val",
		},
		MapSubmsg: map[string]*pb.MessageV2{
			"map_submsg key": &pb.MessageV2{
				FString: proto.String("map_submsg val"),
			},
		},
		FNestedSubmsg: &pb.MessageV2_NestedMessage{
			FString: proto.String("nested_submsg val"),
		},
		FToplevelEnum: pb.ToplevelEnumV2_TOPLEVEL_ENUM_V2_B.Enum(),
		FNestedEnum:   pb.MessageV2_NESTED_ENUM_B.Enum(),
		FOneof:        &pb.MessageV2_FOneofA{"string in oneof"},
		FBytes:        []byte("also some string"),
	}
	checkProtoEqual(t, wantMsg, gotMsg)

	wantAttrs := map[string]string{
		"f_int32":         "1010",
		"f_int64":         "1020",
		"f_uint32":        "1030",
		"f_uint64":        "1040",
		"f_float32":       "10.5",
		"f_float64":       "10.6",
		"f_string":        `"some string"`,
		"f_bool":          "True",
		"f_submsg":        `<skycfg.test_proto.MessageV2 f_string:"string in submsg" >`,
		"r_string":        `["r_string1", "r_string2"]`,
		"r_submsg":        `[<skycfg.test_proto.MessageV2 f_string:"string in r_submsg" >]`,
		"map_string":      `{"map_string key": "map_string val"}`,
		"map_submsg":      `{"map_submsg key": <skycfg.test_proto.MessageV2 f_string:"map_submsg val" >}`,
		"f_nested_submsg": `<skycfg.test_proto.MessageV2.NestedMessage f_string:"nested_submsg val" >`,
		"f_toplevel_enum": `<skycfg.test_proto.ToplevelEnumV2 TOPLEVEL_ENUM_V2_B=1>`,
		"f_nested_enum":   `<skycfg.test_proto.MessageV2.NestedEnum NESTED_ENUM_B=1>`,
		"f_oneof_a":       `"string in oneof"`,
		"f_oneof_b":       `""`,
		"f_bytes":         `"also some string"`,
	}
	attrs := val.(starlark.HasAttrs)
	for attrName, wantAttr := range wantAttrs {
		wantAttr = removeRandomSpace(wantAttr)
		attr, err := attrs.Attr(attrName)
		if err != nil {
			t.Fatalf("val.Attr(%q): %v", attrName, err)
		}
		gotAttr := removeRandomSpace(attr.String())
		if wantAttr != gotAttr {
			t.Errorf("val.Attr(%q): wanted %q, got %q", attrName, wantAttr, gotAttr)
		}
	}
}

func TestMessageV3(t *testing.T) {
	val, err := eval(`proto.package("skycfg.test_proto").MessageV3(
		f_int32 = 1010,
		f_int64 = 1020,
		f_uint32 = 1030,
		f_uint64 = 1040,
		f_float32 = 10.50,
		f_float64 = 10.60,
		f_string = "some string",
		f_bool = True,
		f_submsg = proto.package("skycfg.test_proto").MessageV3(
			f_string = "string in submsg",
		),
		r_string = ["r_string1", "r_string2"],
		r_submsg = [
			proto.package("skycfg.test_proto").MessageV3(
				f_string = "string in r_submsg",
			),
		],
		map_string = {
			"map_string key": "map_string val",
		},
		map_submsg = {
			"map_submsg key": proto.package("skycfg.test_proto").MessageV3(
				f_string = "map_submsg val",
			),
		},
		f_nested_submsg = proto.package("skycfg.test_proto").MessageV3.NestedMessage(
			f_string = "nested_submsg val",
		),
		f_toplevel_enum = proto.package("skycfg.test_proto").ToplevelEnumV3.TOPLEVEL_ENUM_V3_B,
		f_nested_enum = proto.package("skycfg.test_proto").MessageV3.NestedEnum.NESTED_ENUM_B,
		f_oneof_a = "string in oneof",
		f_bytes = "also some string",
	)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotMsg := mustProtoMessage(t, val)
	wantMsg := &pb.MessageV3{
		FInt32:   1010,
		FInt64:   1020,
		FUint32:  1030,
		FUint64:  1040,
		FFloat32: 10.50,
		FFloat64: 10.60,
		FString:  "some string",
		FBool:    true,
		FSubmsg: &pb.MessageV3{
			FString: "string in submsg",
		},
		RString: []string{"r_string1", "r_string2"},
		RSubmsg: []*pb.MessageV3{{
			FString: "string in r_submsg",
		}},
		MapString: map[string]string{
			"map_string key": "map_string val",
		},
		MapSubmsg: map[string]*pb.MessageV3{
			"map_submsg key": &pb.MessageV3{
				FString: "map_submsg val",
			},
		},
		FNestedSubmsg: &pb.MessageV3_NestedMessage{
			FString: "nested_submsg val",
		},
		FToplevelEnum: pb.ToplevelEnumV3_TOPLEVEL_ENUM_V3_B,
		FNestedEnum:   pb.MessageV3_NESTED_ENUM_B,
		FOneof:        &pb.MessageV3_FOneofA{"string in oneof"},
		FBytes:        []byte("also some string"),
	}
	checkProtoEqual(t, wantMsg, gotMsg)

	wantAttrs := map[string]string{
		"f_int32":         "1010",
		"f_int64":         "1020",
		"f_uint32":        "1030",
		"f_uint64":        "1040",
		"f_float32":       "10.5",
		"f_float64":       "10.6",
		"f_string":        `"some string"`,
		"f_bool":          "True",
		"f_submsg":        `<skycfg.test_proto.MessageV3 f_string:"string in submsg">`,
		"r_string":        `["r_string1", "r_string2"]`,
		"r_submsg":        `[<skycfg.test_proto.MessageV3 f_string:"string in r_submsg">]`,
		"map_string":      `{"map_string key": "map_string val"}`,
		"map_submsg":      `{"map_submsg key": <skycfg.test_proto.MessageV3 f_string:"map_submsg val">}`,
		"f_nested_submsg": `<skycfg.test_proto.MessageV3.NestedMessage f_string:"nested_submsg val">`,
		"f_toplevel_enum": `<skycfg.test_proto.ToplevelEnumV3 TOPLEVEL_ENUM_V3_B=1>`,
		"f_nested_enum":   `<skycfg.test_proto.MessageV3.NestedEnum NESTED_ENUM_B=1>`,
		"f_oneof_a":       `"string in oneof"`,
		"f_oneof_b":       `""`,
		"f_bytes":         `"also some string"`,
	}
	attrs := val.(starlark.HasAttrs)
	for attrName, wantAttr := range wantAttrs {
		attr, err := attrs.Attr(attrName)
		if err != nil {
			t.Fatalf("val.Attr(%q): %v", attrName, err)
		}
		gotAttr := removeRandomSpace(attr.String())
		if wantAttr != gotAttr {
			t.Errorf("val.Attr(%q): wanted\n%q\n%q", attrName, wantAttr, gotAttr)
		}
	}
}

func TestAttrValidation(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{

		// Scalar type mismatch
		{
			name:    "int32",
			src:     `MessageV3(f_int32 = '')`,
			wantErr: `TypeError: value "" (type "string") can't be assigned to type "int32".`,
		},
		{
			name:    "int64",
			src:     `MessageV3(f_int64 = '')`,
			wantErr: `TypeError: value "" (type "string") can't be assigned to type "int64".`,
		},
		{
			name:    "uint32",
			src:     `MessageV3(f_uint32 = '')`,
			wantErr: `TypeError: value "" (type "string") can't be assigned to type "uint32".`,
		},
		{
			name:    "uint64",
			src:     `MessageV3(f_uint64 = '')`,
			wantErr: `TypeError: value "" (type "string") can't be assigned to type "uint64".`,
		},
		{
			name:    "float32",
			src:     `MessageV3(f_float32 = '')`,
			wantErr: `TypeError: value "" (type "string") can't be assigned to type "float".`,
		},
		{
			name:    "float64",
			src:     `MessageV3(f_float64 = '')`,
			wantErr: `TypeError: value "" (type "string") can't be assigned to type "double".`,
		},
		{
			name:    "string",
			src:     `MessageV3(f_string = 0)`,
			wantErr: `TypeError: value 0 (type "int") can't be assigned to type "string".`,
		},
		{
			name:    "bool",
			src:     `MessageV3(f_bool = '')`,
			wantErr: `TypeError: value "" (type "string") can't be assigned to type "bool".`,
		},
		{
			name:    "enum",
			src:     `MessageV3(f_toplevel_enum = 0)`,
			wantErr: `TypeError: value 0 (type "int") can't be assigned to type "skycfg.test_proto.ToplevelEnumV3".`,
		},

		// Non-scalar type mismatch
		{
			name:    "string list assignment",
			src:     `MessageV3(r_string = {'': ''})`,
			wantErr: `TypeError: value {"": ""} (type "dict") can't be assigned to type "[]string".`,
		},
		{
			name:    "string list field assignment",
			src:     `MessageV3(r_string = [123])`,
			wantErr: `TypeError: value 123 (type "int") can't be assigned to type "string".`,
		},
		{
			name:    "string map assignment",
			src:     `MessageV3(map_string = [123])`,
			wantErr: `TypeError: value [123] (type "list") can't be assigned to type "map[string]string".`,
		},
		{
			name:    "string map key assignment",
			src:     `MessageV3(map_string = {123: ''})`,
			wantErr: `TypeError: value 123 (type "int") can't be assigned to type "string".`,
		},
		{
			name:    "string map value assignment",
			src:     `MessageV3(map_string = {'': 456})`,
			wantErr: `TypeError: value 456 (type "int") can't be assigned to type "string".`,
		},
		{
			name:    "message map value assignment",
			src:     `MessageV3(map_submsg = {'': 456})`,
			wantErr: `TypeError: value 456 (type "int") can't be assigned to type "skycfg.test_proto.MessageV3".`,
		},
		{
			name:    "message assignment with wrong type",
			src:     `MessageV3(f_submsg = pb.MessageV2())`,
			wantErr: `TypeError: value <skycfg.test_proto.MessageV2 > (type "skycfg.test_proto.MessageV2") can't be assigned to type "skycfg.test_proto.MessageV3".`,
		},

		// Repeated and map fields can't be assigned `None`. Scalar fields can't be assigned `None`
		// in proto3, but the error message is specialized.
		{
			name:    "none to scalar",
			src:     `MessageV3(f_int32 = None)`,
			wantErr: `TypeError: value None (type "NoneType") can't be assigned to type "int32" in proto3 mode.`,
		},
		{
			name:    "none to string list",
			src:     `MessageV3(r_string = None)`,
			wantErr: `TypeError: value None (type "NoneType") can't be assigned to type "[]string".`,
		},
		{
			name:    "none to string map",
			src:     `MessageV3(map_string = None)`,
			wantErr: `TypeError: value None (type "NoneType") can't be assigned to type "map[string]string".`,
		},
		{
			name:    "none to message is allowed",
			src:     `MessageV3(f_submsg = None)`,
			wantErr: "",
		},
		{
			name:    "none to message list",
			src:     `MessageV3(r_submsg = None)`,
			wantErr: `TypeError: value None (type "NoneType") can't be assigned to type "[]skycfg.test_proto.MessageV3".`,
		},

		// Numeric overflow
		{
			name:    "int32 overflow",
			src:     fmt.Sprintf(`MessageV3(f_int32 = %d + 1)`, math.MaxInt32),
			wantErr: `ValueError: value 2147483648 overflows type "int32".`,
		},
		{
			name:    "int32 underflow",
			src:     fmt.Sprintf(`MessageV3(f_int32 = %d - 1)`, math.MinInt32),
			wantErr: `ValueError: value -2147483649 overflows type "int32".`,
		},
		{
			name:    "int64 overflow",
			src:     fmt.Sprintf(`MessageV3(f_int64 = %d + 1)`, math.MaxInt64),
			wantErr: `ValueError: value 9223372036854775808 overflows type "int64".`,
		},
		{
			name:    "int64 underflow",
			src:     fmt.Sprintf(`MessageV3(f_int64 = %d - 1)`, math.MinInt64),
			wantErr: `ValueError: value -9223372036854775809 overflows type "int64".`,
		},
		{
			name:    "uint32 overflow",
			src:     fmt.Sprintf(`MessageV3(f_uint32 = %d + 1)`, math.MaxUint32),
			wantErr: `ValueError: value 4294967296 overflows type "uint32".`,
		},
		{
			name:    "uint32 underflow",
			src:     fmt.Sprintf(`MessageV3(f_uint32 = %d - 1)`, 0),
			wantErr: `ValueError: value -1 overflows type "uint32".`,
		},
		{
			name:    "uint64 overflow",
			src:     fmt.Sprintf(`MessageV3(f_uint64 = %d + 1)`, uint64(math.MaxUint64)),
			wantErr: `ValueError: value 18446744073709551616 overflows type "uint64".`,
		},
		{
			name:    "uint64 underflow",
			src:     fmt.Sprintf(`MessageV3(f_uint64 = %d - 1)`, 0),
			wantErr: `ValueError: value -1 overflows type "uint64".`,
		},
	}

	globals := starlark.StringDict{
		"pb": NewProtoPackage(newRegistry(), "skycfg.test_proto"),
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := starlark.Eval(&starlark.Thread{}, "", `pb.`+test.src, globals)
			if test.wantErr != "" {
				if err == nil {
					t.Errorf("eval(%q): expected error", test.src)
				} else if test.wantErr != err.Error() {
					t.Errorf("eval(%q): expected error\nexpected: %q\ngot: %q", test.src, test.wantErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error got: %v\n", err)
				}
			}
		})
	}
}

func TestProtoMessageString(t *testing.T) {
	val, err := eval(`proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := val.String()
	want := `<skycfg.test_proto.MessageV3 f_string:"some string">`
	if want != got {
		t.Fatalf("protoMessage.String(): wanted %q, got %q", want, got)
	}
}

func TestNestedMessages(t *testing.T) {
	testPb := `proto.package("skycfg.test_proto").`

	tests := []struct {
		src     string
		wantVal string
	}{
		{
			src:     testPb + `MessageV2.NestedMessage()`,
			wantVal: `<skycfg.test_proto.MessageV2.NestedMessage >`,
		},
		{
			src:     testPb + `MessageV2.NestedMessage.DoubleNestedMessage()`,
			wantVal: `<skycfg.test_proto.MessageV2.NestedMessage.DoubleNestedMessage >`,
		},

		{
			src:     testPb + `MessageV3.NestedMessage()`,
			wantVal: `<skycfg.test_proto.MessageV3.NestedMessage >`,
		},
		{
			src:     testPb + `MessageV3.NestedMessage.DoubleNestedMessage()`,
			wantVal: `<skycfg.test_proto.MessageV3.NestedMessage.DoubleNestedMessage >`,
		},
	}
	for _, test := range tests {
		gotVal, err := eval(test.src, nil)
		if err != nil {
			t.Fatal(err)
		}
		if test.wantVal != gotVal.String() {
			t.Errorf("eval(%q): expected value %q, got %q", test.src, test.wantVal, gotVal)
		}
	}
}

func TestProtoComparisonEqual(t *testing.T) {
	msg := &pb.MessageV2{
		RString: []string{"a", "b", "c"},
	}
	skyMsg, _ := NewMessage(msg)

	// create a separate msg to ensure the underlying reference in skyMsgOther is different
	msgOther := &pb.MessageV2{
		RString: []string{"a", "b", "c"},
	}
	skyMsgOther, _ := NewMessage(msgOther)
	ok, err := starlark.Compare(syntax.EQL, skyMsg, skyMsgOther)
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("Expected protos to be equal")
	}
}

func TestProtoComparisonNotEqual(t *testing.T) {
	msg := &pb.MessageV2{
		RString: []string{"a", "b", "c"},
	}
	skyMsg, _ := NewMessage(msg)

	// create a separate msg to ensure the underlying reference in skyMsgOther is different
	msgOther := &pb.MessageV2{
		RString: []string{"a", "b"},
	}
	skyMsgOther, _ := NewMessage(msgOther)

	ok, err := starlark.Compare(syntax.EQL, skyMsg, skyMsgOther)
	if err != nil {
		t.Error(err)
	}
	if ok {
		t.Error("Expected protos to not be equal")
	}

	ok, err = starlark.Compare(syntax.NEQ, skyMsg, skyMsgOther)
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("Expected protos to not be equal")
	}
}

func TestProtoSetDefaultV2(t *testing.T) {
	var setInt int32 = 123
	setString := "abc"
	defaultString := "default_str"

	tests := []struct {
		src  string
		want *pb.MessageV2
	}{
		{
			src: `proto.set_defaults(proto.package("skycfg.test_proto").MessageV2(f_int32 = 123))`,
			want: &pb.MessageV2{
				FInt32:  &setInt,
				FString: &defaultString,
			},
		},
		{
			src: `proto.set_defaults(proto.package("skycfg.test_proto").MessageV2(f_int32 = 123, f_string = "abc"))`,
			want: &pb.MessageV2{
				FInt32:  &setInt,
				FString: &setString,
			},
		},
		{
			src:  `proto.package("skycfg.test_proto").MessageV2()`,
			want: &pb.MessageV2{},
		},
		{
			src: `proto.set_defaults(proto.package("skycfg.test_proto").MessageV2(f_submsg = proto.package("skycfg.test_proto").MessageV2()))`,
			want: &pb.MessageV2{
				FSubmsg: &pb.MessageV2{},
				FString: &defaultString,
			},
		},
	}
	for _, test := range tests {
		val, err := eval(test.src, nil)
		if err != nil {
			t.Fatal(err)
		}
		gotMsg := mustProtoMessage(t, val)
		checkProtoEqual(t, test.want, gotMsg)
	}
}

func TestProtoSetDefaultV3(t *testing.T) {
	tests := []struct {
		src  string
		want *pb.MessageV3
	}{
		{
			src: `proto.set_defaults(proto.package("skycfg.test_proto").MessageV3(f_int32 = 123))`,
			want: &pb.MessageV3{
				FInt32: 123,
			},
		},
		{
			src: `proto.set_defaults(proto.package("skycfg.test_proto").MessageV3(f_int32 = 123, f_string = "abc"))`,
			want: &pb.MessageV3{
				FInt32:  123,
				FString: "abc",
			},
		},
		{
			src:  `proto.package("skycfg.test_proto").MessageV3()`,
			want: &pb.MessageV3{},
		},
		{
			src: `proto.set_defaults(proto.package("skycfg.test_proto").MessageV3(f_submsg = proto.package("skycfg.test_proto").MessageV3()))`,
			want: &pb.MessageV3{
				FSubmsg: &pb.MessageV3{},
			},
		},
	}
	for _, test := range tests {
		val, err := eval(test.src, nil)
		if err != nil {
			t.Fatal(err)
		}
		gotMsg := mustProtoMessage(t, val)
		checkProtoEqual(t, test.want, gotMsg)
	}
}

func TestProtoClearV2(t *testing.T) {
	val, err := eval(`proto.clear(proto.package("skycfg.test_proto").MessageV2(
		f_string = "some string",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotMsg := mustProtoMessage(t, val)
	wantMsg := &pb.MessageV2{}
	checkProtoEqual(t, wantMsg, gotMsg)
}

func TestProtoClearV3(t *testing.T) {
	val, err := eval(`proto.clear(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotMsg := mustProtoMessage(t, val)
	wantMsg := &pb.MessageV3{
		FInt32:   0,
		FInt64:   0,
		FUint32:  0,
		FUint64:  0,
		FFloat32: 0.0,
		FFloat64: 0.0,
		FString:  "",
		FBool:    false,
	}
	checkProtoEqual(t, wantMsg, gotMsg)
}

func TestProtoMergeV2(t *testing.T) {
	val, err := eval(`proto.merge(proto.package("skycfg.test_proto").MessageV2(
		f_int32 = 1010,
		f_uint32 = 1030,
		f_string = "some string",
		f_submsg = proto.package("skycfg.test_proto").MessageV2(
			f_int32 = 1010,
			f_string = "f_submsg msg1",
		),
		r_string = ["r_string1", "r_string2"],
		r_submsg = [
			proto.package("skycfg.test_proto").MessageV2(
				f_string = "r_submsg.f_string msg1",
			),
		],
		map_string = {
			"map_string key shared": "map_string msg1",
			"map_string key msg1": "map_string msg1",
		},
		map_submsg = {
			"map_submsg key": proto.package("skycfg.test_proto").MessageV2(
				f_string = "map_submsg.f_string msg1",
			),
		},
		f_nested_submsg = proto.package("skycfg.test_proto").MessageV2.NestedMessage(
			f_string = "f_nested_submsg.f_string msg1",
		),
		f_toplevel_enum = proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B,
		f_nested_enum = proto.package("skycfg.test_proto").MessageV2.NestedEnum.NESTED_ENUM_B,
		f_oneof_a = "f_oneof_a msg1 string in oneof",
		f_bytes = "f_bytes msg1",
	), proto.package("skycfg.test_proto").MessageV2(
		f_int32 = 2010,
		f_int64 = 2020,
		f_string = "another string",
		f_submsg = proto.package("skycfg.test_proto").MessageV2(
			f_string = "f_submsg msg2",
			f_int64 = 2020,
		),
		r_string = ["r_string3", "r_string4"],
		r_submsg = [
			proto.package("skycfg.test_proto").MessageV2(
				f_string = "r_submsg.f_string msg2",
			),
		],
		map_string = {
			"map_string key shared": "map_string msg2",
			"map_string key msg2": "map_string msg2",
		},
		map_submsg = {
			"map_submsg key": proto.package("skycfg.test_proto").MessageV2(
				f_string = "map_submsg.f_string msg2",
			),
		},
		f_nested_submsg = proto.package("skycfg.test_proto").MessageV2.NestedMessage(
			f_string = "f_nested_submsg.f_string msg2",
		),
		f_toplevel_enum = proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B,
		f_nested_enum = proto.package("skycfg.test_proto").MessageV2.NestedEnum.NESTED_ENUM_B,
		f_oneof_b = "f_oneof_b msg2 string in oneof",
		f_bytes = "f_bytes msg2",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotMsg := mustProtoMessage(t, val)

	// Merge msg2 onto msg1
	msg1 := &pb.MessageV2{
		FInt32: proto.Int32(1010),
		// FInt64: empty for merging src=nil field
		FUint32: proto.Uint32(1030),
		FString: proto.String("some string"),
		FSubmsg: &pb.MessageV2{
			FInt32:  proto.Int32(1010),
			FString: proto.String("f_submsg msg1"),
		},
		RString: []string{"r_string1", "r_string2"},
		RSubmsg: []*pb.MessageV2{{
			FString: proto.String("r_submsg.f_string msg1"),
		}},
		MapString: map[string]string{
			"map_string key shared": "map_string msg1",
			"map_string key msg1":   "map_string msg1",
		},
		MapSubmsg: map[string]*pb.MessageV2{
			"map_submsg key": &pb.MessageV2{
				FString: proto.String("map_submsg.f_string msg1"),
			},
		},
		FNestedSubmsg: &pb.MessageV2_NestedMessage{
			FString: proto.String("f_nested_submsg.f_string msg1"),
		},
		FToplevelEnum: pb.ToplevelEnumV2_TOPLEVEL_ENUM_V2_B.Enum(),
		FNestedEnum:   pb.MessageV2_NESTED_ENUM_B.Enum(),
		FOneof:        &pb.MessageV2_FOneofA{"f_oneof_a msg1 string in oneof"},
		FBytes:        []byte("f_bytes msg1"),
	}
	msg2 := &pb.MessageV2{
		FInt32: proto.Int32(2010),
		FInt64: proto.Int64(2020),
		// FUint32: empty for mergin dst=nil field
		FString: proto.String("another string"),
		FSubmsg: &pb.MessageV2{
			FString: proto.String("f_submsg msg2"),
			FInt64:  proto.Int64(2020),
		},
		RString: []string{"r_string3", "r_string4"},
		RSubmsg: []*pb.MessageV2{{
			FString: proto.String("r_submsg.f_string msg2"),
		}},
		MapString: map[string]string{
			"map_string key shared": "map_string msg2",
			"map_string key msg2":   "map_string msg2",
		},
		MapSubmsg: map[string]*pb.MessageV2{
			"map_submsg key": &pb.MessageV2{
				FString: proto.String("map_submsg.f_string msg2"),
			},
		},
		FNestedSubmsg: &pb.MessageV2_NestedMessage{
			FString: proto.String("f_nested_submsg.f_string msg2"),
		},
		FToplevelEnum: pb.ToplevelEnumV2_TOPLEVEL_ENUM_V2_B.Enum(),
		FNestedEnum:   pb.MessageV2_NESTED_ENUM_B.Enum(),
		FOneof:        &pb.MessageV2_FOneofB{"f_oneof_b msg2 string in oneof"},
		FBytes:        []byte("f_bytes msg2"),
	}
	proto.Merge(msg1, msg2)

	checkProtoEqual(t, msg1, gotMsg)
}

func TestProtoMergeV3(t *testing.T) {
	val, err := eval(`proto.merge(proto.package("skycfg.test_proto").MessageV3(
		f_int32 = 1010,
		f_uint32 = 1030,
		f_string = "some string",
		f_submsg = proto.package("skycfg.test_proto").MessageV3(
			f_int32 = 1010,
			f_string = "f_submsg msg1",
		),
		r_string = ["r_string1", "r_string2"],
		r_submsg = [
			proto.package("skycfg.test_proto").MessageV3(
				f_string = "r_submsg.f_string msg1",
			),
		],
		map_string = {
			"map_string key shared": "map_string msg1",
			"map_string key msg1": "map_string msg1",
		},
		map_submsg = {
			"map_submsg key": proto.package("skycfg.test_proto").MessageV3(
				f_string = "map_submsg.f_string msg1",
			),
		},
		f_nested_submsg = proto.package("skycfg.test_proto").MessageV3.NestedMessage(
			f_string = "f_nested_submsg.f_string msg1",
		),
		f_toplevel_enum = proto.package("skycfg.test_proto").ToplevelEnumV3.TOPLEVEL_ENUM_V3_B,
		f_nested_enum = proto.package("skycfg.test_proto").MessageV3.NestedEnum.NESTED_ENUM_B,
		f_oneof_a = "f_oneof_a msg1 string in oneof",
		f_bytes = "f_bytes msg1",
	), proto.package("skycfg.test_proto").MessageV3(
		f_int32 = 2010,
		f_int64 = 2020,
		f_string = "another string",
		f_submsg = proto.package("skycfg.test_proto").MessageV3(
			f_string = "f_submsg msg2",
			f_int64 = 2020,
		),
		r_string = ["r_string3", "r_string4"],
		r_submsg = [
			proto.package("skycfg.test_proto").MessageV3(
				f_string = "r_submsg.f_string msg2",
			),
		],
		map_string = {
			"map_string key shared": "map_string msg2",
			"map_string key msg2": "map_string msg2",
		},
		map_submsg = {
			"map_submsg key": proto.package("skycfg.test_proto").MessageV3(
				f_string = "map_submsg.f_string msg2",
			),
		},
		f_nested_submsg = proto.package("skycfg.test_proto").MessageV3.NestedMessage(
			f_string = "f_nested_submsg.f_string msg2",
		),
		f_toplevel_enum = proto.package("skycfg.test_proto").ToplevelEnumV3.TOPLEVEL_ENUM_V3_B,
		f_nested_enum = proto.package("skycfg.test_proto").MessageV3.NestedEnum.NESTED_ENUM_B,
		f_oneof_b = "f_oneof_b msg2 string in oneof",
		f_bytes = "f_bytes msg2",
	))`, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotMsg := mustProtoMessage(t, val)

	// Merge msg2 onto msg1
	msg1 := &pb.MessageV3{
		FInt32: 1010,
		// FInt64: empty for merging src=nil field
		FUint32: 1030,
		FString: "some string",
		FSubmsg: &pb.MessageV3{
			FInt32:  1010,
			FString: "f_submsg msg1",
		},
		RString: []string{"r_string1", "r_string2"},
		RSubmsg: []*pb.MessageV3{{
			FString: "r_submsg.f_string msg1",
		}},
		MapString: map[string]string{
			"map_string key shared": "map_string msg1",
			"map_string key msg1":   "map_string msg1",
		},
		MapSubmsg: map[string]*pb.MessageV3{
			"map_submsg key": &pb.MessageV3{
				FString: "map_submsg.f_string msg1",
			},
		},
		FNestedSubmsg: &pb.MessageV3_NestedMessage{
			FString: "f_nested_submsg.f_string msg1",
		},
		FToplevelEnum: pb.ToplevelEnumV3_TOPLEVEL_ENUM_V3_B,
		FNestedEnum:   pb.MessageV3_NESTED_ENUM_B,
		FOneof:        &pb.MessageV3_FOneofA{"f_oneof_a msg1 string in oneof"},
		FBytes:        []byte("f_bytes msg1"),
	}
	msg2 := &pb.MessageV3{
		FInt32: 2010,
		FInt64: 2020,
		// FUint32: empty for mergin dst=nil field
		FString: "another string",
		FSubmsg: &pb.MessageV3{
			FString: "f_submsg msg2",
			FInt64:  2020,
		},
		RString: []string{"r_string3", "r_string4"},
		RSubmsg: []*pb.MessageV3{{
			FString: "r_submsg.f_string msg2",
		}},
		MapString: map[string]string{
			"map_string key shared": "map_string msg2",
			"map_string key msg2":   "map_string msg2",
		},
		MapSubmsg: map[string]*pb.MessageV3{
			"map_submsg key": &pb.MessageV3{
				FString: "map_submsg.f_string msg2",
			},
		},
		FNestedSubmsg: &pb.MessageV3_NestedMessage{
			FString: "f_nested_submsg.f_string msg2",
		},
		FToplevelEnum: pb.ToplevelEnumV3_TOPLEVEL_ENUM_V3_B,
		FNestedEnum:   pb.MessageV3_NESTED_ENUM_B,
		FOneof:        &pb.MessageV3_FOneofB{"f_oneof_b msg2 string in oneof"},
		FBytes:        []byte("f_bytes msg2"),
	}
	proto.Merge(msg1, msg2)

	checkProtoEqual(t, msg1, gotMsg)
}

func TestProtoMergeDiffTypes(t *testing.T) {
	errorMsg := "proto.merge: types are not the same: got skycfg.test_proto.MessageV3 and skycfg.test_proto.MessageV2"
	globals := starlark.StringDict{
		"proto": NewModule(newRegistry()),
	}
	src, err := starlark.Eval(&starlark.Thread{}, "",
		`proto.merge(proto.package("skycfg.test_proto").MessageV2(), proto.package("skycfg.test_proto").MessageV3())`, globals)
	if err == nil {
		t.Errorf("expected error %q, got %q", errorMsg, src)
	}
	if errorMsg != err.Error() {
		t.Errorf("expected error %q, got %q", errorMsg, err.Error())
	}
}

// Pre 1.0 Skycfg allowed maps to be constructed with None values for proto2 (see protoMap.SetKey)
func TestMapNoneCompatibility(t *testing.T) {
	val, err := evalFunc(`
def fun():
    pb = proto.package("skycfg.test_proto")
    msg = pb.MessageV2()
    m = {
	    "a": pb.MessageV2(),
	    "b": pb.MessageV2(),
	    "c": pb.MessageV2(),
	    "d": None,
    }
    msg.map_submsg = m

    m2 = msg.map_submsg
    m2["b"] = None
    m2.setdefault("e", None)
    m2.update([("c", None)])

    return msg
`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := mustProtoMessage(t, val).(*pb.MessageV2)

	checkProtoEqual(t, &pb.MessageV2{
		MapSubmsg: map[string]*pb.MessageV2{
			"a": &pb.MessageV2{},
		},
	}, got)

	// Confirm this only works for all in proto2, only message values in proto3
	// This is an artifact of set to None being allow for scalar values in proto2
	val, err = evalFunc(`
def fun():
    pb = proto.package("skycfg.test_proto")
    msg = pb.MessageV2(
	map_string = {
	    "a": None
        }
    )
    return msg
`, nil)
	if err != nil {
		t.Fatal(err)
	}

	val, err = evalFunc(`
def fun():
    pb = proto.package("skycfg.test_proto")
    msg = pb.MessageV3(
	map_string = {
	    "a": None
        }
    )
    return msg
`, nil)
	wantErr := fmt.Errorf(`TypeError: value None (type "NoneType") can't be assigned to type "string" in proto3 mode.`)
	if !checkError(err, wantErr) {
		t.Fatalf("eval: expected error %v, got %v", wantErr, err)
	}

	// An odd resulting behavior of both ensuring assignment does not copy
	// and setting to None deletes is that assignment can mutate a raw starlark dict
	// This is not ideal but this test is here to just document the behavior
	val, err = evalFunc(`
def fun():
    pb = proto.package("skycfg.test_proto")
    a = {
        "ka": "va",
        "ba": None,
    }
    msg = pb.MessageV2(
	map_string = a
    )
    return a
`, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"ka": "va"}`
	if want != val.String() {
		t.Fatalf("Result differed\nwant: %s\ngot : %s", want, val.String())
	}
}

func TestUnsetProto2Fields(t *testing.T) {
	// Proto v2 distinguishes between unset and set-to-empty.
	msg, err := eval(`proto.package("skycfg.test_proto").MessageV2(
                f_string = None,
        )`, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := mustProtoMessage(t, msg)
	want := &pb.MessageV2{
		FString: nil,
	}
	checkProtoEqual(t, want, got)
}
