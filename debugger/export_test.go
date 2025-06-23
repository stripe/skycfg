package debugger

// Export functions for testing.

import (
	"reflect"
)

func SafeEq(a, b any) bool {
	return safeEq(reflect.ValueOf(a), reflect.ValueOf(b))
}
