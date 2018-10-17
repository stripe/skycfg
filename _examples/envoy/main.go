package main

import (
	"fmt"

	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	_ "github.com/stripe/skycfg"
)

func main() {
	fmt.Println("Done!")
}
