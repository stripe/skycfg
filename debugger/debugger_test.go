package debugger_test

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/stripe/skycfg/debugger"
)

var runExample = flag.Bool("example", false, "run debugger on example program")

func TestMain(m *testing.M) {
	flag.Parse()
	if *runExample {
		Example()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type (
	fakeMapValue    map[string]string
	fakeFuncValue   func()
	fakeStructValue struct {
		f func()
		a int
	}
)

func TestSafeEq(t *testing.T) {
	t.Parallel()

	t.Run("Map", func(t *testing.T) {
		t.Parallel()

		a, b, nilMap := fakeMapValue{}, fakeMapValue{}, fakeMapValue(nil)
		assert.False(t, SafeEq(a, b))
		assert.True(t, SafeEq(a, a))
		assert.False(t, SafeEq(a, nilMap))
	})

	t.Run("Func", func(t *testing.T) {
		t.Parallel()

		f := func() {}
		a, b, nilFunc := fakeFuncValue(f), fakeFuncValue(f), fakeFuncValue(nil)
		assert.True(t, SafeEq(a, b))
		assert.True(t, SafeEq(a, a))
		assert.False(t, SafeEq(a, nilFunc))
		// Remember SafeEq errs on the side of returning true, so don't test anything more complex than this.
	})

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()

		f := func() {}
		a, b, nilFunc := fakeStructValue{f: f, a: 1}, fakeStructValue{f: f, a: 2}, fakeStructValue{f: nil, a: 1}
		assert.False(t, SafeEq(a, b))
		assert.True(t, SafeEq(a, a))
		assert.False(t, SafeEq(a, nilFunc))
	})
}
