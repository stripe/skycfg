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

// type_conversions.go provides protomodule-to-starlark and
// starlark-to-protomodule conversions
package protomodule

import (
	"fmt"
	"math"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func valueFromStarlark(msg protoreflect.Message, fieldDesc protoreflect.FieldDescriptor, val starlark.Value) (protoreflect.Value, error) {
	if fieldDesc.IsList() {
		if list, ok := val.(*protoRepeated); ok {
			protoListValue := msg.New().NewField(fieldDesc)
			protoList := protoListValue.List()
			for i := 0; i < list.Len(); i++ {
				v, err := scalarValueFromStarlark(fieldDesc, list.Index(i))
				if err != nil {
					return protoreflect.Value{}, err
				}
				protoList.Append(v)
			}

			return protoListValue, nil
		}

		return protoreflect.Value{}, typeError(fieldDesc, val, false)
	} else if fieldDesc.IsMap() {
		if mapVal, ok := val.(*protoMap); ok {
			protoMapValue := msg.New().NewField(fieldDesc)
			protoMap := protoMapValue.Map()
			for _, item := range mapVal.Items() {
				protoK, err := scalarValueFromStarlark(fieldDesc.MapKey(), item[0])
				if err != nil {
					return protoreflect.Value{}, err
				}

				protoV, err := scalarValueFromStarlark(fieldDesc.MapValue(), item[1])
				if err != nil {
					return protoreflect.Value{}, err
				}

				protoMap.Set(protoreflect.MapKey(protoK), protoV)
			}

			return protoMapValue, nil
		}
		return protoreflect.Value{}, typeError(fieldDesc, val, false)
	}

	return scalarValueFromStarlark(fieldDesc, val)
}

func scalarValueFromStarlark(fieldDesc protoreflect.FieldDescriptor, val starlark.Value) (protoreflect.Value, error) {
	k := fieldDesc.Kind()
	switch k {
	case protoreflect.BoolKind:
		if val, ok := val.(starlark.Bool); ok {
			return protoreflect.ValueOf(bool(val)), nil
		}
	case protoreflect.StringKind:
		if val, ok := val.(starlark.String); ok {
			return protoreflect.ValueOf(string(val)), nil
		}
	case protoreflect.DoubleKind:
		if val, ok := starlark.AsFloat(val); ok {
			return protoreflect.ValueOf(val), nil
		}
	case protoreflect.FloatKind:
		if val, ok := starlark.AsFloat(val); ok {
			return protoreflect.ValueOf(float32(val)), nil
		}
	case protoreflect.Int64Kind:
		if valInt, ok := val.(starlark.Int); ok {
			if val, ok := valInt.Int64(); ok {
				return protoreflect.ValueOf(val), nil
			}
			return protoreflect.Value{}, fmt.Errorf("ValueError: value %v overflows type \"int64\".", valInt)
		}
	case protoreflect.Uint64Kind:
		if valInt, ok := val.(starlark.Int); ok {
			if val, ok := valInt.Uint64(); ok {
				return protoreflect.ValueOf(val), nil
			}
			return protoreflect.Value{}, fmt.Errorf("ValueError: value %v overflows type \"uint64\".", valInt)
		}
	case protoreflect.Int32Kind:
		if valInt, ok := val.(starlark.Int); ok {
			if val, ok := valInt.Int64(); ok && val >= math.MinInt32 && val <= math.MaxInt32 {
				return protoreflect.ValueOf(int32(val)), nil
			}
			return protoreflect.Value{}, fmt.Errorf("ValueError: value %v overflows type \"int32\".", valInt)
		}
	case protoreflect.Uint32Kind:
		if valInt, ok := val.(starlark.Int); ok {
			if val, ok := valInt.Uint64(); ok && val <= math.MaxUint32 {
				return protoreflect.ValueOf(uint32(val)), nil
			}
			return protoreflect.Value{}, fmt.Errorf("ValueError: value %v overflows type \"uint32\".", valInt)
		}
	case protoreflect.MessageKind:
		return protoreflect.Value{}, fmt.Errorf("MessageKind: Unimplemented")
	case protoreflect.EnumKind:
		if enum, ok := val.(*protoEnumValue); ok {
			return protoreflect.ValueOf(enum.enumNumber()), nil
		}
	case protoreflect.BytesKind:
		if valString, ok := val.(starlark.String); ok {
			return protoreflect.ValueOf([]byte(valString)), nil
		}
	}

	return protoreflect.Value{}, typeError(fieldDesc, val, true)
}

// Wrap a protobuf field value as a starlark.Value
func valueToStarlark(val protoreflect.Value, fieldDesc protoreflect.FieldDescriptor) (starlark.Value, error) {
	if fieldDesc.IsList() {
		if listVal, ok := val.Interface().(protoreflect.List); ok {
			out := newProtoRepeated(fieldDesc)
			for i := 0; i < listVal.Len(); i++ {
				starlarkValue, err := scalarValueToStarlark(listVal.Get(i), fieldDesc)
				if err != nil {
					return starlark.None, err
				}
				out.Append(starlarkValue)
			}
			return out, nil
		} else if val.Interface() == nil {
			return newProtoRepeated(fieldDesc), nil
		}
		return starlark.None, fmt.Errorf("TypeError: cannot convert %T into list", val.Interface())
	} else if fieldDesc.IsMap() {
		if mapVal, ok := val.Interface().(protoreflect.Map); ok {
			out := newProtoMap(fieldDesc.MapKey(), fieldDesc.MapValue())
			var rangeErr error
			mapVal.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
				starlarkKey, err := scalarValueToStarlark(protoreflect.Value(k), fieldDesc.MapKey())
				if err != nil {
					rangeErr = err
					return false
				}

				starlarkValue, err := scalarValueToStarlark(v, fieldDesc.MapValue())
				if err != nil {
					rangeErr = err
					return false
				}

				out.SetKey(starlarkKey, starlarkValue)
				return true
			})
			if rangeErr != nil {
				return starlark.None, rangeErr
			}

			return out, nil
		} else if val.Interface() == nil {
			return newProtoMap(fieldDesc.MapKey(), fieldDesc.MapValue()), nil
		}
		return starlark.None, fmt.Errorf("TypeError: cannot convert %T into map", val.Interface())
	}

	return scalarValueToStarlark(val, fieldDesc)
}

func scalarValueToStarlark(val protoreflect.Value, fieldDesc protoreflect.FieldDescriptor) (starlark.Value, error) {
	switch fieldDesc.Kind() {
	case protoreflect.BoolKind:
		return starlark.Bool(val.Bool()), nil
	case protoreflect.Int32Kind:
		return starlark.MakeInt64(val.Int()), nil
	case protoreflect.Int64Kind:
		return starlark.MakeInt64(val.Int()), nil
	case protoreflect.Uint32Kind:
		return starlark.MakeUint64(val.Uint()), nil
	case protoreflect.Uint64Kind:
		return starlark.MakeUint64(val.Uint()), nil
	case protoreflect.FloatKind:
		return starlark.Float(val.Float()), nil
	case protoreflect.DoubleKind:
		return starlark.Float(val.Float()), nil
	case protoreflect.StringKind:
		return starlark.String(val.String()), nil
	case protoreflect.BytesKind:
		// Handle []byte ([]uint8) -> string special case.
		return starlark.String(val.Bytes()), nil
	case protoreflect.MessageKind:
		return nil, fmt.Errorf("MessageKind: Unimplemented")
	}

	return starlark.None, fmt.Errorf("valueToStarlark: Value unuspported: %s\n", string(fieldDesc.FullName()))
}

// Verify v can act as fieldDesc
func scalarTypeCheck(fieldDesc protoreflect.FieldDescriptor, v starlark.Value) error {
	_, err := scalarValueFromStarlark(fieldDesc, v)
	return err
}

func typeError(fieldDesc protoreflect.FieldDescriptor, val starlark.Value, scalar bool) error {
	expectedType := typeName(fieldDesc)

	// FieldDescriptor has the same typeName for []string and string
	// and typeError needs to distinguish setting a []string = int versus
	// appending a value in []string
	if !scalar {
		if fieldDesc.IsList() {
			expectedType = fmt.Sprintf("[]%s", typeName(fieldDesc))
		} else if fieldDesc.IsMap() {
			expectedType = fmt.Sprintf("map[%s]%s", typeName(fieldDesc.MapKey()), typeName(fieldDesc.MapValue()))
		}
	}

	return fmt.Errorf("TypeError: value %s (type %q) can't be assigned to type %q.",
		val.String(), val.Type(), expectedType,
	)
}

// Returns a type name for a descriptor, ignoring list/map qualifiers
func typeName(fieldDesc protoreflect.FieldDescriptor) string {
	k := fieldDesc.Kind()
	switch k {
	case protoreflect.EnumKind:
		return string(fieldDesc.Enum().FullName())
	case protoreflect.MessageKind:
		return string(fieldDesc.Message().FullName())
	default:
		return k.String()
	}
}
