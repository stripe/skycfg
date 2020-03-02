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
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gogo/protobuf/jsonpb"
	"go.starlark.net/repl"
	"go.starlark.net/starlark"
	yaml "gopkg.in/yaml.v2"

	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "k8s.io/api/apps/v1"
	_ "k8s.io/api/batch/v1"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/storage/v1"

	"github.com/stripe/skycfg"
	"github.com/stripe/skycfg/gogocompat"
)

func main() {
	flag.Parse()
	argv := flag.Args()
	var mode, filename string
	switch len(argv) {
	case 1:
		mode = "repl"
		filename = argv[0]
	case 2:
		mode = argv[0]
		filename = argv[1]
	default:
		fmt.Fprintf(os.Stderr, `Demo REPL for Skycfg, a library for building complex typed configs.

usage: %s FILENAME
       %s [ repl | json | yaml ] FILENAME
`, os.Args[0], os.Args[0])
		os.Exit(1)
	}

	switch mode {
	case "repl", "json", "yaml":
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand %q: expected `repl', `json', or `yaml'\n")
		os.Exit(1)
	}

	config, err := skycfg.Load(context.Background(), filename, skycfg.WithProtoRegistry(gogocompat.ProtoRegistry()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %q: %v\n", filename, err)
		os.Exit(1)
	}

	if mode == "repl" {
		fmt.Printf("Entering Skycfg REPL for %s\n", config.Filename())
		thread := &starlark.Thread{}
		globals := make(starlark.StringDict)
		globals["help"] = &replHelp{config}
		globals["exit"] = &replExit{}
		for key, value := range config.Globals() {
			globals[key] = value
		}
		for key, value := range config.Locals() {
			globals[key] = value
		}
		repl.REPL(thread, globals)
		return
	}

	var jsonMarshaler = &jsonpb.Marshaler{OrigName: true}
	protos, err := config.Main(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error evaluating %q: %v\n", config.Filename(), err)
		os.Exit(1)
	}
	for _, msg := range protos {
		marshaled, err := jsonMarshaler.MarshalToString(msg)
		sep := ""
		if err != nil {
			fmt.Fprintf(os.Stderr, "json.Marshal: %v\n", err)
			continue
		}
		if mode == "yaml" {
			var yamlMap yaml.MapSlice
			if err := yaml.Unmarshal([]byte(marshaled), &yamlMap); err != nil {
				panic(fmt.Sprintf("yaml.Unmarshal: %v", err))
			}
			yamlMarshaled, err := yaml.Marshal(yamlMap)
			if err != nil {
				panic(fmt.Sprintf("yaml.Marshal: %v", err))
			}
			marshaled = string(yamlMarshaled)
			sep = "---\n"
		}
		fmt.Printf("%s%s\n", sep, marshaled)
	}
}

type replHelp struct {
	config *skycfg.Config
}

func (*replHelp) Type() string         { return "skycfg_repl_help" }
func (*replHelp) Freeze()              {}
func (*replHelp) Truth() starlark.Bool { return starlark.True }

func (help *replHelp) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", help.Type())
}

func (help *replHelp) String() string {
	var buf bytes.Buffer
	buf.WriteString(`Skycfg, a library for building complex typed configs.

Homepage: https://github.com/stripe/skycfg
API docs: https://godoc.org/github.com/stripe/skycfg

Pre-defined values:
`)
	for name := range help.config.Globals() {
		fmt.Fprintf(&buf, "* %s\n", name)
	}

	fmt.Fprintf(&buf, "\nConfig values (from %s):\n", help.config.Filename())
	for name := range help.config.Locals() {
		fmt.Fprintf(&buf, "* %s\n", name)
	}
	return buf.String()
}

type replExit struct{}

func (*replExit) Type() string         { return "skycfg_repl_exit" }
func (*replExit) Freeze()              {}
func (*replExit) Truth() starlark.Bool { return starlark.True }
func (*replExit) String() string       { return "Use exit() or Ctrl-D (i.e. EOF) to exit" }
func (*replExit) Name() string         { return "exit" }

func (exit *replExit) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", exit.Type())
}

func (*replExit) Call(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs("exit", args, kwargs, 0); err != nil {
		return nil, err
	}
	fmt.Printf("\n")
	os.Exit(0)
	return starlark.None, nil
}
