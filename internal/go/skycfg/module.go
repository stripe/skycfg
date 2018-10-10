package skycfg

import (
	"fmt"
	"sort"

	"github.com/google/skylark"
)

// A Skylark module, for namespacing of built-in functions.
type Module struct {
	Name  string
	Attrs skylark.StringDict
}

var _ skylark.HasAttrs = (*Module)(nil)

func (mod *Module) String() string        { return fmt.Sprintf("<module %q>", mod.Name) }
func (mod *Module) Type() string          { return "module" }
func (mod *Module) Freeze()               { mod.Attrs.Freeze() }
func (mod *Module) Truth() skylark.Bool   { return skylark.True }
func (mod *Module) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: %s", mod.Type()) }

func (mod *Module) Attr(name string) (skylark.Value, error) {
	if val, ok := mod.Attrs[name]; ok {
		return val, nil
	}
	return nil, nil
}

func (mod *Module) AttrNames() []string {
	var names []string
	for name := range mod.Attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
