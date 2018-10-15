package main

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"syscall/js"

	gogo_proto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	yaml "gopkg.in/yaml.v2"

	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "k8s.io/api/apps/v1"
	_ "k8s.io/api/batch/v1"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/storage/v1"

	"github.com/stripe/skycfg"
)

type stubFileReader struct {
	content string
}

func (*stubFileReader) Resolve(ctx context.Context, name, fromPath string) (string, error) {
	if fromPath == "" {
		return name, nil
	}
	return "", fmt.Errorf("load(%q): not available in webasm demo", name)
}

func (r *stubFileReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return []byte(r.content), nil
}

type protoRegistry struct{}

func (*protoRegistry) UnstableProtoMessageType(name string) (reflect.Type, error) {
	if t := proto.MessageType(name); t != nil {
		return t, nil
	}
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

func messagesToYaml(msgs []proto.Message) (string, error) {
	var buf bytes.Buffer
	var jsonMarshaler = &jsonpb.Marshaler{OrigName: true}
	for _, msg := range msgs {
		jsonData, err := jsonMarshaler.MarshalToString(msg)
		if err != nil {
			return "", fmt.Errorf("json.Marshal: %v", err)
		}
		var yamlMap yaml.MapSlice
		if err := yaml.Unmarshal([]byte(jsonData), &yamlMap); err != nil {
			return "", fmt.Errorf("yaml.Unmarshal: %v", err)
		}
		yamlData, err := yaml.Marshal(yamlMap)
		if err != nil {
			return "", fmt.Errorf("yaml.Marshal: %v", err)
		}
		fmt.Fprintf(&buf, "---\n%s", string(yamlData))
	}
	return buf.String(), nil
}

func main() {
	toYaml := func(args []js.Value) {
		content := args[0].String()
		config, err := skycfg.Load(
			context.Background(),
			"<stdin>",
			skycfg.WithProtoRegistry(&protoRegistry{}),
			skycfg.WithFileReader(&stubFileReader{content}))
		if err != nil {
			args[1].Call("err", err.Error())
			return
		}
		messages, err := config.Main(context.Background())
		if err != nil {
			args[1].Call("err", err.Error())
			return
		}
		yaml, err := messagesToYaml(messages)
		if err != nil {
			args[1].Call("err", err.Error())
			return
		}
		args[1].Call("ok", yaml)
	}
	js.Global().Set("skycfg_to_yaml", js.NewCallback(toYaml))
	c := make(chan struct{}, 0)
	<-c
}
