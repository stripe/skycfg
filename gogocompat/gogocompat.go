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
	"reflect"
	"strings"

	gogo_proto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/proto"

	impl "github.com/stripe/skycfg/internal/go/skycfg"
)

type unstableProtoRegistry interface {
	impl.ProtoRegistry
}

type protoRegistry struct{}

func (*protoRegistry) UnstableProtoMessageType(name string) (reflect.Type, error) {
	if t := proto.MessageType(name); t != nil {
		return t, nil
	}
	name = strings.TrimPrefix(name, "gogo:")
	if t := gogo_proto.MessageType(name); t != nil {
		return t, nil
	}
	return nil, nil
}

func (*protoRegistry) UnstableEnumValueMap(name string) map[string]int32 {
	if ev := proto.EnumValueMap(name); ev != nil {
		return ev
	}
	if ev := gogo_proto.EnumValueMap(name); ev != nil {
		return ev
	}
	return nil
}

// ProtoRegistry returns a Protobuf message registry that falls back to GoGo.
//
// To support types that might differ between Protobuf and GoGo registrations,
// the special prefix "gogo:" can be used to skip looking up messages in the
// standard Protobuf registry.
//
//  pb = proto.package("google.protobuf")
//  gogo_pb = proto.package("gogo:google.protobuf")
//  # pb.Timestamp and gogo_pb.Timestamp are distinct types.
//
// The exact type of the return value is not yet stabilized, but the result
// is guaranteed to be accepted by the skycfg.WithProtoRegistry() load option.
func ProtoRegistry() unstableProtoRegistry {
	return &protoRegistry{}
}
