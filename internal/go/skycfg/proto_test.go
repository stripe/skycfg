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

package skycfg

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	gogo_proto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	_ "github.com/gogo/protobuf/types"

	pb "github.com/stripe/skycfg/test_proto"
)

func init() {
	resolve.AllowFloat = true
}

type gogoRegistry struct{}

func (*gogoRegistry) UnstableProtoMessageType(name string) (reflect.Type, error) {
	if t := gogo_proto.MessageType(name); t != nil {
		return t, nil
	}
	return nil, nil
}

func (*gogoRegistry) UnstableEnumValueMap(name string) map[string]int32 {
	return gogo_proto.EnumValueMap(name)
}

func skyEval(t *testing.T, src string) starlark.Value {
	t.Helper()
	globals := starlark.StringDict{
		"proto":      NewProtoModule(nil),
		"gogo_proto": NewProtoModule(&gogoRegistry{}),
	}
	val, err := starlark.Eval(&starlark.Thread{}, "", src, globals)
	if err != nil {
		t.Fatalf("eval(%q): %v", src, err)
	}
	return val
}

// Generate a diff of two structs, which may contain protobuf messages.
func ProtoDiff(want, got interface{}) string {
	return (&pretty.Config{
		Diffable:          true,
		IncludeUnexported: false,
		Formatter:         pretty.DefaultFormatter,
		SkipStructField: func(structType reflect.Type, field reflect.StructField) bool {
			return strings.HasPrefix(field.Name, "XXX_")
		},
	}).Compare(want, got)
}

func TestProtoPackage(t *testing.T) {
	val := skyEval(t, `proto.package("skycfg.test_proto")`)
	if !strings.Contains(val.String(), "skycfg.test_proto") {
		t.Fatalf("proto.package() should return a value that can describe itself")
	}
}

func TestProtoMessageString(t *testing.T) {
	val := skyEval(t, `proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	)`)
	got := val.String()
	want := `<skycfg.test_proto.MessageV3 f_string:"some string" >`
	if want != got {
		t.Fatalf("skyProtoMessage.String(): wanted %q, got %q", want, got)
	}
}

func TestProtoSetDefaultV2(t *testing.T) {
	val := skyEval(t, `proto.set_defaults(proto.package("skycfg.test_proto").MessageV2())`)
	gotMsg := val.(*skyProtoMessage).msg
	wantMsg := &pb.MessageV2{
		FString: proto.String("default_str"),
	}
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}
}

func TestProtoSetDefaultV3(t *testing.T) {
	val := skyEval(t, `proto.set_defaults(proto.package("skycfg.test_proto").MessageV3())`)
	gotMsg := val.(*skyProtoMessage).msg
	wantMsg := &pb.MessageV3{}
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}
}

func TestProtoClearV2(t *testing.T) {
	val := skyEval(t, `proto.clear(proto.package("skycfg.test_proto").MessageV2(
		f_string = "some string",
	))`)
	gotMsg := val.(*skyProtoMessage).msg
	wantMsg := &pb.MessageV2{}
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}
}

func TestProtoClearV3(t *testing.T) {
	val := skyEval(t, `proto.clear(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`)
	gotMsg := val.(*skyProtoMessage).msg
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
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}
}

func TestProtoMergeV2(t *testing.T) {
	val := skyEval(t, `proto.merge(proto.package("skycfg.test_proto").MessageV2(
		f_int32 = 1010,
		f_string = "some string",
		r_string = ["r_string1", "r_string2"],
	), proto.package("skycfg.test_proto").MessageV2(
		f_int32 = 2010,
		f_int64 = 2020,
		f_string = "another string",
		r_string = ["r_string3", "r_string4"],
	))`)
	gotMsg := val.(*skyProtoMessage).msg
	wantMsg := &pb.MessageV2{
		FInt32:  proto.Int32(2010),
		FInt64:  proto.Int64(2020),
		FString: proto.String("another string"),
		RString: []string{"r_string1", "r_string2", "r_string3", "r_string4"},
	}
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}
}

func TestProtoMergeV3(t *testing.T) {
	val := skyEval(t, `proto.merge(proto.package("skycfg.test_proto").MessageV3(
		f_int32 = 1010,
		f_uint32 = 1030,
		f_float32 = 10.50,
		f_float64 = 10.60,
		f_string = "some string",
		f_bool = True,
		r_string = ["r_string1", "r_string2"],
	), proto.package("skycfg.test_proto").MessageV3(
		f_int32 = 2010,
		f_int64 = 2020,
		f_uint32 = 2030,
		f_uint64 = 2040,
		f_float32 = 20.50,
		f_float64 = 20.60,
		f_string = "another string",
		r_string = ["r_string3", "r_string4"],
	))`)
	gotMsg := val.(*skyProtoMessage).msg
	wantMsg := &pb.MessageV3{
		FInt32:   2010,
		FInt64:   2020,
		FUint32:  2030,
		FUint64:  2040,
		FFloat32: 20.50,
		FFloat64: 20.60,
		FString:  "another string",
		FBool:    true,
		RString:  []string{"r_string1", "r_string2", "r_string3", "r_string4"},
	}
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}
}

func TestProtoMergeDiffTypes(t *testing.T) {
	errorMsg := "proto.merge: types are not the same: got skycfg.test_proto.MessageV3 and skycfg.test_proto.MessageV2"
	globals := starlark.StringDict{
		"proto": NewProtoModule(nil),
	}
	src, err := starlark.Eval(&starlark.Thread{}, "",
		`proto.merge(proto.package("skycfg.test_proto").MessageV2(), proto.package("skycfg.test_proto").MessageV3())`, globals)
	if err == nil {
		t.Errorf("expected error, got %q", src)
	}
	if errorMsg != err.Error() {
		t.Errorf("expected error %q, got %q", errorMsg, err.Error())
	}
}

func TestProtoToText(t *testing.T) {
	val := skyEval(t, `proto.to_text(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`)
	got := string(val.(starlark.String))
	want := "f_string:\"some string\" "
	if want != got {
		t.Fatalf("to_text: wanted %q, got %q", want, got)
	}
}

func TestProtoToTextCompact(t *testing.T) {
	val := skyEval(t, `proto.to_text(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=True)`)
	got := string(val.(starlark.String))
	want := "f_string:\"some string\" "
	if want != got {
		t.Fatalf("to_text_compact: wanted %q, got %q", want, got)
	}
}

func TestProtoToTextFull(t *testing.T) {
	val := skyEval(t, `proto.to_text(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=False)`)
	got := string(val.(starlark.String))
	want := "f_string: \"some string\"\n"
	if want != got {
		t.Fatalf("to_text_full: wanted %q, got %q", want, got)
	}
}

func TestProtoToJson(t *testing.T) {
	val := skyEval(t, `proto.to_json(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`)
	got := string(val.(starlark.String))
	want := `{"f_string":"some string"}`
	if want != got {
		t.Fatalf("to_json: wanted %q, got %q", want, got)
	}
}

func TestProtoToJsonCompact(t *testing.T) {
	val := skyEval(t, `proto.to_json(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=True)`)
	got := string(val.(starlark.String))
	want := `{"f_string":"some string"}`
	if want != got {
		t.Fatalf("to_json_compact: wanted %q, got %q", want, got)
	}
}

func TestProtoToJsonFull(t *testing.T) {
	val := skyEval(t, `proto.to_json(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	), compact=False)`)
	got := string(val.(starlark.String))
	want := "{\n\t\"f_string\": \"some string\"\n}"
	if want != got {
		t.Fatalf("to_json_full: wanted %q, got %q", want, got)
	}
}

func TestProtoToYaml(t *testing.T) {
	val := skyEval(t, `proto.to_yaml(proto.package("skycfg.test_proto").MessageV3(
		f_string = "some string",
	))`)
	got := string(val.(starlark.String))
	want := "f_string: some string\n"
	if want != got {
		t.Fatalf("to_yaml: wanted %q, got %q", want, got)
	}
}

func TestMessageAttrNames(t *testing.T) {
	val := skyEval(t, `proto.package("skycfg.test_proto").MessageV3()`)
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
	}
	sort.Strings(want)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("skyProtoMessage.AttrNames: wanted %#v, got %#v", want, got)
	}
}

func TestMessageV2(t *testing.T) {
	val := skyEval(t, `proto.package("skycfg.test_proto").MessageV2(
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
	)`)
	gotMsg := val.(*skyProtoMessage).msg
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
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}

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
		"f_oneof_b":       `None`,
		"f_bytes":         `"also some string"`,
	}
	attrs := val.(starlark.HasAttrs)
	for attrName, wantAttr := range wantAttrs {
		attr, err := attrs.Attr(attrName)
		if err != nil {
			t.Fatalf("val.Attr(%q): %v", attrName, err)
		}
		gotAttr := attr.String()
		if wantAttr != gotAttr {
			t.Errorf("val.Attr(%q): wanted %q, got %q", attrName, wantAttr, gotAttr)
		}
	}
}

func TestMessageV3(t *testing.T) {
	val := skyEval(t, `proto.package("skycfg.test_proto").MessageV3(
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
	)`)
	gotMsg := val.(*skyProtoMessage).msg
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
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}

	wantAttrs := map[string]string{
		"f_int32":         "1010",
		"f_int64":         "1020",
		"f_uint32":        "1030",
		"f_uint64":        "1040",
		"f_float32":       "10.5",
		"f_float64":       "10.6",
		"f_string":        `"some string"`,
		"f_bool":          "True",
		"f_submsg":        `<skycfg.test_proto.MessageV3 f_string:"string in submsg" >`,
		"r_string":        `["r_string1", "r_string2"]`,
		"r_submsg":        `[<skycfg.test_proto.MessageV3 f_string:"string in r_submsg" >]`,
		"map_string":      `{"map_string key": "map_string val"}`,
		"map_submsg":      `{"map_submsg key": <skycfg.test_proto.MessageV3 f_string:"map_submsg val" >}`,
		"f_nested_submsg": `<skycfg.test_proto.MessageV3.NestedMessage f_string:"nested_submsg val" >`,
		"f_toplevel_enum": `<skycfg.test_proto.ToplevelEnumV3 TOPLEVEL_ENUM_V3_B=1>`,
		"f_nested_enum":   `<skycfg.test_proto.MessageV3.NestedEnum NESTED_ENUM_B=1>`,
		"f_oneof_a":       `"string in oneof"`,
		"f_oneof_b":       `None`,
		"f_bytes":         `"also some string"`,
	}
	attrs := val.(starlark.HasAttrs)
	for attrName, wantAttr := range wantAttrs {
		attr, err := attrs.Attr(attrName)
		if err != nil {
			t.Fatalf("val.Attr(%q): %v", attrName, err)
		}
		gotAttr := attr.String()
		if wantAttr != gotAttr {
			t.Errorf("val.Attr(%q): wanted %q, got %q", attrName, wantAttr, gotAttr)
		}
	}
}

func TestMessageGogo(t *testing.T) {
	val := skyEval(t, `gogo_proto.package("skycfg.test_proto").MessageGogo(
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
		f_nested_submsg = gogo_proto.package("skycfg.test_proto").MessageGogo.NestedMessage(
			f_string = "nested_submsg val",
		),
		f_toplevel_enum = proto.package("skycfg.test_proto").ToplevelEnumV2.TOPLEVEL_ENUM_V2_B,
		f_nested_enum = gogo_proto.package("skycfg.test_proto").MessageGogo.NestedEnum.NESTED_ENUM_B,
		f_oneof_a = "string in oneof",
		f_bytes = "also some string",
		f_duration = proto.package("google.protobuf").Duration(seconds = 1),
	)`)
	gotMsg := val.(*skyProtoMessage).msg
	wantMsg := &pb.MessageGogo{
		FInt32:   proto.Int32(1010),
		FInt64:   proto.Int64(1020),
		FUint32:  proto.Uint32(1030),
		FUint64:  proto.Uint64(1040),
		FFloat32: proto.Float32(10.50),
		FFloat64: proto.Float64(10.60),
		FString:  proto.String("some string"),
		FBool:    proto.Bool(true),
		FSubmsg: pb.MessageV2{
			FString: proto.String("string in submsg"),
		},
		RString: []string{"r_string1", "r_string2"},
		RSubmsg: []pb.MessageV2{{
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
		FNestedSubmsg: &pb.MessageGogo_NestedMessage{
			FString: proto.String("nested_submsg val"),
		},
		FToplevelEnum: pb.ToplevelEnumV2_TOPLEVEL_ENUM_V2_B.Enum(),
		FNestedEnum:   pb.MessageGogo_NESTED_ENUM_B.Enum(),
		FOneof:        &pb.MessageGogo_FOneofA{"string in oneof"},
		FBytes:        []byte("also some string"),
		FDuration:     time.Second,
	}
	if diff := ProtoDiff(wantMsg, gotMsg); diff != "" {
		t.Fatalf("diff from expected message:\n%s", diff)
	}

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
		"f_nested_submsg": `<skycfg.test_proto.MessageGogo.NestedMessage f_string:"nested_submsg val" >`,
		"f_toplevel_enum": `<skycfg.test_proto.ToplevelEnumV2 TOPLEVEL_ENUM_V2_B=1>`,
		"f_nested_enum":   `<skycfg.test_proto.MessageGogo.NestedEnum NESTED_ENUM_B=1>`,
		"f_oneof_a":       `"string in oneof"`,
		"f_oneof_b":       `None`,
		"f_bytes":         `"also some string"`,
		"f_duration":      `<google.protobuf.Duration seconds:1 >`,
	}
	attrs := val.(starlark.HasAttrs)
	for attrName, wantAttr := range wantAttrs {
		attr, err := attrs.Attr(attrName)
		if err != nil {
			t.Fatalf("val.Attr(%q): %v", attrName, err)
		}
		gotAttr := attr.String()
		if wantAttr != gotAttr {
			t.Errorf("val.Attr(%q): wanted %q, got %q", attrName, wantAttr, gotAttr)
		}
	}
}

func TestAttrValidation(t *testing.T) {
	globals := starlark.StringDict{
		"proto": NewProtoModule(nil),
	}
	tests := []struct {
		src     string
		wantErr string
	}{

		// Scalar type mismatch
		{
			src:     `MessageV3(f_int32 = '')`,
			wantErr: "TypeError: value \"\" (type `string') can't be assigned to type `int32'.",
		},
		{
			src:     `MessageV3(f_int64 = '')`,
			wantErr: "TypeError: value \"\" (type `string') can't be assigned to type `int64'.",
		},
		{
			src:     `MessageV3(f_uint32 = '')`,
			wantErr: "TypeError: value \"\" (type `string') can't be assigned to type `uint32'.",
		},
		{
			src:     `MessageV3(f_uint64 = '')`,
			wantErr: "TypeError: value \"\" (type `string') can't be assigned to type `uint64'.",
		},
		{
			src:     `MessageV3(f_float32 = '')`,
			wantErr: "TypeError: value \"\" (type `string') can't be assigned to type `float32'.",
		},
		{
			src:     `MessageV3(f_float64 = '')`,
			wantErr: "TypeError: value \"\" (type `string') can't be assigned to type `float64'.",
		},
		{
			src:     `MessageV3(f_string = 0)`,
			wantErr: "TypeError: value 0 (type `int') can't be assigned to type `string'.",
		},
		{
			src:     `MessageV3(f_bool = '')`,
			wantErr: "TypeError: value \"\" (type `string') can't be assigned to type `bool'.",
		},
		{
			src:     `MessageV3(f_toplevel_enum = 0)`,
			wantErr: "TypeError: value 0 (type `int') can't be assigned to type `skycfg.test_proto.ToplevelEnumV3'.",
		},

		// Non-scalar type mismatch
		{
			src:     `MessageV3(r_string = {'': ''})`,
			wantErr: "TypeError: value {\"\": \"\"} (type `dict') can't be assigned to type `[]string'.",
		},
		{
			src:     `MessageV3(r_string = [123])`,
			wantErr: "TypeError: value 123 (type `int') can't be assigned to type `string'.",
		},
		{
			src:     `MessageV3(map_string = [123])`,
			wantErr: "TypeError: value [123] (type `list') can't be assigned to type `map[string]string'.",
		},
		{
			src:     `MessageV3(map_string = {123: ''})`,
			wantErr: "TypeError: value 123 (type `int') can't be assigned to type `string'.",
		},
		{
			src:     `MessageV3(map_string = {'': 456})`,
			wantErr: "TypeError: value 456 (type `int') can't be assigned to type `string'.",
		},
		{
			src:     `MessageV3(map_submsg = {'': 456})`,
			wantErr: "TypeError: value 456 (type `int') can't be assigned to type `skycfg.test_proto.MessageV3'.",
		},
		{
			src:     `MessageV3(f_submsg = proto.package("skycfg.test_proto").MessageV2())`,
			wantErr: "TypeError: value <skycfg.test_proto.MessageV2 > (type `skycfg.test_proto.MessageV2') can't be assigned to type `skycfg.test_proto.MessageV3'.",
		},

		// Repeated and map fields can't be assigned `None`. Scalar fields can't be assigned `None`
		// in proto3, but the error message is specialized.
		{
			src:     `MessageV3(f_int32 = None)`,
			wantErr: "TypeError: value None can't be assigned to type `int32' in proto3 mode.",
		},
		{
			src:     `MessageV3(r_string = None)`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `[]string'.",
		},
		{
			src:     `MessageV3(map_string = None)`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `map[string]string'.",
		},

		// Numeric overflow
		{
			src:     fmt.Sprintf(`MessageV3(f_int32 = %d + 1)`, math.MaxInt32),
			wantErr: "ValueError: value 2147483648 overflows type `int32'.",
		},
		{
			src:     fmt.Sprintf(`MessageV3(f_int32 = %d - 1)`, math.MinInt32),
			wantErr: "ValueError: value -2147483649 overflows type `int32'.",
		},
		{
			src:     fmt.Sprintf(`MessageV3(f_int64 = %d + 1)`, math.MaxInt64),
			wantErr: "ValueError: value 9223372036854775808 overflows type `int64'.",
		},
		{
			src:     fmt.Sprintf(`MessageV3(f_int64 = %d - 1)`, math.MinInt64),
			wantErr: "ValueError: value -9223372036854775809 overflows type `int64'.",
		},
		{
			src:     fmt.Sprintf(`MessageV3(f_uint32 = %d + 1)`, math.MaxUint32),
			wantErr: "ValueError: value 4294967296 overflows type `uint32'.",
		},
		{
			src:     fmt.Sprintf(`MessageV3(f_uint32 = %d - 1)`, 0),
			wantErr: "ValueError: value -1 overflows type `uint32'.",
		},
		{
			src:     fmt.Sprintf(`MessageV3(f_uint64 = %d + 1)`, uint64(math.MaxUint64)),
			wantErr: "ValueError: value 18446744073709551616 overflows type `uint64'.",
		},
		{
			src:     fmt.Sprintf(`MessageV3(f_uint64 = %d - 1)`, 0),
			wantErr: "ValueError: value -1 overflows type `uint64'.",
		},
	}
	for _, test := range tests {
		_, err := starlark.Eval(&starlark.Thread{}, "", `proto.package("skycfg.test_proto").`+test.src, globals)
		if err == nil {
			t.Errorf("eval(%q): expected error", test.src)
			continue
		}
		if test.wantErr != err.Error() {
			t.Errorf("eval(%q): expected error %q, got %q", test.src, test.wantErr, err.Error())
		}
	}
}

func TestListMutation(t *testing.T) {
	tests := []struct {
		src     string
		want    []string
		wantErr string
	}{
		{
			src:  `msg.r_string.clear()`,
			want: []string{},
		},
		{
			src:  `msg.r_string.append('d')`,
			want: []string{"a", "b", "c", "d"},
		},
		{
			src:     `msg.r_string.append(None)`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `string'.",
		},
		{
			src:     `msg.r_submsg.append(None)`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `skycfg.test_proto.MessageV2'.",
		},
		{
			src:  `msg.r_string.extend(['d'])`,
			want: []string{"a", "b", "c", "d"},
		},
		{
			src:     `msg.r_string.extend([None])`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `string'.",
		},
		{
			src:     `msg.r_submsg.extend([None])`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `skycfg.test_proto.MessageV2'.",
		},
	}
	for _, test := range tests {
		msg := &pb.MessageV2{
			RString: []string{"a", "b", "c"},
		}
		globals := starlark.StringDict{
			"msg": NewSkyProtoMessage(msg),
		}
		_, err := starlark.Eval(&starlark.Thread{}, "", test.src, globals)
		if test.wantErr != "" {
			if err == nil {
				t.Errorf("eval(%q): expected error", test.src)
			} else if test.wantErr != err.Error() {
				t.Errorf("eval(%q): expected error %q, got %q", test.src, test.wantErr, err.Error())
			}
		} else if err != nil {
			t.Errorf("eval(%q): %v", test.src, err)
			continue
		} else if !reflect.DeepEqual(msg.RString, test.want) {
			t.Errorf("eval(%q): expected msg.r_string = %v, got %v", test.src, test.want, msg.RString)
		}
	}
}

func TestMapMutation(t *testing.T) {
	tests := []struct {
		src     string
		want    map[string]string
		wantErr string
	}{
		{
			src:  `msg.map_string.clear()`,
			want: map[string]string{},
		},
		{
			src: `msg.map_string.setdefault('a', 'Z')`,
			want: map[string]string{
				"a": "A",
				"b": "B",
				"c": "C",
			},
		},
		{
			src: `msg.map_string.setdefault('d', 'D')`,
			want: map[string]string{
				"a": "A",
				"b": "B",
				"c": "C",
				"d": "D",
			},
		},
		{
			src:     `msg.map_string.setdefault('d', None)`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `string'.",
		},
		{
			src:     `msg.map_submsg.setdefault('d', None)`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `skycfg.test_proto.MessageV2'.",
		},
		{
			src: `msg.map_string.update({'a': 'Z', 'd': 'D'})`,
			want: map[string]string{
				"a": "Z",
				"b": "B",
				"c": "C",
				"d": "D",
			},
		},
		{
			src:     `msg.map_string.update({'a': None})`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `string'.",
		},
		{
			src:     `msg.map_submsg.update({'a': None})`,
			wantErr: "TypeError: value None (type `NoneType') can't be assigned to type `skycfg.test_proto.MessageV2'.",
		},
	}
	for _, test := range tests {
		msg := &pb.MessageV2{
			MapString: map[string]string{
				"a": "A",
				"b": "B",
				"c": "C",
			},
		}
		globals := starlark.StringDict{
			"msg": NewSkyProtoMessage(msg),
		}
		_, err := starlark.Eval(&starlark.Thread{}, "", test.src, globals)
		if test.wantErr != "" {
			if err == nil {
				t.Errorf("eval(%q): expected error", test.src)
			} else if test.wantErr != err.Error() {
				t.Errorf("eval(%q): expected error %q, got %q", test.src, test.wantErr, err.Error())
			}
		} else if err != nil {
			t.Errorf("eval(%q): %v", test.src, err)
			continue
		} else if !reflect.DeepEqual(msg.MapString, test.want) {
			t.Errorf("eval(%q): expected msg.r_string = %v, got %v", test.src, test.want, msg.MapString)
		}
	}
}

func TestUnsetProto2Fields(t *testing.T) {
	// Proto v2 distinguishes between unset and set-to-empty.
	msg := skyEval(t, `proto.package("skycfg.test_proto").MessageV2(
		f_string = None,
	)`)
	got := msg.String()
	want := `<skycfg.test_proto.MessageV2 >`
	if want != got {
		t.Fatalf("wanted %q, got %q", want, got)
	}

	field, err := msg.(starlark.HasAttrs).Attr("f_string")
	if err != nil {
		t.Fatalf(`msg.Attr("f_string"): %v`, err)
	}
	if _, isNone := field.(starlark.NoneType); !isNone {
		t.Fatalf("field set to None should be returned as None")
	}
}

func TestProtoFromText(t *testing.T) {
	val := skyEval(t, `proto.from_text(proto.package("skycfg.test_proto").MessageV3, "f_int32: 1010").f_int32`)
	got := val.String()
	want := "1010"
	if want != got {
		t.Fatalf("from_text: wanted %q, got %q", want, got)
	}
}

func TestProtoFromJson(t *testing.T) {
	val := skyEval(t, `proto.from_json(proto.package("skycfg.test_proto").MessageV3, "{\"f_int32\": 1010}").f_int32`)
	got := val.String()
	want := "1010"
	if want != got {
		t.Fatalf("from_json: wanted %q, got %q", want, got)
	}
}

func TestProtoFromYaml(t *testing.T) {
	val := skyEval(t, `proto.from_yaml(proto.package("skycfg.test_proto").MessageV3, "f_int32: 1010").f_int32`)
	got := val.String()
	want := "1010"
	if want != got {
		t.Fatalf("from_yaml: wanted %q, got %q", want, got)
	}
}

func TestProtoComparisonEqual(t *testing.T) {
	msg := &pb.MessageV2{
		RString: []string{"a", "b", "c"},
	}
	skyMsg := NewSkyProtoMessage(msg)

	// create a separate msg to ensure the underlying reference in skyMsgOther is different
	msgOther := &pb.MessageV2{
		RString: []string{"a", "b", "c"},
	}
	skyMsgOther := NewSkyProtoMessage(msgOther)
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
	skyMsg := NewSkyProtoMessage(msg)

	// create a separate msg to ensure the underlying reference in skyMsgOther is different
	msgOther := &pb.MessageV2{
		RString: []string{"a", "b"},
	}
	skyMsgOther := NewSkyProtoMessage(msgOther)
	ok, err := starlark.Compare(syntax.EQL, skyMsg, skyMsgOther)
	if err != nil {
		t.Error(err)
	}
	if ok {
		t.Error("Expected protos to not be equal")
	}
}
