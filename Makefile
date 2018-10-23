# Using https://github.com/golang/go/wiki/Modules
export GO111MODULE = on

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GODOC=$(GOCMD)doc
GOCLEAN=$(GOCMD) clean
PROTOC=protoc

all: build

build:
	$(GOBUILD) ./...

test: test_deps
	mkdir -p test_proto
	$(PROTOC) --go_out=paths=source_relative:test_proto --proto_path=testdata test_proto_v2.proto test_proto_v3.proto
	$(PROTOC) --gogofast_out=paths=source_relative:test_proto --proto_path=testdata test_proto_gogo.proto
	$(GOTEST) -v ./...

test_deps:
	$(GOGET) github.com/golang/protobuf/protoc-gen-go
	$(GOGET) github.com/gogo/protobuf/protoc-gen-gogofast

docs:
	$(GOGET) golang.org/x/tools/cmd/godoc
	@echo
	@echo "Go to http://localhost:8080/pkg/github.com/stripe/skycfg"
	@echo
	$(GODOC) -http 'localhost:8080'

clean:
	$(GOCLEAN)
