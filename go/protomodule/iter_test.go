// Copyright 2025 The Skycfg Authors.
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

//go:build go1.23

package protomodule

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"go.starlark.net/starlark"

	pb "github.com/stripe/skycfg/internal/test_proto"
)

func TestListType_Elements(t *testing.T) {
	msg := (&pb.MessageV3{}).ProtoReflect().Descriptor()
	listFieldDesc := msg.Fields().ByName("r_string")

	rep := newProtoRepeated(listFieldDesc)
	rep.Append(starlark.String("a"))
	rep.Append(starlark.String("b"))

	got := slices.Collect(rep.Elements())
	want := []starlark.Value{starlark.String("a"), starlark.String("b")}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("wanted: %v\ngot   : %v", want, got)
	}
}

func TestMapType_Entries(t *testing.T) {
	msg := (&pb.MessageV3{}).ProtoReflect().Descriptor()
	mapFieldDesc := msg.Fields().ByName("map_string")

	pmap := newProtoMap(mapFieldDesc.MapKey(), mapFieldDesc.MapValue())
	pmap.SetKey(starlark.String("a"), starlark.String("1"))
	pmap.SetKey(starlark.String("b"), starlark.String("2"))

	got := maps.Collect(pmap.Entries())
	want := map[starlark.Value]starlark.Value{
		starlark.String("a"): starlark.String("1"),
		starlark.String("b"): starlark.String("2"),
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("wanted: %v\ngot   : %v", want, got)
	}
}
