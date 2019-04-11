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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	gogo_proto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/proto"
	yaml "gopkg.in/yaml.v2"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/api/apps/v1"
	_ "k8s.io/api/batch/v1"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/storage/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/stripe/skycfg"
	"github.com/stripe/skycfg/gogocompat"
)

var (
	dryRun     = flag.Bool("dry_run", false, "Print objects rendered by Skycfg and don't do anything.")
	namespace  = flag.String("namespace", "default", "Namespace to create/delete objects in.")
	configPath = flag.String("kubeconfig", os.Getenv("HOME")+"/.kube/config", "Kubernetes client config path.")
)

var k8sProtoMagic = []byte("k8s\x00")

// marshal wraps msg into runtime.Unknown object and prepends magic sequence
// to conform with Kubernetes protobuf content type.
func marshal(msg proto.Message) ([]byte, error) {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	unknownBytes, err := proto.Marshal(&runtime.Unknown{Raw: msgBytes})
	if err != nil {
		return nil, err
	}
	return append(k8sProtoMagic, unknownBytes...), nil
}

// pathForResource returns absolute URL path to access a gvr under a namespace.
func pathForResource(namespace string, gvr schema.GroupVersionResource) string {
	return pathForResourceWithName("", namespace, gvr)
}

// pathForResourceWithName is same as pathForResource but with name segment.
func pathForResourceWithName(name, namespace string, gvr schema.GroupVersionResource) string {
	segments := []string{"/apis"}

	if gvr.Group == "" {
		segments = []string{"/api", gvr.Version}
	} else {
		segments = append(segments, gvr.Group, gvr.Version)
	}

	if namespace != "" {
		segments = append(segments, "namespaces", namespace)
	}

	if gvr.Resource != "" { // Skip explicit resource for namespaces.
		segments = append(segments, gvr.Resource)
	}

	if name != "" {
		segments = append(segments, name)
	}

	return path.Join(segments...)
}

// k8sAPIPrefix is a prefix for all Kubernetes types.
const k8sAPIPrefix = "k8s.io.api."

// gvkFromMsgType assembles schema.GroupVersionKind based on Protobuf
// message type.
// e.g "k8s.io.api.core.v1.Pod" turned into "", "v1", "Pod".
// Returns error if message is not prefixed with k8sAPIPerfix.
func gvkFromMsgType(m proto.Message) (group, version, kind string, err error) {
	t := gogo_proto.MessageName(m)
	if !strings.HasPrefix(t, k8sAPIPrefix) {
		err = errors.New("unexpected message type: " + t)
		return
	}
	ss := strings.Split(t[len(k8sAPIPrefix):], ".")
	if ss[0] == "core" { // Is there a better way?
		ss[0] = ""
	}
	group, version, kind = ss[0], ss[1], ss[2]
	return
}

type kube struct {
	dClient    discovery.DiscoveryInterface
	httpClient *http.Client
	// host:port of the master.
	Master string
	dryRun bool
}

// resourceForMsg extract type information from msg and discovers appropriate
// gvr for it using discovery client.
func (k *kube) resourceForMsg(msg proto.Message) (*schema.GroupVersionResource, error) {
	g, v, kind, err := gvkFromMsgType(msg)
	if err != nil {
		return nil, err
	}

	gr, err := restmapper.GetAPIGroupResources(k.dClient)
	if err != nil {
		return nil, err
	}

	mapping, err := restmapper.NewDiscoveryRESTMapper(gr).RESTMapping(schema.GroupKind{g, kind}, v)
	if err != nil {
		return nil, err
	}
	return &mapping.Resource, nil
}

// parseResponse parses response body to extract unstructred.Unstructured
// and extracts http error code.
// Returns status message on success and error on failure (includes HTTP
// response codes not in 2XX).
func parseResponse(r *http.Response) (status string, err error) {
	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body (response code: %d): %v", r.StatusCode, err)
	}

	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, raw)
	if err != nil {
		return "", fmt.Errorf("failed to parse Status (response code: %d): %v", r.StatusCode, err)
	}

	if r.StatusCode < 200 || r.StatusCode >= 300 {
		return "", fmt.Errorf("%s (response code: %d)", apierrors.FromObject(obj).Error(), r.StatusCode)
	}

	un := obj.(*unstructured.Unstructured)
	gvk := un.GroupVersionKind()

	if gvk.Kind == "Status" {
		ds, found, err := unstructured.NestedStringMap(un.Object, "details")
		if err != nil {
			return "", err
		}
		if !found {
			return "", errors.New("`details' map is missing from Status")
		}
		return fmt.Sprintf("%s.%s `%s'", ds["kind"], ds["group"], ds["name"]), nil
	}

	return fmt.Sprintf("%s.%s `%s'", strings.ToLower(gvk.Kind), gvk.Group, un.GetName()), nil
}

// Up creates msg object in Kubernetes. Determines the path based on msg
// registered type. Object by the same path must not already exist.
func (k *kube) Up(ctx context.Context, namespace string, msg proto.Message) (string, error) {
	gvr, err := k.resourceForMsg(msg)
	if err != nil {
		return "", err
	}
	uri := pathForResource(namespace, *gvr)
	bs, err := marshal(msg)
	if err != nil {
		return "", err
	}

	url := k.Master + uri
	if k.dryRun {
		return "POST to " + url, nil
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bs))

	// NB: Will not work for CRDs (only json encoding is supported).
	// Set body type as marshalled Protobuf.
	req.Header.Set("Content-Type", "application/vnd.kubernetes.protobuf")
	// Set reply type to accept Protobuf as well.
	//req.Header.Set("Accept", "application/vnd.kubernetes.protobuf")

	resp, err := k.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}

	rMsg, err := parseResponse(resp)
	if err != nil {
		return "", err
	}
	return rMsg + " created", nil
}

// Down deletes object name in namespace. Resource is computed based on msg
// registered type.
func (k *kube) Down(ctx context.Context, name, namespace string, msg proto.Message) (string, error) {
	gvr, err := k.resourceForMsg(msg)
	if err != nil {
		return "", err
	}

	url := k.Master + pathForResourceWithName(name, namespace, *gvr)
	if k.dryRun {
		return "POST to " + url, nil
	}

	req, err := http.NewRequest("DELETE", url, nil)

	resp, err := k.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}

	rMsg, err := parseResponse(resp)
	if err != nil {
		return "", err
	}
	return rMsg + " deleted", nil
}

func main() {
	flag.Parse()
	argv := flag.Args()

	if len(argv) != 3 {
		fmt.Fprintf(os.Stderr, `Demo Kubernetes CLI for Skycfg, a library for building complex typed configs.

usage: %s [ up | down ] NAME FILENAME
`, os.Args[0])
		os.Exit(1)
	}
	action, name, filename := argv[0], argv[1], argv[2]

	restConfig, err := clientcmd.BuildConfigFromFlags("", *configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build rest config: %v\n", err)
		os.Exit(1)
	}
	dc := discovery.NewDiscoveryClientForConfigOrDie(restConfig)
	t, err := rest.TransportFor(restConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build HTTP roundtripper: %v\n", err)
		os.Exit(1)
	}
	k := &kube{
		httpClient: &http.Client{
			Transport: t,
		},
		Master:  restConfig.Host,
		dClient: dc,
		dryRun:  *dryRun,
	}

	config, err := skycfg.Load(context.Background(), filename, skycfg.WithProtoRegistry(gogocompat.ProtoRegistry()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %q: %v\n", filename, err)
		os.Exit(1)
	}

	var jsonMarshaler = &jsonpb.Marshaler{OrigName: true}
	protos, err := config.Main(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error evaluating %q: %v\n", config.Filename(), err)
		os.Exit(1)
	}
	for _, msg := range protos {
		ctx := context.Background()
		var err error
		var reply string
		switch action {
		case "up":
			reply, err = k.Up(ctx, *namespace, msg)
		case "down":
			reply, err = k.Down(ctx, name, *namespace, msg)
		default:
			panic("Unknown action: " + action)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		} else {
			fmt.Printf("%s\n", reply)
		}

		if *dryRun {
			marshaled, err := jsonMarshaler.MarshalToString(msg)
			sep := ""
			if err != nil {
				fmt.Fprintf(os.Stderr, "json.Marshal: %v\n", err)
				continue
			}
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
			fmt.Printf("%s%s\n", sep, marshaled)
		}

	}
}
