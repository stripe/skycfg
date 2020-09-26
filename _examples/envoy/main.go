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
	"context"
	"fmt"
	"net"
	"os"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/ratelimit"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"google.golang.org/grpc"

	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	server "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/stripe/skycfg"
)

const (
	node = ""
	port = 8080
)

var (
	logger = log.New()
)

type hasher func(node *core.Node) string

func (h hasher) ID(node *core.Node) string {
	return h(node)
}

func resourcesByType(version string, protos []proto.Message) [envoy_types.UnknownType]cache.Resources {
	m := map[envoy_types.ResponseType][]envoy_types.Resource{}
	for _, proto := range protos {
		switch proto.(type) {
		case *api.ClusterLoadAssignment:
			m[envoy_types.Endpoint] = append(m[envoy_types.Endpoint], envoy_types.Resource(proto))
		case *api.Listener:
			m[envoy_types.Listener] = append(m[envoy_types.Listener], envoy_types.Resource(proto))
		case *api.Cluster:
			m[envoy_types.Cluster] = append(m[envoy_types.Cluster], envoy_types.Resource(proto))
		default:
			m[envoy_types.UnknownType] = append(m[envoy_types.UnknownType], envoy_types.Resource(proto))
		}
	}

	ret := [envoy_types.UnknownType]cache.Resources{}
	for k, v := range m {
		ret[k] = cache.NewResources(version, v)
	}

	return ret
}

type ConfigLoader struct {
	Cache   cache.SnapshotCache
	version uint
}

func (c *ConfigLoader) evalSkycfg(filename string) ([]proto.Message, error) {
	config, err := skycfg.Load(
		context.Background(), filename,
		skycfg.WithGlobals(skycfg.UnstablePredeclaredModules(nil)),
	)
	if err != nil {
		return nil, fmt.Errorf("error loading %q: %v", filename, err)
	}

	protos, err := config.Main(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error evaluating %q: %v", config.Filename(), err)
	}

	return protos, nil
}

func (c *ConfigLoader) Load(filename string) {
	protos, err := c.evalSkycfg(filename)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	resourcesByType := resourcesByType(fmt.Sprintf("%d", c.version), protos)

	snapshot := cache.Snapshot{
		Resources: resourcesByType,
	}

	c.Cache.SetSnapshot(node, snapshot)
	c.version++
}

func main() {
	argv := os.Args

	if len(argv) != 2 {
		fmt.Fprintf(os.Stderr, `Demo Envoy CLI for Skycfg, a library for building complex typed configs.

usage: %s FILENAME
`, os.Args[0])
		os.Exit(1)
	}

	filename := argv[1]
	h := hasher(func(_ *core.Node) string {
		return node
	})
	c := cache.NewSnapshotCache(true, h, logger)

	loader := &ConfigLoader{
		Cache: c,
	}
	loader.Load(filename)

	cbFuncs := server.CallbackFuncs{
		StreamOpenFunc: func(ctx context.Context, id int64, s string) error {
			logger.Printf("[%d] accepted connection from peer: %+v", id, s)
			return nil
		},
		StreamRequestFunc: func(id int64, req *api.DiscoveryRequest) error {
			logger.Printf(
				"[%d, %s] recieved discovery request from peer: %+v",
				id, req.GetTypeUrl(), req.GetNode().GetId(),
			)
			return nil
		},
		StreamResponseFunc: func(id int64, req *api.DiscoveryRequest, res *api.DiscoveryResponse) {
			logger.Printf(
				"[%d, %s] sending discovery response to peer: %+v",
				id, req.GetTypeUrl(), req.GetNode().GetId(),
			)
		},
	}

	ctx := context.Background()
	server := server.NewServer(ctx, c, cbFuncs)

	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Infof("Starting server on port %v", addr)
	grpcServer := grpc.NewServer()

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("err: %+v", err)
	}
}
