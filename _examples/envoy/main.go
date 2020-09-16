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
package main

import (
	"os"
	"fmt"
	"context"

	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/ratelimit"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stripe/skycfg"
)

func main() {
	argv := os.Args

	if len(argv) != 2 {
		fmt.Fprintf(os.Stderr, `Demo Envoy CLI for Skycfg, a library for building complex typed configs.

usage: %s FILENAME
`, os.Args[0])
		os.Exit(1)
	}
	filename := argv[1]

	config, err := skycfg.Load(
		context.Background(), filename,
		skycfg.WithGlobals(skycfg.UnstablePredeclaredModules(nil)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %q: %v\n", filename, err)
		os.Exit(1)
	}

	marshaler := &jsonpb.Marshaler{
		OrigName: true,
		Indent: "\t",
	}
	protos, err := config.Main(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error evaluating %q: %v\n", config.Filename(), err)
		os.Exit(1)
	}

	for _, p := range protos {
		marshaler.Marshal(os.Stdout, p)
	}
}