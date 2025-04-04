module github.com/stripe/skycfg/_examples/wasm

go 1.24

require (
	github.com/golang/protobuf v1.5.0
	github.com/stripe/skycfg v0.0.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v2 v2.2.1
)

replace github.com/stripe/skycfg => ../../
