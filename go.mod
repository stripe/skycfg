module github.com/stripe/skycfg

go 1.15

require (
	github.com/cncf/udpa/go v0.0.0-20200909154343-1f710aca26a9 // indirect
	github.com/envoyproxy/go-control-plane v0.9.6 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/kylelemons/godebug v0.0.0-20170820004349-d65d576e9348
	go.starlark.net v0.0.0-20190604130855-6ddc71c0ba77
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.1
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0 // indirect
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800 // indirect
)

replace github.com/kylelemons/godebug => github.com/jmillikin-stripe/godebug v0.0.0-20180620173319-8279e1966bc1
