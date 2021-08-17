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

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var allowedListMethods = map[string]func(*protoRepeated) starlark.Value{
	"clear":  nil,
	"append": (*protoRepeated).wrapAppend,
	"extend": (*protoRepeated).wrapExtend,
}

// protoRepeated wraps an underlying starlark.List to provide typechecking on
// writes
//
// starlark.List is heterogeneous, where protoRepeated enforces all values
// conform to the given fieldDesc
type protoRepeated struct {
	fieldDesc protoreflect.FieldDescriptor
	list      *starlark.List
}

var _ starlark.Value = (*protoRepeated)(nil)
var _ starlark.Iterable = (*protoRepeated)(nil)
var _ starlark.Sequence = (*protoRepeated)(nil)
var _ starlark.Indexable = (*protoRepeated)(nil)
var _ starlark.HasAttrs = (*protoRepeated)(nil)
var _ starlark.HasSetIndex = (*protoRepeated)(nil)
var _ starlark.HasBinary = (*protoRepeated)(nil)
var _ starlark.Comparable = (*protoRepeated)(nil)

func newProtoRepeated(fieldDesc protoreflect.FieldDescriptor) *protoRepeated {
	return &protoRepeated{fieldDesc, starlark.NewList(nil)}
}

func newProtoRepeatedFromList(fieldDesc protoreflect.FieldDescriptor, l *starlark.List) (*protoRepeated, error) {
	out := &protoRepeated{fieldDesc, l}
	for i := 0; i < l.Len(); i++ {
		err := scalarTypeCheck(fieldDesc, l.Index(i))
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *protoRepeated) Attr(name string) (starlark.Value, error) {
	wrapper, ok := allowedListMethods[name]
	if !ok {
		return nil, nil
	}
	if wrapper != nil {
		return wrapper(r), nil
	}
	return r.list.Attr(name)
}

func (r *protoRepeated) AttrNames() []string                 { return r.list.AttrNames() }
func (r *protoRepeated) Freeze()                             { r.list.Freeze() }
func (r *protoRepeated) Hash() (uint32, error)               { return r.list.Hash() }
func (r *protoRepeated) Index(i int) starlark.Value          { return r.list.Index(i) }
func (r *protoRepeated) Iterate() starlark.Iterator          { return r.list.Iterate() }
func (r *protoRepeated) Len() int                            { return r.list.Len() }
func (r *protoRepeated) Slice(x, y, step int) starlark.Value { return r.list.Slice(x, y, step) }
func (r *protoRepeated) String() string                      { return r.list.String() }
func (r *protoRepeated) Truth() starlark.Bool                { return r.list.Truth() }

func (r *protoRepeated) Type() string {
	return fmt.Sprintf("list<%s>", typeName(r.fieldDesc))
}

func (r *protoRepeated) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	other, ok := y.(*protoRepeated)
	if !ok {
		return false, nil
	}

	return starlark.CompareDepth(op, r.list, other.list, depth)
}

func (r *protoRepeated) Append(v starlark.Value) error {
	err := scalarTypeCheck(r.fieldDesc, v)
	if err != nil {
		return err
	}

	return r.list.Append(v)
}

func (r *protoRepeated) SetIndex(i int, v starlark.Value) error {
	err := scalarTypeCheck(r.fieldDesc, v)
	if err != nil {
		return err
	}

	return r.list.SetIndex(i, v)
}

func (r *protoRepeated) Extend(iterable starlark.Iterable) error {
	iter := iterable.Iterate()
	defer iter.Done()

	var val starlark.Value
	for iter.Next(&val) {
		err := r.Append(val)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *protoRepeated) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	if op == syntax.PLUS {
		if side == starlark.Left {
			switch y := y.(type) {
			case *starlark.List:
				return starlark.Binary(op, r.list, y)
			case *protoRepeated:
				return starlark.Binary(op, r.list, y.list)
			}
			return nil, nil
		}
		if side == starlark.Right {
			if _, ok := y.(*starlark.List); ok {
				return starlark.Binary(op, y, r.list)
			}
			return nil, nil
		}
	}
	return nil, nil
}

func (r *protoRepeated) wrapAppend() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var val starlark.Value
		if err := starlark.UnpackPositionalArgs("append", args, kwargs, 1, &val); err != nil {
			return nil, err
		}
		if err := r.Append(val); err != nil {
			return nil, err
		}
		return starlark.None, nil
	}
	return starlark.NewBuiltin("append", impl).BindReceiver(r)
}

func (r *protoRepeated) wrapExtend() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var val starlark.Iterable
		if err := starlark.UnpackPositionalArgs("extend", args, kwargs, 1, &val); err != nil {
			return nil, err
		}
		if err := r.Extend(val); err != nil {
			return nil, err
		}
		return starlark.None, nil
	}
	return starlark.NewBuiltin("extend", impl).BindReceiver(r)
}
