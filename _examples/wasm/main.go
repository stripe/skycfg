// Copyright 2018 The Skycfg Authors.
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

package main

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	yaml "gopkg.in/yaml.v2"

	"github.com/stripe/skycfg"
	_ "github.com/stripe/skycfg/_examples/wasm/addressbook"
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

func runDemo(content string) ([]js.Value, error) {
	config, err := skycfg.Load(
		context.Background(),
		"<stdin>",
		skycfg.WithFileReader(&stubFileReader{content}),
	)
	if err != nil {
		return nil, err
	}

	messages, err := config.Main(context.Background())
	if err != nil {
		return nil, err
	}

	var out []js.Value
	for _, msg := range messages {
		msg := proto.MessageV2(msg)
		jsonData, err := (protojson.MarshalOptions{
			UseProtoNames:   true,
			Indent:          "  ",
			EmitUnpopulated: true,
		}).Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("json.Marshal: %v", err)
		}
		var yamlMap yaml.MapSlice
		if err := yaml.Unmarshal(jsonData, &yamlMap); err != nil {
			return nil, fmt.Errorf("yaml.Unmarshal: %v", err)
		}
		yamlData, err := yaml.Marshal(yamlMap)
		if err != nil {
			return nil, fmt.Errorf("yaml.Marshal: %v", err)
		}
		textData, err := (prototext.MarshalOptions{
			Indent: "  ",
		}).Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("prototext.Marshal: %v", err)
		}
		out = append(out, js.ValueOf(map[string]interface{}{
			"yaml":  string(yamlData),
			"json":  string(jsonData),
			"proto": string(textData),
		}))
	}
	return out, nil
}

func jsMain(this js.Value, args []js.Value) interface{} {
	content := args[0].String()
	result, err := runDemo(content)
	if err != nil {
		args[1].Call("err", err.Error())
		return nil
	}
	var out []interface{}
	for _, item := range result {
		out = append(out, js.ValueOf(item))
	}
	args[1].Call("ok", out)
	return nil
}

func main() {
	js.Global().Set("skycfg_main", js.FuncOf(jsMain).Value)
	c := make(chan struct{}, 0)
	<-c
}
