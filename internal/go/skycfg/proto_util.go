package skycfg

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
	descriptor_pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func mustParseFileDescriptor(gzBytes []byte) *descriptor_pb.FileDescriptorProto {
	gz, err := gzip.NewReader(bytes.NewReader(gzBytes))
	if err != nil {
		panic(fmt.Sprintf("EnumDescriptor: %v", err))
	}
	defer gz.Close()

	fileDescBytes, err := ioutil.ReadAll(gz)
	if err != nil {
		panic(fmt.Sprintf("EnumDescriptor: %v", err))
	}

	fileDesc := &descriptor_pb.FileDescriptorProto{}
	if err := proto.Unmarshal(fileDescBytes, fileDesc); err != nil {
		panic(fmt.Sprintf("EnumDescriptor: %v", err))
	}
	return fileDesc
}

func messageTypeName(msg proto.Message) string {
	if hasName, ok := msg.(interface {
		XXX_MessageName() string
	}); ok {
		return hasName.XXX_MessageName()
	}

	hasDesc, ok := msg.(descriptor.Message)
	if !ok {
		return proto.MessageName(msg)
	}

	gzBytes, path := hasDesc.Descriptor()
	fileDesc := mustParseFileDescriptor(gzBytes)
	var chunks []string
	if pkg := fileDesc.GetPackage(); pkg != "" {
		chunks = append(chunks, pkg)
	}

	msgDesc := fileDesc.MessageType[path[0]]
	for ii := 1; ii < len(path); ii++ {
		chunks = append(chunks, msgDesc.GetName())
		msgDesc = msgDesc.NestedType[path[ii]]
	}
	chunks = append(chunks, msgDesc.GetName())
	return strings.Join(chunks, ".")
}
