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
	"errors"
	"testing"

	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/protobuf/reflect/protoregistry"

	pb "github.com/stripe/skycfg/internal/testdata/test_proto"
)

func init() {
	resolve.AllowFloat = true
}

func newRegistry() *protoregistry.Types {
	registry := &protoregistry.Types{}
	registry.RegisterMessage((&pb.MessageV2{}).ProtoReflect().Type())
	registry.RegisterMessage((&pb.MessageV3{}).ProtoReflect().Type())
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
		"pb": newProtoPackage(newRegistry(), "skycfg.test_proto"),
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
		"pb": newProtoPackage(newRegistry(), "skycfg.test_proto"),
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

func checkError(got, want error) bool {
	if got == nil {
		return false
	}
	return got.Error() == want.Error()
}
