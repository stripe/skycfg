package debugger

// Export functions for testing.

import (
	"reflect"

	"go.starlark.net/starlark"
)

func GetPredeclared(fn *starlark.Function) starlark.StringDict {
	return getPredeclared(fn)
}

func SafeEq(a, b any) bool {
	return safeEq(reflect.ValueOf(a), reflect.ValueOf(b))
}
