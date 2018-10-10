package skycfg

import (
	"fmt"
	"sort"
	"reflect"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/google/skylark"
	yaml "gopkg.in/yaml.v2"
)

// UNSTABLE extension point for configuring how protobuf messages are loaded.
//
// This will be stabilized after the go-protobuf v2 API has reached GA.
type unstableProtoRegistry interface {
	// UNSTABLE lookup from full protobuf message name to a Go type of the
	// generated message struct.
	UnstableProtoMessageType(name string) (reflect.Type, error)
}

func newProtoModule(registry unstableProtoRegistry) skylark.Value {
	mod := &protoModule{
		registry: registry,
		attrs: skylark.StringDict{
			"clear":        skylark.NewBuiltin("proto.clear", fnProtoClear),
			"clone":        skylark.NewBuiltin("proto.clone", fnProtoClone),
			"merge":        skylark.NewBuiltin("proto.merge", fnProtoMerge),
			"set_defaults": skylark.NewBuiltin("proto.set_defaults", fnProtoSetDefaults),
			"to_json":      skylark.NewBuiltin("proto.to_json", fnProtoToJson),
			"to_text":      skylark.NewBuiltin("proto.to_text", fnProtoToText),
			"to_yaml":      skylark.NewBuiltin("proto.to_json", fnProtoToYaml),
		},
	}
	mod.attrs["package"] = skylark.NewBuiltin("proto.package", mod.fnProtoPackage)
	return mod
}

type protoModule struct {
	registry unstableProtoRegistry
	attrs skylark.StringDict
}

var _ skylark.HasAttrs = (*protoModule)(nil)

func (mod *protoModule) String() string      { return fmt.Sprintf("<module %q>", "proto") }
func (mod *protoModule) Type() string        { return "module" }
func (mod *protoModule) Freeze()             { mod.attrs.Freeze() }
func (mod *protoModule) Truth() skylark.Bool { return skylark.True }
func (mod *protoModule) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", mod.Type())
}

func (mod *protoModule) Attr(name string) (skylark.Value, error) {
	if val, ok := mod.attrs[name]; ok {
		return val, nil
	}
	return nil, nil
}

func (mod *protoModule) AttrNames() []string {
	var names []string
	for name := range mod.attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Implementation of the `proto.clear()` built-in function.
// Reset protobuf state to the default values.
func fnProtoClear(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var msg *skyProtoMessage
	if err := wantSingleProtoMessage("proto.clear", args, kwargs, &msg); err != nil {
		return nil, err
	}
	if err := msg.checkMutable("clear"); err != nil {
		return nil, err
	}
	msg.msg.Reset()
	return msg, nil
}

// Implementation of the `proto.clone()` built-in function.
// Creates a deep copy of a protobuf.
func fnProtoClone(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var msg *skyProtoMessage
	if err := wantSingleProtoMessage("proto.clear", args, kwargs, &msg); err != nil {
		return nil, err
	}
	return newSkyProtoMessage(proto.Clone(msg.msg)), nil
}

// Implementation of the `proto.merge()` built-in function.
// Merge merges src into dst. Repeated fields will be appended.
func fnProtoMerge(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var val1, val2 skylark.Value
	if err := skylark.UnpackPositionalArgs("proto.merge", args, kwargs, 2, &val1, &val2); err != nil {
		return nil, err
	}
	dst, ok := val1.(*skyProtoMessage)
	if !ok {
		return nil, fmt.Errorf("%s: for parameter 1: got %s, want proto.Message", "proto.merge", val1.Type())
	}
	src, ok := val2.(*skyProtoMessage)
	if !ok {
		return nil, fmt.Errorf("%s: for parameter 2: got %s, want proto.Message", "proto.merge", val2.Type())
	}
	if src.Type() != dst.Type() {
		return nil, fmt.Errorf("%s: types are not the same: got %s and %s", "proto.merge", src.Type(), dst.Type())
	}
	if err := dst.checkMutable("merge into"); err != nil {
		return nil, err
	}
	proto.Merge(dst.msg, src.msg)
	return dst, nil
}

// Implementation of the `proto.set_default()` built-in function.
// Sets unset protobuf fields to their default values.
func fnProtoSetDefaults(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var msg *skyProtoMessage
	if err := wantSingleProtoMessage("proto.set_defaults", args, kwargs, &msg); err != nil {
		return nil, err
	}
	if err := msg.checkMutable("set field defaults of"); err != nil {
		return nil, err
	}
	proto.SetDefaults(msg.msg)
	return msg, nil
}

// Implementation of the `proto.package()` built-in function.
//
// Note: doesn't do any sort of input validation, because the go-protobuf
// message registration data isn't currently exported in a useful way
// (see https://github.com/golang/protobuf/issues/623).
func (mod *protoModule) fnProtoPackage(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var packageName string
	if err := skylark.UnpackPositionalArgs("proto.package", args, kwargs, 1, &packageName); err != nil {
		return nil, err
	}
	return &skyProtoPackage{
		registry: mod.registry,
		name: packageName,
	}, nil
}

func wantSingleProtoMessage(fnName string, args skylark.Tuple, kwargs []skylark.Tuple, msg **skyProtoMessage) error {
	var val skylark.Value
	if err := skylark.UnpackPositionalArgs(fnName, args, kwargs, 1, &val); err != nil {
		return err
	}
	gotMsg, ok := val.(*skyProtoMessage)
	if !ok {
		return fmt.Errorf("%s: for parameter 1: got %s, want proto.Message", fnName, val.Type())
	}
	*msg = gotMsg
	return nil
}

// Implementation of the `proto.to_text()` built-in function. Returns the
// text-formatted content of a protobuf message.
func fnProtoToText(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var msg *skyProtoMessage
	if err := wantSingleProtoMessage("proto.to_text", args, []skylark.Tuple{}, &msg); err != nil {
		return nil, err
	}
	var textMarshaler = &proto.TextMarshaler{Compact: true}
	if len(kwargs) > 0 {
		compact := true
		if err := skylark.UnpackArgs("proto.to_text", nil, kwargs, "compact", &compact); err != nil {
			return nil, err
		}
		if compact {
			textMarshaler.Compact = true
		} else {
			textMarshaler.Compact = false
		}
	}
	text := (textMarshaler).Text(msg.msg)
	return skylark.String(text), nil
}

// Implementation of the `proto.to_json()` built-in function. Returns the
// JSON-formatted content of a protobuf message.
func fnProtoToJson(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var msg *skyProtoMessage
	if err := wantSingleProtoMessage("proto.to_json", args, []skylark.Tuple{}, &msg); err != nil {
		return nil, err
	}
	var jsonMarshaler = &jsonpb.Marshaler{OrigName: true}
	if len(kwargs) > 0 {
		compact := true
		if err := skylark.UnpackArgs("proto.to_json", nil, kwargs, "compact", &compact); err != nil {
			return nil, err
		}
		if compact {
			jsonMarshaler.Indent = ""
		} else {
			jsonMarshaler.Indent = "\t"
		}
	}
	jsonData, err := (jsonMarshaler).MarshalToString(msg.msg)
	if err != nil {
		return nil, err
	}
	return skylark.String(jsonData), nil
}

// Implementation of the `proto.to_yaml()` built-in function. Returns the
// YAML-formatted content of a protobuf message.
func fnProtoToYaml(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var msg *skyProtoMessage
	if err := wantSingleProtoMessage("proto.to_yaml", args, kwargs, &msg); err != nil {
		return nil, err
	}
	jsonData, err := (&jsonpb.Marshaler{OrigName: true}).MarshalToString(msg.msg)
	if err != nil {
		return nil, err
	}
	var yamlMap yaml.MapSlice
	if err := yaml.Unmarshal([]byte(jsonData), &yamlMap); err != nil {
		return nil, err
	}
	yamlData, err := yaml.Marshal(yamlMap)
	if err != nil {
		return nil, err
	}
	return skylark.String(yamlData), nil
}
