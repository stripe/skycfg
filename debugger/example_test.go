package debugger_test

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	"github.com/stripe/skycfg/debugger"
)

func Example() {
	// Run this example with
	//
	//	go test -c github.com/stripe/skycfg/debugger && ./debugger.test -example

	prog := `
def g():
    closure = [42]
    def f():
        local_var = 1
        breakpoint()
        print("closure=" + str(closure))
    return f

g()()
`

	dbg := debugger.New()
	thread := starlark.Thread{
		Print: printWithPos,
	}
	starlark.ExecFileOptions(&syntax.FileOptions{}, &thread, "<prog>", prog, starlark.StringDict{
		"predeclared": starlark.String("value"),
		"breakpoint":  dbg.Breakpoint(),
	})
}

func printWithPos(thread *starlark.Thread, msg string) {
	var buf bytes.Buffer
	if thread.CallStackDepth() > 1 {
		pos := thread.CallFrame(1).Pos
		fmt.Fprintf(&buf, "[%v] ", pos)
	}
	fmt.Fprintf(&buf, msg)
	buf.WriteByte('\n')

	io.Copy(os.Stdout, &buf)
}
