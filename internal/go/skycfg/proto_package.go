package skycfg

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"go.starlark.net/starlark"
)

// NewProtoPackage creates a Starlark value representing a named Protobuf package.
//
// Protobuf packagess are conceptually similar to a C++ namespace or Ruby
// module, in that they're aggregated from multiple .proto source files.
func newProtoPackage(registry ProtoRegistry, name string) starlark.Value {
	return &skyProtoPackage{
		registry: registry,
		name:     name,
	}
}

type skyProtoPackage struct {
	registry ProtoRegistry
	name     string
}

func (pkg *skyProtoPackage) String() string       { return fmt.Sprintf("<proto.Package %q>", pkg.name) }
func (pkg *skyProtoPackage) Type() string         { return "proto.Package" }
func (pkg *skyProtoPackage) Freeze()              {}
func (pkg *skyProtoPackage) Truth() starlark.Bool { return starlark.True }
func (pkg *skyProtoPackage) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", pkg.Type())
}

func (pkg *skyProtoPackage) AttrNames() []string {
	// TODO: Implement when go-protobuf gains support for listing the
	// registered message types in a Protobuf package.
	//
	// https://github.com/golang/protobuf/issues/623
	return nil
}

func (pkg *skyProtoPackage) Attr(attrName string) (starlark.Value, error) {
	fullName := fmt.Sprintf("%s.%s", pkg.name, attrName)
	if ev := proto.EnumValueMap(fullName); ev != nil {
		return &skyProtoEnumType{
			name:     fullName,
			valueMap: ev,
		}, nil
	}
	return newMessageType(pkg.registry, fullName)
}
