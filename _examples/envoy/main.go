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
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/ratelimit"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	server "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/stripe/skycfg"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	health_pb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	node = ""
)

var (
	addr  = flag.String("addr", ":8080", "Address to start the discovery service on")
	level = flag.String("log-level", "info", "Log level to write application logs")

	logger  = log.New()
	limiter = rate.NewLimiter(1, 1)

	cbFuncs = server.CallbackFuncs{
		StreamOpenFunc: func(ctx context.Context, id int64, s string) error {
			logger.Printf("[xds][%d] accepted connection from peer: %+v", id, s)
			return nil
		},
		StreamClosedFunc: func(id int64) {
			logger.Printf("[xds][%d] connection closed for peer", id)
		},
		StreamRequestFunc: func(id int64, req *api.DiscoveryRequest) error {
			if err := limiter.Wait(context.Background()); err != nil {
				logger.Errorf("[xds][%d, %s] could not enforce rate limit: %+v", id, req.GetTypeUrl(), err)
			}

			logger.Printf(
				"[xds][%d, %s] recieved discovery request (version %q) from peer: %+v",
				id, req.GetTypeUrl(), req.GetVersionInfo(), req.GetNode().GetId(),
			)
			logger.Debugf(
				"[%d, %s] discovery request contents: %#v",
				id, req.GetTypeUrl(), req,
			)
			return nil
		},
		StreamResponseFunc: func(id int64, req *api.DiscoveryRequest, res *api.DiscoveryResponse) {
			logger.Printf(
				"[xds][%d, %s] sending discovery response (version %q) to peer: %+v",
				id, req.GetTypeUrl(), res.GetVersionInfo(), req.GetNode().GetId(),
			)
			logger.Debugf(
				"[xds][%d, %s] discovery response contents: %#v",
				id, req.GetTypeUrl(), res,
			)
		},
	}
)

type nodeHashFunc func(node *core.Node) string

func (h nodeHashFunc) ID(node *core.Node) string {
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

func resourcesEqual(a, b cache.Resources) bool {
	for aKey, aValue := range a.Items {
		bValue := b.Items[aKey]
		if !proto.Equal(aValue, bValue) {
			return false
		}
	}

	for bKey, bValue := range b.Items {
		aValue := a.Items[bKey]
		if !proto.Equal(aValue, bValue) {
			return false
		}
	}
	return true
}

func snapshotsEqual(a, b cache.Snapshot) bool {
	for i := 0; i < len(a.Resources) && i < len(b.Resources); i++ {
		aResources := a.Resources[i]
		bResources := b.Resources[i]

		if !resourcesEqual(aResources, bResources) {
			return false
		}
	}
	return true
}

type ConfigLoader struct {
	Cache   cache.SnapshotCache
	version uint

	// Protects cache and version
	sync.Mutex
}

type validatableProto interface {
	proto.Message
	Validate() error
}

func (c *ConfigLoader) validateProtos(protos []proto.Message) error {
	for _, proto := range protos {
		validatable, ok := proto.(validatableProto)
		logger.Debugf("validating: %+v", proto)
		if !ok {
			logger.Debugf("cannot validate: %+v", proto)
			continue
		}

		if err := validatable.Validate(); err != nil {
			return err
		}
	}
	return nil
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

	if err := c.validateProtos(protos); err != nil {
		return nil, fmt.Errorf("validation error: %+v", err)
	}

	return protos, nil
}

func (c *ConfigLoader) Load(filename string) error {
	protos, err := c.evalSkycfg(filename)
	if err != nil {
		return err
	}

	resourcesByType := resourcesByType(fmt.Sprintf("%d-%d-%d", rand.Int(), os.Getpid(), c.version), protos)

	c.Lock()
	defer c.Unlock()
	oldSnapshot, _ := c.Cache.GetSnapshot(node)
	snapshot := oldSnapshot

	// Update all changed resources in the new snapshot
	for resourceType, oldResources := range oldSnapshot.Resources {
		newResources := resourcesByType[resourceType]
		if !resourcesEqual(oldResources, newResources) {
			snapshot.Resources[resourceType] = newResources
		}
	}

	if err := snapshot.Consistent(); err != nil {
		return fmt.Errorf("snapshot not consistent: %+v", err)
	}

	// Only send a config update if we've actually changed anything
	if snapshotsEqual(snapshot, oldSnapshot) {
		logger.Printf("[skycfg] skipping update as all resources are equal")
		return nil
	}

	c.Cache.SetSnapshot(node, snapshot)
	c.version++
	return nil
}

func main() {
	flag.Parse()

	argv := flag.Args()

	logrusLevel, err := log.ParseLevel(*level)
	if err != nil {
		logger.Fatalf("could not set log level to %q: %+v", *level, err)
	}
	logger.SetLevel(logrusLevel)

	if len(argv) != 1 {
		fmt.Fprintf(os.Stderr, `Demo Envoy CLI for Skycfg, a library for building complex typed configs.

usage: %s FILENAME
`, os.Args[0])
		os.Exit(1)
	}

	rand.Seed(time.Now().UnixNano())

	filename := argv[0]

	h := nodeHashFunc(func(_ *core.Node) string {
		return node
	})
	c := cache.NewSnapshotCache(true, h, logger)

	loader := &ConfigLoader{
		Cache: c,
	}

	if err := loader.Load(filename); err != nil {
		logger.Fatalf("%+v", err)
	}

	server := server.NewServer(context.Background(), c, cbFuncs)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}

	logger.Infof("Starting server on %s", *addr)
	grpcServer := grpc.NewServer()

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	health_pb.RegisterHealthServer(grpcServer, health.NewServer())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSTOP)

	go func(sigCh chan os.Signal) {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGHUP:
				if err := loader.Load(filename); err != nil {
					logger.Errorf("%+v", err)
				}
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGSTOP:
				logger.Warnf("caught signal %v, exiting", sig)
				grpcServer.Stop()
			default:
			}
		}
	}(sigCh)

	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatalf("%+v", err)
	}
}
