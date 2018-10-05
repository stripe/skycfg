package skycfg

import (
	"fmt"
	"sort"

	"github.com/google/skylark"
)

// A Skylark module, for namespacing of built-in functions.
type skyModule struct {
	name  string
	attrs skylark.StringDict
}

var _ skylark.HasAttrs = (*skyModule)(nil)

func (mod *skyModule) String() string        { return fmt.Sprintf("<module %q>", mod.name) }
func (mod *skyModule) Type() string          { return "module" }
func (mod *skyModule) Freeze()               { mod.attrs.Freeze() }
func (mod *skyModule) Truth() skylark.Bool   { return skylark.True }
func (mod *skyModule) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: %s", mod.Type()) }

func (mod *skyModule) Attr(name string) (skylark.Value, error) {
	if val, ok := mod.attrs[name]; ok {
		return val, nil
	}
	return nil, nil
}

func (mod *skyModule) AttrNames() []string {
	var names []string
	for name := range mod.attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
