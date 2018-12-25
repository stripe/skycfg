module github.com/stripe/skycfg

require (
	github.com/gogo/protobuf v1.1.1
	github.com/golang/protobuf v1.2.0
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/kylelemons/godebug v0.0.0-20170820004349-d65d576e9348
	go.starlark.net v0.0.0-20181108041844-f4938bde4080
	golang.org/x/net v0.0.0-20180911220305-26e67e76b6c3 // indirect
	golang.org/x/sync v0.0.0-20180314180146-1d60e4601c6f // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.1
	k8s.io/apimachinery v0.0.0-20181222072933-b814ad55d7c5
)

replace github.com/kylelemons/godebug => github.com/jmillikin-stripe/godebug v0.0.0-20180620173319-8279e1966bc1
