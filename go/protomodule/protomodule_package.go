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
	"fmt"
	"sort"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func starlarkPackageFn(registry *protoregistry.Types, filesRegistry *protoregistry.Files) starlark.Callable {
	return starlark.NewBuiltin("proto.package", func(
		t *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		var rawPackageName string
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &rawPackageName); err != nil {
			return nil, err
		}
		packageName := protoreflect.FullName(rawPackageName)
		if !packageName.IsValid() {
			return nil, fmt.Errorf("invalid Protobuf package name %q", packageName)
		}
		return NewProtoPackage(registry, filesRegistry, packageName), nil
	})
}

type protoPackage struct {
	name     protoreflect.FullName
	registry *protoregistry.Types
	attrs    starlark.StringDict
}

func NewProtoPackage(
	registry *protoregistry.Types,
	filesRegistry *protoregistry.Files,
	packageName protoreflect.FullName,
) *protoPackage {
	attrs := make(starlark.StringDict)

	registry.RangeEnums(func(t protoreflect.EnumType) bool {
		desc := t.Descriptor()
		name := desc.Name()
		if packageName.Append(name) == desc.FullName() {
			attrs[string(name)] = newEnumType(desc)
		}
		return true
	})

	registry.RangeMessages(func(t protoreflect.MessageType) bool {
		desc := t.Descriptor()
		name := desc.Name()
		if packageName.Append(name) == desc.FullName() {
			msgType := newMessageType(registry, t.New().Interface())
			attrs[string(name)] = msgType
		}
		return true
	})

	// TODO:: Iterate over the Services
	filesRegistry.RangeFilesByPackage(packageName, func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			sd := fd.Services().Get(i)
			// attrs[] = nil
			for j := 0; j < sd.Methods().Len(); j++ {
				svc := sd.Methods().Get(j)
				attrs[fmt.Sprintf("%s.%s", sd.Name(), svc.Name())] = newService(svc)
			}
		}

		return true
	})

	return &protoPackage{
		name:     packageName,
		registry: registry,
		attrs:    attrs,
	}
}

//	Type Callable interface {
//		Value
//		Name() string
//		CallInternal(thread *Thread, args Tuple, kwargs []Tuple) (Value, error)
//	}
func newService(svc protoreflect.MethodDescriptor) starlark.Callable {
	return protoService{}
}

type protoService struct {
}

func (pkg *protoPackage) String() string       { return fmt.Sprintf("<proto.Package %q>", pkg.name) }
func (pkg *protoPackage) Type() string         { return "proto.package" }
func (pkg *protoPackage) Freeze()              {}
func (pkg *protoPackage) Truth() starlark.Bool { return starlark.True }
func (pkg *protoPackage) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", pkg.Type())
}

func (pkg *protoPackage) AttrNames() []string {
	names := make([]string, 0, len(pkg.attrs))
	for name := range pkg.attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (pkg *protoPackage) Attr(attrName string) (starlark.Value, error) {
	if attr, ok := pkg.attrs[attrName]; ok {
		return attr, nil
	}
	fullName := pkg.name.Append(protoreflect.Name(attrName))
	return nil, fmt.Errorf("Protobuf type %q not found", fullName)
}
