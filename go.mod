module github.com/stripe/skycfg

go 1.16

require (
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.1
	github.com/kylelemons/godebug v0.0.0-20170820004349-d65d576e9348
	go.starlark.net v0.0.0-20201204201740-42d4f566359b
	google.golang.org/protobuf v1.25.1-0.20201016220047-aa45c4675289
	gopkg.in/yaml.v2 v2.2.1
)

replace github.com/kylelemons/godebug => github.com/jmillikin-stripe/godebug v0.0.0-20180620173319-8279e1966bc1
