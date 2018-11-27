# Skycfg

https://github.com/golang/go/wiki/Modules

```
$ go version
go version go1.11 darwin/amd64
$ export GO111MODULE=on
$ go build ./...
[...]
$ godoc -http 'localhost:8080'
```

```
$ protoc --go_out="${GOPATH}/src" --proto_path=testdata test_proto_v2.proto test_proto_v3.proto
$ protoc --gogofast_out="${GOPATH}/src" --proto_path=testdata test_proto_gogo.proto
$ go test -v ./...
```
