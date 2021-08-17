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

	"go.starlark.net/starlark"
)

// Implements proto.merge merging src into dst, returning merged value
func mergeField(dst, src starlark.Value) (starlark.Value, error) {
	if dst == nil {
		return src, nil
	}
	if src == nil {
		return dst, nil
	}

	if dst.Type() != src.Type() {
		return nil, mergeError(dst, src)
	}

	switch dst := dst.(type) {
	case *protoRepeated:
		if src, ok := src.(*protoRepeated); ok {
			newList := newProtoRepeated(dst.fieldDesc)

			err := newList.Extend(dst)
			if err != nil {
				return nil, err
			}

			err = newList.Extend(src)
			if err != nil {
				return nil, err
			}

			return newList, nil
		}
		return nil, mergeError(dst, src)
	case *protoMap:
		if src, ok := src.(*protoMap); ok {
			newMap := newProtoMap(dst.mapKey, dst.mapValue)

			for _, item := range dst.Items() {
				err := newMap.SetKey(item[0], item[1])
				if err != nil {
					return nil, err
				}
			}

			for _, item := range src.Items() {
				err := newMap.SetKey(item[0], item[1])
				if err != nil {
					return nil, err
				}
			}

			return newMap, nil
		}
		return nil, mergeError(dst, src)
	case *protoMessage:
		if src, ok := src.(*protoMessage); ok {
			newMessage, err := NewMessage(dst.msg)
			if err != nil {
				return nil, err
			}

			newMessage.Merge(dst)
			newMessage.Merge(src)

			return newMessage, nil
		}
		return nil, mergeError(dst, src)
	default:
		return src, nil
	}
}

func mergeError(dst, src starlark.Value) error {
	return fmt.Errorf("MergeError: Cannot merge protobufs of different types: Merge(%s, %s)", dst.Type(), src.Type())
}
