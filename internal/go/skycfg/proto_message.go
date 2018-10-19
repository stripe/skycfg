package skycfg

import (
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/google/skylark"
	"github.com/google/skylark/syntax"
)

// A Skylark built-in type representing a Protobuf message. Provides attributes
// for accessing message fields using their original protobuf names.
type skyProtoMessage struct {
	msg    proto.Message
	val    reflect.Value
	fields []*proto.Properties
	oneofs map[string]*proto.OneofProperties
	frozen bool

	// lets the message wrapper keep track of per-field wrappers, for freezing.
	attrCache map[string]skylark.Value
}

var _ skylark.HasAttrs = (*skyProtoMessage)(nil)
var _ skylark.HasSetField = (*skyProtoMessage)(nil)

func (msg *skyProtoMessage) String() string {
	return fmt.Sprintf("<%s %s>", msg.Type(), proto.CompactTextString(msg.msg))
}
func (msg *skyProtoMessage) Type() string        { return messageTypeName(msg.msg) }
func (msg *skyProtoMessage) Truth() skylark.Bool { return skylark.True }

func (msg *skyProtoMessage) Freeze() {
	if !msg.frozen {
		msg.frozen = true
		for _, attr := range msg.attrCache {
			attr.Freeze()
		}
	}
}

func (msg *skyProtoMessage) Hash() (uint32, error) {
	return 0, fmt.Errorf("skyProtoMessage.Hash: TODO")
}

func NewSkyProtoMessage(msg proto.Message) *skyProtoMessage {
	wrapper := &skyProtoMessage{
		msg:       msg,
		val:       reflect.ValueOf(msg).Elem(),
		oneofs:    make(map[string]*proto.OneofProperties),
		attrCache: make(map[string]skylark.Value),
	}
	for fieldName, prop := range proto.GetProperties(wrapper.val.Type()).OneofTypes {
		wrapper.fields = append(wrapper.fields, prop.Prop)
		wrapper.oneofs[fieldName] = prop
	}

	for _, prop := range proto.GetProperties(wrapper.val.Type()).Prop {
		if prop.Tag == 0 {
			// Skip attributes that don't correspond to a protobuf field.
			continue
		}
		wrapper.fields = append(wrapper.fields, prop)
	}
	return wrapper
}

func ToProtoMessage(val skylark.Value) (proto.Message, bool) {
	if msg, ok := val.(*skyProtoMessage); ok {
		return msg.msg, true
	}
	return nil, false
}

func (msg *skyProtoMessage) checkMutable(verb string) error {
	if msg.frozen {
		return fmt.Errorf("cannot %s frozen message", verb)
	}
	return nil
}

func (msg *skyProtoMessage) Attr(name string) (skylark.Value, error) {
	if attr, ok := msg.attrCache[name]; ok {
		return attr, nil
	}
	for _, field := range msg.fields {
		if field.OrigName != name {
			continue
		}
		var out skylark.Value
		if oneofProp, isOneof := msg.oneofs[name]; isOneof {
			out = msg.getOneofField(name, oneofProp)
		} else {
			out = valueToSkylark(msg.val.FieldByName(field.Name))
		}
		if msg.frozen {
			out.Freeze()
		}
		msg.attrCache[name] = out
		return out, nil
	}
	return nil, nil
}

func (msg *skyProtoMessage) getOneofField(name string, prop *proto.OneofProperties) skylark.Value {
	ifaceField := msg.val.Field(prop.Field)
	if ifaceField.IsNil() {
		return skylark.None
	}
	if ifaceField.Elem().Type() == prop.Type {
		return valueToSkylark(ifaceField.Elem().Elem().Field(0))
	}
	return skylark.None
}

func (msg *skyProtoMessage) AttrNames() []string {
	var names []string
	for _, field := range msg.fields {
		names = append(names, field.OrigName)
	}
	sort.Strings(names)
	return names
}

func (msg *skyProtoMessage) SetField(name string, sky skylark.Value) error {
	var prop *proto.Properties
	for _, fieldProp := range msg.fields {
		if name != fieldProp.OrigName {
			continue
		}
		prop = fieldProp
		break
	}
	if prop == nil {
		return fmt.Errorf("AttributeError: `%s' value has no field %q", msg.Type(), name)
	}
	if oneofProp, isOneof := msg.oneofs[name]; isOneof {
		return msg.setOneofField(name, oneofProp, sky)
	}
	return msg.setSingleField(name, prop, sky)
}

func (msg *skyProtoMessage) setOneofField(name string, prop *proto.OneofProperties, sky skylark.Value) error {
	// Oneofs are stored in a two-part format, where `msg.val` has a field of an intermediate interface
	// type that can be constructed from the property type.
	ifaceField := msg.val.Field(prop.Field)

	field, ok := prop.Type.Elem().FieldByName(prop.Prop.Name)
	if !ok {
		return fmt.Errorf("InternalError: field %q not found in generated type %v", name, prop.Type)
	}
	val, err := valueFromSkylark(field.Type, sky)
	if err != nil {
		return err
	}
	if err := msg.checkMutable("set field of"); err != nil {
		return err
	}

	// Construct the intermediate per-field struct.
	box := reflect.New(prop.Type.Elem())
	box.Elem().Field(0).Set(val)

	delete(msg.attrCache, name)
	ifaceField.Set(box)
	return nil
}

func (msg *skyProtoMessage) setSingleField(name string, prop *proto.Properties, sky skylark.Value) error {
	field, ok := msg.val.Type().FieldByName(prop.Name)
	if !ok {
		return fmt.Errorf("InternalError: field %q not found in generated type %v", prop.OrigName, msg.val.Type())
	}

	val, err := valueFromSkylark(field.Type, sky)
	if err != nil {
		return err
	}
	if err := msg.checkMutable("set field of"); err != nil {
		return err
	}
	delete(msg.attrCache, name)
	msg.val.FieldByName(prop.Name).Set(val)
	return nil
}

func valueToSkylark(val reflect.Value) skylark.Value {
	if scalar := scalarToSkylark(val); scalar != nil {
		return scalar
	}
	iface := val.Interface()
	if msg, ok := iface.(proto.Message); ok {
		return NewSkyProtoMessage(msg)
	}
	if val.Type().Kind() == reflect.Struct {
		// Might have been generated by gogo-protobuf
		//
		// Need to check if this is a non-pointer map value, which
		// cannot be addressed and therefore can never become a
		// `proto.Message`.
		if val.CanAddr() {
			if msg, ok := val.Addr().Interface().(proto.Message); ok {
				return NewSkyProtoMessage(msg)
			}
		}
	}
	if val.Type().Kind() == reflect.Slice {
		var items []skylark.Value
		for ii := 0; ii < val.Len(); ii++ {
			items = append(items, valueToSkylark(val.Index(ii)))
		}
		return &protoRepeated{
			field: val,
			list:  skylark.NewList(items),
		}
	}
	if val.Type().Kind() == reflect.Map {
		dict := &skylark.Dict{}
		for _, keyVal := range val.MapKeys() {
			elemVal := val.MapIndex(keyVal)
			key := valueToSkylark(keyVal)
			elem := valueToSkylark(elemVal)
			if err := dict.Set(key, elem); err != nil {
				panic(fmt.Sprintf("dict.Set(%s, %s): %v", key, elem, err))
			}
		}
		return &protoMap{
			field: val,
			dict:  dict,
		}
	}
	// This should be impossible, because the set of types present
	// in a generated protobuf struct is small and limited.
	panic(fmt.Errorf("valueToSkylark: unknown type %v", val.Type()))
}

func scalarToSkylark(val reflect.Value) skylark.Value {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return skylark.None
		}
		val = val.Elem()
	}
	iface := val.Interface()
	switch f := iface.(type) {
	case int32:
		return skylark.MakeInt64(int64(f))
	case int64:
		return skylark.MakeInt64(f)
	case uint32:
		return skylark.MakeUint64(uint64(f))
	case uint64:
		return skylark.MakeUint64(f)
	case float32:
		return skylark.Float(f)
	case float64:
		return skylark.Float(f)
	case string:
		return skylark.String(f)
	case bool:
		return skylark.Bool(f)
	}
	if enum, ok := iface.(protoEnum); ok {
		return &skyProtoEnumValue{
			typeName:  enumTypeName(enum),
			valueName: enum.String(),
			value:     val.Convert(reflect.TypeOf(int32(0))).Interface().(int32),
		}
	}
	return nil
}

func valueFromSkylark(t reflect.Type, sky skylark.Value) (reflect.Value, error) {
	switch sky := sky.(type) {
	case skylark.Int:
		return scalarFromSkylark(t, sky)
	case skylark.Float:
		return scalarFromSkylark(t, sky)
	case skylark.String:
		return scalarFromSkylark(t, sky)
	case skylark.Bool:
		return scalarFromSkylark(t, sky)
	case skylark.NoneType:
		if t.Kind() == reflect.Ptr {
			return reflect.Zero(t), nil
		}
		// Give a better error message for true type mismatch, instead of
		// "pointer or non-pointer" caused by Go's different representation
		// of proto3 messages.
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
			return reflect.Value{}, typeError(t, sky)
		}
		return reflect.Value{}, fmt.Errorf("TypeError: value None can't be assigned to type `%s' in proto3 mode.", t)
	case *skyProtoEnumValue:
		return enumFromSkylark(t, sky)
	case *skyProtoMessage:
		if reflect.TypeOf(sky.msg) == t {
			val := reflect.New(t.Elem())
			val.Elem().Set(reflect.ValueOf(sky.msg).Elem())
			return val, nil
		}
		if reflect.TypeOf(sky.msg) == reflect.PtrTo(t) {
			val := reflect.New(t)
			val.Elem().Set(reflect.ValueOf(sky.msg).Elem())
			return val.Elem(), nil
		}
	case *protoRepeated:
		return valueFromSkylark(t, sky.list)
	case *skylark.List:
		if t.Kind() == reflect.Slice {
			elemType := t.Elem()
			val := reflect.MakeSlice(t, sky.Len(), sky.Len())
			for ii := 0; ii < sky.Len(); ii++ {
				elem, err := valueFromSkylark(elemType, sky.Index(ii))
				if err != nil {
					return reflect.Value{}, err
				}
				val.Index(ii).Set(elem)
			}
			return val, nil
		}
	case *protoMap:
		return valueFromSkylark(t, sky.dict)
	case *skylark.Dict:
		if t.Kind() == reflect.Map {
			keyType := t.Key()
			elemType := t.Elem()
			val := reflect.MakeMapWithSize(t, sky.Len())
			for _, item := range sky.Items() {
				key, err := valueFromSkylark(keyType, item[0])
				if err != nil {
					return reflect.Value{}, err
				}
				elem, err := valueFromSkylark(elemType, item[1])
				if err != nil {
					return reflect.Value{}, err
				}
				val.SetMapIndex(key, elem)
			}
			return val, nil
		}
	}
	return reflect.Value{}, typeError(t, sky)
}

func scalarFromSkylark(t reflect.Type, sky skylark.Value) (reflect.Value, error) {
	switch t.Kind() {
	case reflect.Ptr:
		val := reflect.New(t.Elem())
		elem, err := scalarFromSkylark(t.Elem(), sky)
		if err != nil {
			return reflect.Value{}, err
		}
		val.Elem().Set(elem)
		return val, nil
	case reflect.Bool:
		if val, ok := sky.(skylark.Bool); ok {
			return reflect.ValueOf(bool(val)), nil
		}
	case reflect.String:
		if val, ok := sky.(skylark.String); ok {
			return reflect.ValueOf(string(val)), nil
		}
	case reflect.Float64:
		if val, ok := skylark.AsFloat(sky); ok {
			return reflect.ValueOf(val), nil
		}
	case reflect.Float32:
		if val, ok := skylark.AsFloat(sky); ok {
			return reflect.ValueOf(float32(val)), nil
		}
	case reflect.Int64:
		if skyInt, ok := sky.(skylark.Int); ok {
			if val, ok := skyInt.Int64(); ok {
				return reflect.ValueOf(val), nil
			}
			return reflect.Value{}, fmt.Errorf("ValueError: value %v overflows type `int64'.", skyInt)
		}
	case reflect.Uint64:
		if skyInt, ok := sky.(skylark.Int); ok {
			if val, ok := skyInt.Uint64(); ok {
				return reflect.ValueOf(val), nil
			}
			return reflect.Value{}, fmt.Errorf("ValueError: value %v overflows type `uint64'.", skyInt)
		}
	case reflect.Int32:
		if skyInt, ok := sky.(skylark.Int); ok {
			if val, ok := skyInt.Int64(); ok && val >= math.MinInt32 && val <= math.MaxInt32 {
				return reflect.ValueOf(int32(val)), nil
			}
			return reflect.Value{}, fmt.Errorf("ValueError: value %v overflows type `int32'.", skyInt)
		}
	case reflect.Uint32:
		if skyInt, ok := sky.(skylark.Int); ok {
			if val, ok := skyInt.Uint64(); ok && val <= math.MaxUint32 {
				return reflect.ValueOf(uint32(val)), nil
			}
			return reflect.Value{}, fmt.Errorf("ValueError: value %v overflows type `uint32'.", skyInt)
		}
	}
	return reflect.Value{}, typeError(t, sky)
}

func enumFromSkylark(t reflect.Type, sky *skyProtoEnumValue) (reflect.Value, error) {
	if t.Kind() == reflect.Ptr {
		val := reflect.New(t.Elem())
		elem, err := enumFromSkylark(t.Elem(), sky)
		if err != nil {
			return reflect.Value{}, err
		}
		val.Elem().Set(elem)
		return val, nil
	}
	if t.Kind() == reflect.Int32 {
		if enum, ok := reflect.Zero(t).Interface().(protoEnum); ok {
			if enumTypeName(enum) == sky.typeName {
				return reflect.ValueOf(sky.value).Convert(t), nil
			}
		}
	}
	return reflect.Value{}, typeError(t, sky)
}

func typeName(t reflect.Type) string {
	// Special-case protobuf types to get more useful error messages when
	// the wrong protobuf type is assigned.
	typeName := t.String()
	messageType := reflect.TypeOf((*proto.Message)(nil)).Elem()
	enumType := reflect.TypeOf((*protoEnum)(nil)).Elem()
	if t.Implements(messageType) {
		typeName = messageTypeName(reflect.Zero(t).Interface().(proto.Message))
	} else if t.Implements(enumType) {
		typeName = enumTypeName(reflect.Zero(t).Interface().(protoEnum))
	}
	return typeName
}

func typeError(t reflect.Type, sky skylark.Value) error {
	return fmt.Errorf("TypeError: value %s (type `%s') can't be assigned to type `%s'.", sky.String(), sky.Type(), typeName(t))
}

type protoRepeated struct {
	// var x []T; reflect.ValueOf(x)
	field reflect.Value
	list  *skylark.List
}

var _ skylark.Value = (*protoRepeated)(nil)
var _ skylark.Iterable = (*protoRepeated)(nil)
var _ skylark.Sequence = (*protoRepeated)(nil)
var _ skylark.Indexable = (*protoRepeated)(nil)
var _ skylark.HasAttrs = (*protoRepeated)(nil)
var _ skylark.HasSetIndex = (*protoRepeated)(nil)
var _ skylark.HasBinary = (*protoRepeated)(nil)

func (r *protoRepeated) Attr(name string) (skylark.Value, error) {
	wrapper, ok := listMethods[name]
	if !ok {
		return nil, nil
	}
	if wrapper != nil {
		return wrapper(r), nil
	}
	return r.list.Attr(name)
}

func (r *protoRepeated) AttrNames() []string                { return r.list.AttrNames() }
func (r *protoRepeated) Freeze()                            { r.list.Freeze() }
func (r *protoRepeated) Hash() (uint32, error)              { return r.list.Hash() }
func (r *protoRepeated) Index(i int) skylark.Value          { return r.list.Index(i) }
func (r *protoRepeated) Iterate() skylark.Iterator          { return r.list.Iterate() }
func (r *protoRepeated) Len() int                           { return r.list.Len() }
func (r *protoRepeated) Slice(x, y, step int) skylark.Value { return r.list.Slice(x, y, step) }
func (r *protoRepeated) String() string                     { return r.list.String() }
func (r *protoRepeated) Truth() skylark.Bool                { return r.list.Truth() }

func (r *protoRepeated) Type() string {
	return fmt.Sprintf("list<%s>", typeName(r.field.Type().Elem()))
}

func (r *protoRepeated) wrapClear() skylark.Value {
	impl := func(thread *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		if err := skylark.UnpackPositionalArgs("clear", args, kwargs, 0); err != nil {
			return nil, err
		}
		if err := r.Clear(); err != nil {
			return nil, err
		}
		return skylark.None, nil
	}
	return skylark.NewBuiltin("clear", impl).BindReceiver(r)
}

func (r *protoRepeated) wrapAppend() skylark.Value {
	impl := func(thread *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		var val skylark.Value
		if err := skylark.UnpackPositionalArgs("append", args, kwargs, 1, &val); err != nil {
			return nil, err
		}
		if err := r.Append(val); err != nil {
			return nil, err
		}
		return skylark.None, nil
	}
	return skylark.NewBuiltin("append", impl).BindReceiver(r)
}

func (r *protoRepeated) wrapExtend() skylark.Value {
	impl := func(thread *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		var val skylark.Iterable
		if err := skylark.UnpackPositionalArgs("extend", args, kwargs, 1, &val); err != nil {
			return nil, err
		}
		if err := r.implExtend(thread, val); err != nil {
			return nil, err
		}
		return skylark.None, nil
	}
	return skylark.NewBuiltin("extend", impl).BindReceiver(r)
}

var listMethods = map[string]func(*protoRepeated) skylark.Value{
	"clear":  (*protoRepeated).wrapClear,
	"append": (*protoRepeated).wrapAppend,
	"extend": (*protoRepeated).wrapExtend,
	// "insert": (*protoRepeated).wrapInsert,
	// "pop":    (*protoRepeated).wrapPop,
	// "remove": (*protoRepeated).wrapRemove,
}

func (r *protoRepeated) Clear() error {
	if err := r.list.Clear(); err != nil {
		return err
	}
	r.field.SetLen(0)
	return nil
}

func (r *protoRepeated) Append(v skylark.Value) error {
	itemType := r.field.Type().Elem()
	if v == skylark.None {
		return typeError(itemType, v)
	}
	goVal, err := valueFromSkylark(itemType, v)
	if err != nil {
		return err
	}
	if err := r.list.Append(v); err != nil {
		return err
	}
	r.field.Set(reflect.Append(r.field, goVal))
	return nil
}

func (r *protoRepeated) implExtend(t *skylark.Thread, iterable skylark.Iterable) error {
	itemType := r.field.Type().Elem()
	var skyValues []skylark.Value
	var goValues []reflect.Value
	iter := iterable.Iterate()
	defer iter.Done()
	var skyVal skylark.Value
	for iter.Next(&skyVal) {
		if skyVal == skylark.None {
			return typeError(itemType, skyVal)
		}
		goVal, err := valueFromSkylark(itemType, skyVal)
		if err != nil {
			return err
		}
		skyValues = append(skyValues, skyVal)
		goValues = append(goValues, goVal)
	}

	listExtend, _ := r.list.Attr("extend")
	args := skylark.Tuple([]skylark.Value{
		skylark.NewList(skyValues),
	})
	if _, err := listExtend.(*skylark.Builtin).Call(t, args, nil); err != nil {
		return err
	}
	r.field.Set(reflect.Append(r.field, goValues...))
	return nil
}

func (r *protoRepeated) SetIndex(i int, v skylark.Value) error {
	itemType := r.field.Type().Elem()
	if v == skylark.None {
		return typeError(itemType, v)
	}
	goVal, err := valueFromSkylark(itemType, v)
	if err != nil {
		return err
	}
	if err := r.list.SetIndex(i, v); err != nil {
		return err
	}
	r.field.Index(i).Set(goVal)
	return nil
}

func (r *protoRepeated) Binary(op syntax.Token, y skylark.Value, side skylark.Side) (skylark.Value, error) {
	if op == syntax.PLUS {
		if side == skylark.Left {
			switch y := y.(type) {
			case *skylark.List:
				return skylark.Binary(op, r.list, y)
			case *protoRepeated:
				return skylark.Binary(op, r.list, y.list)
			}
			return nil, nil
		}
		if side == skylark.Right {
			if _, ok := y.(*skylark.List); ok {
				return skylark.Binary(op, y, r.list)
			}
			return nil, nil
		}
	}
	return nil, nil
}

type protoMap struct {
	field reflect.Value
	dict  *skylark.Dict
}

var _ skylark.Value = (*protoMap)(nil)
var _ skylark.Iterable = (*protoMap)(nil)
var _ skylark.Sequence = (*protoMap)(nil)
var _ skylark.HasAttrs = (*protoMap)(nil)
var _ skylark.HasSetKey = (*protoMap)(nil)

func (m *protoMap) Attr(name string) (skylark.Value, error) {
	wrapper, ok := dictMethods[name]
	if !ok {
		return nil, nil
	}
	if wrapper != nil {
		return wrapper(m), nil
	}
	return m.dict.Attr(name)
}

func (m *protoMap) AttrNames() []string                              { return m.dict.AttrNames() }
func (m *protoMap) Freeze()                                          { m.dict.Freeze() }
func (m *protoMap) Hash() (uint32, error)                            { return m.dict.Hash() }
func (m *protoMap) Get(k skylark.Value) (skylark.Value, bool, error) { return m.dict.Get(k) }
func (m *protoMap) Iterate() skylark.Iterator                        { return m.dict.Iterate() }
func (m *protoMap) Len() int                                         { return m.dict.Len() }
func (m *protoMap) String() string                                   { return m.dict.String() }
func (m *protoMap) Truth() skylark.Bool                              { return m.dict.Truth() }

func (m *protoMap) Type() string {
	t := m.field.Type()
	return fmt.Sprintf("map<%s, %s>", typeName(t.Key()), typeName(t.Elem()))
}

func (m *protoMap) wrapClear() skylark.Value {
	impl := func(thread *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		if err := skylark.UnpackPositionalArgs("clear", args, kwargs, 0); err != nil {
			return nil, err
		}
		if err := m.dict.Clear(); err != nil {
			return nil, err
		}
		m.field.Set(reflect.MakeMap(m.field.Type()))
		return skylark.None, nil
	}
	return skylark.NewBuiltin("clear", impl).BindReceiver(m)
}

func (m *protoMap) wrapSetDefault() skylark.Value {
	impl := func(thread *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		var key, defaultValue skylark.Value = nil, skylark.None
		if err := skylark.UnpackPositionalArgs("setdefault", args, kwargs, 1, &key, &defaultValue); err != nil {
			return nil, err
		}
		if val, ok, err := m.dict.Get(key); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return defaultValue, m.SetKey(key, defaultValue)
	}
	return skylark.NewBuiltin("setdefault", impl).BindReceiver(m)
}

func (m *protoMap) wrapUpdate() skylark.Value {
	keyType := m.field.Type().Key()
	itemType := m.field.Type().Elem()
	impl := func(thread *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		// Use the underlying Skylark `dict.update()` to get a Dict containing
		// all the new values, so we don't have to recreate the API here. After
		// the temp dict is constructed, type check.
		tempDict := &skylark.Dict{}
		tempUpdate, _ := tempDict.Attr("update")
		if _, err := tempUpdate.(*skylark.Builtin).Call(thread, args, kwargs); err != nil {
			return nil, err
		}
		for _, item := range tempDict.Items() {
			if item[0] == skylark.None {
				return nil, typeError(keyType, item[0])
			}
			if item[1] == skylark.None {
				return nil, typeError(itemType, item[1])
			}
		}
		tempMap, err := valueFromSkylark(m.field.Type(), tempDict)
		if err != nil {
			return nil, err
		}

		// tempMap is a reflected Go map containing items of the correct type.
		// Update the Dict first to catch potential immutability.
		for _, item := range tempDict.Items() {
			if err := m.dict.SetKey(item[0], item[1]); err != nil {
				return nil, err
			}
		}

		if m.field.IsNil() {
			m.field.Set(reflect.MakeMap(m.field.Type()))
		}
		for _, key := range tempMap.MapKeys() {
			m.field.SetMapIndex(key, tempMap.MapIndex(key))
		}
		return skylark.None, nil
	}
	return skylark.NewBuiltin("update", impl).BindReceiver(m)
}

func (m *protoMap) SetKey(k, v skylark.Value) error {
	keyType := m.field.Type().Key()
	itemType := m.field.Type().Elem()
	if k == skylark.None {
		return typeError(keyType, k)
	}
	if v == skylark.None {
		return typeError(itemType, v)
	}
	goKey, err := valueFromSkylark(keyType, k)
	if err != nil {
		return err
	}
	goVal, err := valueFromSkylark(itemType, v)
	if err != nil {
		return err
	}
	if err := m.dict.SetKey(k, v); err != nil {
		return err
	}
	if m.field.IsNil() {
		m.field.Set(reflect.MakeMap(m.field.Type()))
	}
	m.field.SetMapIndex(goKey, goVal)
	return nil
}

var dictMethods = map[string]func(*protoMap) skylark.Value{
	"clear": (*protoMap).wrapClear,
	"get":   nil,
	"items": nil,
	"keys":  nil,
	// "pop":        (*protoMap).wrapPop,
	// "popitem":    (*protoMap).wrapPopItem,
	"setdefault": (*protoMap).wrapSetDefault,
	"update":     (*protoMap).wrapUpdate,
	"values":     nil,
}
