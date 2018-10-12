package skycfg

import (
	"sort"
	"bytes"
	"strings"
	"compress/gzip"
	"io/ioutil"
	"fmt"

	"github.com/google/skylark"
	"github.com/golang/protobuf/proto"
	descriptor_pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type skyProtoEnumType struct {
	name string
	valueMap map[string]int32
}

var _ skylark.HasAttrs = (*skyProtoEnumType)(nil)

func (t *skyProtoEnumType) String() string {
	return fmt.Sprintf("<proto.EnumType %q>", t.name)
}
func (t *skyProtoEnumType) Type() string        { return "proto.EnumType" }
func (t *skyProtoEnumType) Freeze()             {}
func (t *skyProtoEnumType) Truth() skylark.Bool { return skylark.True }
func (t *skyProtoEnumType) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", t.Type())
}

func (t *skyProtoEnumType) Attr(attrName string) (skylark.Value, error) {
	if value, ok := t.valueMap[attrName]; ok {
		return &skyProtoEnumValue{t.name, attrName, value}, nil
	}
	return nil, nil
}

func (t *skyProtoEnumType) AttrNames() []string {
	names := make([]string, 0, len(t.valueMap))
	for name := range t.valueMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type skyProtoEnumValue struct {
	typeName string
	valueName string
	value int32
}

func (v *skyProtoEnumValue) String() string {
	return fmt.Sprintf("<%s %s=%d>", v.typeName, v.valueName, v.value)
}
func (v *skyProtoEnumValue) Type() string        { return v.typeName }
func (v *skyProtoEnumValue) Freeze()             {}
func (v *skyProtoEnumValue) Truth() skylark.Bool { return skylark.True }
func (v *skyProtoEnumValue) Hash() (uint32, error) {
	return skylark.MakeInt64(int64(v.value)).Hash()
}

// Interface for generated enum types.
type protoEnum interface {
	String() string
	EnumDescriptor() ([]byte, []int)
}

func enumDescriptor(enum protoEnum) (*descriptor_pb.FileDescriptorProto, []int) {
	gzBytes, path := enum.EnumDescriptor()
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
	return fileDesc, path
}

func enumTypeName(enum protoEnum) string {
	fileDesc, path := enumDescriptor(enum)
	var chunks []string
	if pkg := fileDesc.GetPackage(); pkg != "" {
		chunks = append(chunks, pkg)
	}

	if len(path) == 1 {
		enumType := fileDesc.EnumType[path[0]]
		chunks = append(chunks, enumType.GetName())
	} else {
		msgDesc := fileDesc.MessageType[path[0]]
		for ii := 1; ii < len(path) - 1; ii++ {
			chunks = append(chunks, msgDesc.GetName())
			msgDesc = msgDesc.NestedType[path[ii]]
		}
		enumType := msgDesc.EnumType[path[len(path)-1]]
		chunks = append(chunks, msgDesc.GetName(), enumType.GetName())
	}
	return strings.Join(chunks, ".")
}
