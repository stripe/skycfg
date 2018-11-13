package skycfg

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
	descriptor_pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"go.starlark.net/starlark"
)

type defaultProtoRegistry struct{}

func (*defaultProtoRegistry) UnstableProtoMessageType(name string) (reflect.Type, error) {
	return proto.MessageType(name), nil
}

func (*defaultProtoRegistry) UnstableEnumValueMap(name string) map[string]int32 {
	return proto.EnumValueMap(name)
}

// NewMessageType creates a Starlark value representing a named Protobuf message type.
//
// The message type must have been registered with the protobuf library, and implement
// the expected interfaces for a generated .pb.go message struct.
func newMessageType(registry ProtoRegistry, name string) (starlark.Value, error) {
	if registry == nil {
		registry = &defaultProtoRegistry{}
	}
	goType, err := registry.UnstableProtoMessageType(name)
	if err != nil {
		return nil, err
	}
	if goType == nil {
		return nil, fmt.Errorf("Protobuf message type %q not found", name)
	}

	var emptyMsg descriptor.Message
	if goType.Kind() == reflect.Ptr {
		goValue := reflect.New(goType.Elem()).Interface()
		if iface, ok := goValue.(descriptor.Message); ok {
			emptyMsg = iface
		}
	}
	if emptyMsg == nil {
		// Return a slightly useful error in case some clever person has
		// manually registered a `proto.Message` that doesn't use pointer
		// receivers.
		return nil, fmt.Errorf("InternalError: %v is not a generated proto.Message", goType)
	}
	fileDesc, msgDesc := descriptor.ForMessage(emptyMsg)
	mt := &skyProtoMessageType{
		registry: registry,
		fileDesc: fileDesc,
		msgDesc:  msgDesc,
		emptyMsg: emptyMsg,
	}
	if gotName := mt.Name(); name != gotName {
		// All the protobuf lookups are by name, so it's important that
		// buggy self-registered protobuf types don't get mixed in.
		return nil, fmt.Errorf("InternalError: %v has unexpected protobuf type name %q (wanted %q)", goType, gotName, name)
	}
	return mt, nil

}

// A Starlark built-in type representing a Protobuf message type. This is the
// message type itself rather than any particular message value.
type skyProtoMessageType struct {
	registry ProtoRegistry
	fileDesc *descriptor_pb.FileDescriptorProto
	msgDesc  *descriptor_pb.DescriptorProto

	// An empty protobuf message of the appropriate type.
	emptyMsg proto.Message
}

var _ starlark.HasAttrs = (*skyProtoMessageType)(nil)
var _ starlark.Callable = (*skyProtoMessageType)(nil)

func (mt *skyProtoMessageType) String() string {
	return fmt.Sprintf("<proto.MessageType %q>", mt.Name())
}
func (mt *skyProtoMessageType) Type() string         { return "proto.MessageType" }
func (mt *skyProtoMessageType) Freeze()              {}
func (mt *skyProtoMessageType) Truth() starlark.Bool { return starlark.True }
func (mt *skyProtoMessageType) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", mt.Type())
}

func (mt *skyProtoMessageType) Name() string {
	return messageTypeName(mt.emptyMsg)
}

func (mt *skyProtoMessageType) Attr(attrName string) (starlark.Value, error) {
	msgName := fmt.Sprintf("%s.%s", mt.msgDesc.GetName(), attrName)
	enumName := strings.Replace(msgName, ".", "_", -1)
	if pkg := mt.fileDesc.GetPackage(); pkg != "" {
		msgName = fmt.Sprintf("%s.%s", pkg, msgName)
		enumName = fmt.Sprintf("%s.%s", pkg, enumName)
	}

	registry := mt.registry
	if registry == nil {
		registry = &defaultProtoRegistry{}
	}
	if ev := registry.UnstableEnumValueMap(enumName); ev != nil {
		return &skyProtoEnumType{
			name:     msgName, // note: not enumName, use dotted name here
			valueMap: ev,
		}, nil
	}

	return newMessageType(mt.registry, msgName)
}

func (mt *skyProtoMessageType) AttrNames() []string {
	// TODO: Implement when go-protobuf gains support for listing the
	// registered message types in a Protobuf package. Since `dir(msgType)`
	// should return the names of its nested messages, this needs to be
	// implemented as a filtered version of `skyProtoPackage.AttrNames()`
	// that checks for `HasPrefix(msgName, mt.Name() + ".")`.
	//
	// https://github.com/golang/protobuf/issues/623
	return nil
}

func (mt *skyProtoMessageType) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// This is semantically the constructor of a protobuf message, and we
	// want it to accept only kwargs (where keys are protobuf field names).
	// Inject a useful error message if a user tries to pass positional args.
	if err := starlark.UnpackPositionalArgs(mt.Name(), args, nil, 0); err != nil {
		return nil, err
	}

	wrapper := NewSkyProtoMessage(proto.Clone(mt.emptyMsg))

	// Parse the kwarg set into a map[string]starlark.Value, containing one
	// entry for each provided kwarg. Keys are the original protobuf field names.
	// This lets the starlark kwarg parser handle most of the error reporting,
	// except type errors which are deferred until later.
	var parserPairs []interface{}
	parsedKwargs := make(map[string]*starlark.Value, len(kwargs))

	for _, field := range wrapper.fields {
		v := new(starlark.Value)
		parsedKwargs[field.OrigName] = v
		parserPairs = append(parserPairs, field.OrigName+"?", v)
	}
	if err := starlark.UnpackArgs(mt.Name(), nil, kwargs, parserPairs...); err != nil {
		return nil, err
	}
	for fieldName, starlarkValue := range parsedKwargs {
		if *starlarkValue == nil {
			continue
		}
		if err := wrapper.SetField(fieldName, *starlarkValue); err != nil {
			return nil, err
		}
	}
	return wrapper, nil
}
