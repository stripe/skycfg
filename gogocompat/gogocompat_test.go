// Copyright 2019 The Skycfg Authors.
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

// Package gogocompat is a compatibility shim for GoGo.
package gogocompat

import (
	"testing"

	_ "github.com/gogo/protobuf/types"
	_ "github.com/golang/protobuf/ptypes/wrappers"
	"go.starlark.net/starlark"

	impl "github.com/stripe/skycfg/internal/go/skycfg"
)

func skyEval(t *testing.T, src string) starlark.Value {
	t.Helper()
	globals := starlark.StringDict{
		"proto": impl.NewProtoModule(ProtoRegistry()),
	}
	val, err := starlark.Eval(&starlark.Thread{}, "", src, globals)
	if err != nil {
		t.Fatalf("eval(%q): %v", src, err)
	}
	return val
}

func TestGogoRegistry(t *testing.T) {
	protoTimestamp := skyEval(t, `proto.package("google.protobuf").Timestamp()`)
	gogoTimestamp := skyEval(t, `proto.package("gogo:google.protobuf").Timestamp()`)

	protoType := protoTimestamp.Type()
	gogoType := gogoTimestamp.Type()
	if protoType != gogoType {
		t.Fatalf("Expected same type name for Protobuf and GoGo, got proto=%q, gogo=%q)", protoType, gogoType)
	}
}
