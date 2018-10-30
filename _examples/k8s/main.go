package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	gogo_proto "github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/proto"
	yaml "gopkg.in/yaml.v2"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "k8s.io/api/apps/v1"
	_ "k8s.io/api/batch/v1"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/storage/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/stripe/skycfg"
)

var (
	dryRun    = flag.Bool("dry_run", false, "Dry-run mode.")
	namespace = flag.String("namespace", "default", "Namespace to create/delete objects in.")
)

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

const k8sAPIPrefix = "k8s.io.api."

// gvkFromMessageType assembles schema.GroupVersionKind based on Protobuf
// message type.
// e.g "k8s.io.api.core.v1.Pod" turned into "GV=v1, Kind=Pod".
// Panics if message is not prefix with k8sAPIPerfix.
func gvkFromMessageType(m proto.Message) schema.GroupVersionKind {
	t := gogo_proto.MessageName(m)
	if !strings.HasPrefix(t, k8sAPIPrefix) {
		panic("Unexpected message type: " + t)
	}
	ss := strings.Split(t[len(k8sAPIPrefix):], ".")
	if ss[0] == "core" { // Is there a better way?
		ss[0] = ""
	}
	return schema.GroupVersionKind{
		Group:   ss[0],
		Version: ss[1],
		Kind:    ss[2],
	}
}

// k8sRun creates or deletes json object for gvk in namespace.
// Returns object name or error.
func k8sRun(gvk schema.GroupVersionKind, namespace string, json string, delete bool) (string, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return "", fmt.Errorf("failed to build rest config: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return "", fmt.Errorf("failed to initialize dynamic REST client: %v", err)
	}

	dc, err := discovery.NewCachedDiscoveryClientForConfig(restConfig, "", "", 1*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to initialize discovery client: %v", err)
	}
	rm, err := restmapper.NewDeferredDiscoveryRESTMapper(dc).RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", fmt.Errorf("failed to obtain a REST mapping: %v", err)
	}

	obj := runtime.Unknown{
		Raw: []byte(json),
	}
	//obj.SetGroupVersionKind(gvk)
	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
	if err != nil {
		return "", err
	}
	un := &unstructured.Unstructured{objMap}

	r := dynamicClient.Resource(rm.Resource).Namespace(namespace)
	if delete {
		err = r.Delete(un.GetName(), nil)
	} else {
		_, err = r.Create(un, meta_v1.CreateOptions{})
	}
	if err != nil {
		return "", fmt.Errorf("failed to mutate resource: %v", err)
	}
	return un.GetName(), err
}

func main() {
	flag.Parse()
	argv := flag.Args()

	if len(argv) != 2 {
		fmt.Fprintf(os.Stderr, `Demo Kubernetes CLI for Skycfg, a library for building complex typed configs.

usage: %s [ up | down ] FILENAME
`, os.Args[0], os.Args[0])
		os.Exit(1)
	}

	action, filename := argv[0], argv[1]

	config, err := skycfg.Load(context.Background(), filename, skycfg.WithProtoRegistry(&protoRegistry{}))
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
		marshaled, err := jsonMarshaler.MarshalToString(msg)
		sep := ""
		if err != nil {
			fmt.Fprintf(os.Stderr, "json.Marshal: %v\n", err)
			continue
		}

		if *dryRun {
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
			return
		}

		switch action {
		case "up":
			if n, err := k8sRun(gvkFromMessageType(msg), *namespace, marshaled, false); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			} else {
				fmt.Printf("%s/%s created.\n", *namespace, n)
			}
		case "down":
			if n, err := k8sRun(gvkFromMessageType(msg), *namespace, marshaled, true); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			} else {
				fmt.Printf("%s/%s deleted.\n", *namespace, n)
			}
		default:
			panic("Unknown action: " + action)
		}

	}
}
