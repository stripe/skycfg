// Package debugger implements a simple interactive debugger for [starlark-go],
// aka [go.starlark.net/starlark].
//
// The debugger exposes a single breakpoint() function to ordinary Starlark code,
// which starts up an interactive REPL-style debugging session,
// in which users can examine the Starlark thread stack and manipulate values.
//
// Stepping through code is not supported due to restrictions in starlark-go.
// See https://github.com/google/starlark-go/issues/304 for details.
//
// # Enabling and disabling breakpoints
//
// By default, all breakpoints are enabled.
// All breakpoint() calls can be globally disabled by calling [Debugger.GlobalDisable],
// or reenabled with [Debugger.GlobalEnable].
//
// # Debugger commands
//
// The debugger REPL has a few builtin commands implemented as functions:
//
// Basic debugging:
//
//	where(), w()             - prints a backtrace
//	backtrace(), bt()        - prints a backtrace (same as where)
//	continue_(), c(), cont() - exits the debugger and continue Starlark execution
//	quit(), q()              - exits the debugger and fail Starlark execution
//
// Variables:
//
//	vars()    - prints all available variables and their values in the current frame
//	globals() - returns a dict of global variables from the context of the code calling breakpoint()
//	locals()  - returns a dict of local variables from the context of the code calling breakpoint()
//
// Frame manipulation:
//
//	frame(i), f(i) - switches the debugging context to the frame at the given index
//	up(), u()      - switches the debugging context to the upper frame (the caller of the current frame)
//	up(i), u(i)    - switches the debugging context to i'th upper frame (the i'th caller of the current frame)
//	down(), d()    - switches the debugging context to the lower frame (the callee of the current frame)
//	down(i), d(i)  - switches the debugging context to i'th lower frame (the i'th callee of the current frame)
//
// # Concurrency and reentrancy
//
// Since each debugger session takes over the entire [os.Stdin],
// concurrent debugger sessions don't make sense.
// As such, upon entering the debugging session, a lock is taken,
// in order to prevent new debugging sessions from launched,
// and also to wait for previous sessions to finish.
// This lock is shared across all Debuggers.
//
// Reentrant calls to breakpoint() (i.e., calls to breakpoint()
// from within a debugging session) are not supported and will
// be ignored with a warning.
//
// [starlark-go]: https://github.com/google/starlark-go
package debugger // import "github.com/stripe/skycfg/debugger"

import (
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strconv"
	"sync"

	"go.starlark.net/repl"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

var (
	debugLock               sync.Mutex
	threadsWithDebugSession sync.Map // *starlark.Thread -> nil
)

// debugSession includes state that lives for a single breakpoint() call.
type debugSession struct {
	ParentThread *starlark.Thread
	Debugger     *Debugger

	debuggerThread starlark.Thread
	globals        starlark.StringDict
	startFrame     int // the index of caller of breakpoint()
	currentDepth   int // always relative to startFrame

	// The existing variables in the current Starlark frame on parent thread.
	// Used for checking whether any variables got updated
	// at the end of the debug session, or when switching frames.
	currentFrameVars starlark.StringDict

	builtins struct {
		where, cont, quit, frame, up, down, globals, locals, vars *starlark.Builtin
	}
	builtinsMap starlark.StringDict

	// If a user calls continue_(), Starlark execution on debuggerThread would
	// panic with value &continueSentinel.
	// Similar with quit() and &quitSentinel.
	continueSentinel, quitSentinel bool
}

func (s *debugSession) init() {
	s.startFrame = getFrameOffset(s.ParentThread, s.Debugger.breakpoint)

	s.globals = starlark.StringDict{}

	s.debuggerThread.Name = s.ParentThread.Name + "[debug]"
	s.debuggerThread.Print = s.ParentThread.Print
	s.debuggerThread.Load = s.ParentThread.Load

	s.builtins.where = s.makeWhere()
	s.builtins.cont = s.makeContinue()
	s.builtins.quit = s.makeQuit()
	s.builtins.frame = s.makeFrame()
	s.builtins.up = s.makeUp()
	s.builtins.down = s.makeDown()
	s.builtins.globals = s.makeGlobals()
	s.builtins.locals = s.makeLocals()
	s.builtins.vars = s.makeVars()

	s.builtinsMap = starlark.StringDict{
		// pdb uses w(here)
		"w":     s.builtins.where,
		"where": s.builtins.where,
		// gdb uses bt/backtrace
		"bt":        s.builtins.where,
		"backtrace": s.builtins.where,
		// gdb and pdb use c/cont/continue
		"c":         s.builtins.cont,
		"cont":      s.builtins.cont,
		"continue_": s.builtins.cont,
		// gdb and pdb use q/quit
		"q":    s.builtins.quit,
		"quit": s.builtins.quit,
		// Python has globals()
		"globals": s.builtins.globals,
		// Python has locals()
		"locals": s.builtins.locals,
		// Python has vars(), though it does something a bit different
		"vars": s.builtins.vars,
		// gdb uses f/frame
		"f":     s.builtins.frame,
		"frame": s.builtins.frame,
		// pdb uses u/up
		"u":  s.builtins.up,
		"up": s.builtins.up,
		// pdb uses d/down
		"d":    s.builtins.down,
		"down": s.builtins.down,
	}
	s.builtinsMap.Freeze()
}

func (s *debugSession) currentFrameIdx() int {
	return s.startFrame + s.currentDepth
}

func (s *debugSession) currentFrame() starlark.DebugFrame {
	return s.ParentThread.DebugFrame(s.currentFrameIdx())
}

func (s *debugSession) makeWhere() *starlark.Builtin {
	return starlark.NewBuiltin("where", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// We have no args to unpack. Make sure no one tried to set any.
		err := starlark.UnpackArgs(fn.Name(), args, kwargs)
		if err != nil {
			return nil, err
		}

		digits := len(strconv.Itoa(s.ParentThread.CallStackDepth() - 1))

		fmt.Println("Traceback (most recent call last):")
		for frameIdx := s.ParentThread.CallStackDepth() - 1; frameIdx >= s.startFrame; frameIdx-- {
			frame := s.ParentThread.DebugFrame(frameIdx)
			current := "  "
			if frameIdx == s.currentFrameIdx() {
				current = "> "
			}
			fmt.Printf("%v% *d. %v: in %s\n", current, digits, frameIdx-s.startFrame, frame.Position(), frame.Callable().Name())
		}
		return starlark.None, nil
	})
}

func (s *debugSession) makeContinue() *starlark.Builtin {
	return starlark.NewBuiltin("continue_", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// We have no args to unpack. Make sure no one tried to set any.
		err := starlark.UnpackArgs(fn.Name(), args, kwargs)
		if err != nil {
			return nil, err
		}
		// Starlark's repl package doesn't expose a clean way to exit from the REPL, similar to what Ctrl+D does.
		// The only option we have is to panic, to be caught in breakpoint().
		panic(&s.continueSentinel)
	})
}

func (s *debugSession) makeQuit() *starlark.Builtin {
	return starlark.NewBuiltin("quit", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// We have no args to unpack. Make sure no one tried to set any.
		err := starlark.UnpackArgs(fn.Name(), args, kwargs)
		if err != nil {
			return nil, err
		}
		// Starlark's repl package doesn't expose a clean way to exit from the REPL, similar to what Ctrl+D does.
		// The only option we have is to panic, to be caught in breakpoint().
		panic(&s.quitSentinel)
	})
}

func (s *debugSession) refreshGlobals() {
	// First, delete everything in s.globals.
	for key := range s.globals {
		delete(s.globals, key)
	}

	// Add debugger functions.
	maps.Copy(s.globals, s.builtinsMap)

	// Add variables this frame has access to.
	// This should overwrite any debugger function symbol.
	s.populateCurrentFrameVars()
	maps.Copy(s.globals, s.currentFrameVars)
}

// Populate s.currentFrameVars from s.currentFrame().
func (s *debugSession) populateCurrentFrameVars() {
	s.currentFrameVars = starlark.StringDict{}

	frame := s.currentFrame()
	starFunc, ok := frame.Callable().(*starlark.Function)
	if !ok {
		fmt.Printf("Warning: frame for non-Starlark function (%s) %v\n", frame.Callable().Type(), frame.Callable())
		return
	}

	// Add variables the function has access to, in visibility order.

	// Add predeclared symbols.
	for name, val := range starFunc.Module().Predeclared() {
		if val != nil {
			s.currentFrameVars[name] = val
		}
	}
	// Add globals.
	for name, val := range starFunc.Globals() {
		if val != nil {
			s.currentFrameVars[name] = val
		}
	}
	// Add free vars.
	for i := 0; i < starFunc.NumFreeVars(); i++ {
		binding, val := starFunc.FreeVar(i)
		if val != nil {
			s.currentFrameVars[binding.Name] = val
		}
	}
	// Add locals.
	for i := 0; i < frame.NumLocals(); i++ {
		binding, val := frame.Local(i)
		if val != nil {
			s.currentFrameVars[binding.Name] = val
		}
	}
}

// DefaultDebuggerSyntax defines the set of Starlark language features
// available within a debugging session.
var DefaultDebuggerSyntax = syntax.FileOptions{
	Set:               true,
	While:             true,
	TopLevelControl:   true,
	GlobalReassign:    true,
	LoadBindsGlobally: true,
	Recursion:         true,
}

// A Debugger captures shared state across debugging sessions.
// The same Debugger can be used concurrently from multiple starlark.Threads and goroutines,
// though as mentioned in the package documentation, only one debug session can be active across all
// Debuggers at any given point in time.
//
// A new Debugger must be created using [New].
type Debugger struct {
	breakpoint *starlark.Builtin

	// mu protects everything after
	mu         sync.Mutex
	disabled   bool
	syntaxOpts *syntax.FileOptions
}

// New creates a new [Debugger].
func New() *Debugger {
	dbg := &Debugger{}
	dbg.breakpoint = dbg.makeBreakpoint()
	dbg.syntaxOpts = &DefaultDebuggerSyntax
	return dbg
}

// GlobalDisable disables all breakpoints.
func (d *Debugger) GlobalDisable() {
	d.mu.Lock()
	d.disabled = true
	d.mu.Unlock()
}

// GlobalEnable reverses [Debugger.GlobalDisable] and enables all breakpoints.
func (d *Debugger) GlobalEnable() {
	d.mu.Lock()
	d.disabled = false
	d.mu.Unlock()
}

// SetSyntax sets the Starlark language options that can be used in a debugger session.
// If SetSyntax is never called, [DefaultDebuggerSyntax] is used.
func (d *Debugger) SetSyntax(opts *syntax.FileOptions) {
	if opts == nil {
		panic("SetSyntax: nil opts")
	}
	d.mu.Lock()
	d.syntaxOpts = opts
	d.mu.Unlock()
}

// Breakpoint returns the breakpoint() Starlark builtin to be added as a global.
// Breakpoint always returns the same value for the same [Debugger].
func (d *Debugger) Breakpoint() *starlark.Builtin {
	return d.breakpoint
}

func (d *Debugger) makeBreakpoint() *starlark.Builtin {
	return starlark.NewBuiltin("breakpoint", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (val starlark.Value, err error) {
		// We have no args to unpack. Make sure no one tried to set any.
		err = starlark.UnpackArgs(fn.Name(), args, kwargs)
		if err != nil {
			return nil, err
		}

		val, err = starlark.None, nil // set default return value

		// Get all the d fields under d.mu.
		d.mu.Lock()
		disabled, syntaxOpts := d.disabled, d.syntaxOpts
		d.mu.Unlock()

		if disabled {
			return
		}
		if _, loaded := threadsWithDebugSession.LoadOrStore(thread, nil); loaded {
			// loaded means that threadsWithDebugSession already contains thread.
			starlarkPrint(thread, "breakpoint: reentrant breakpoint() call is ignored")
			return
		}
		// Only remove thread from threadsWithDebugSession if it was not already there.
		defer func() {
			threadsWithDebugSession.Delete(thread)
		}()

		if !debugLock.TryLock() {
			starlarkPrint(thread, "breakpoint: concurrent debugging sessions attempted; blocking until other sessions finish")
			debugLock.Lock()
		}
		defer debugLock.Unlock()

		s := debugSession{ParentThread: thread, Debugger: d}
		s.init()
		s.refreshGlobals()

		starlarkPrint(thread, "breakpoint: Stopped")
		starlarkPrint(thread, "Available functions: c[ont[inue_]] (or Ctrl+D), q[uit], w[here]/b[ack]t[race], d[own], u[p], f[rame], vars, globals, locals")

		defer func() {
			if obj := recover(); obj != nil {
				if obj == &s.continueSentinel {
					s.checkForModifications("continue_", "continuing")
					val, err = starlark.None, nil
				} else if obj == &s.quitSentinel {
					val, err = nil, fmt.Errorf("debugger quit")
				} else {
					panic(obj)
				}
			} else {
				// Ctrl+D would go here
				s.checkForModifications("breakpoint", "continuing")
			}
		}()

		// Run the debugger REPL on a separate starlark.Thread to allow using panic().
		repl.REPLOptions(syntaxOpts, &s.debuggerThread, s.globals)

		return
	})
}

func (s *debugSession) checkForModifications(fn, operation string) {
	var modifiedVars []string
	for name, origVal := range s.currentFrameVars {
		newVal := s.globals[name]
		if !safeEq(reflect.ValueOf(origVal), reflect.ValueOf(newVal)) {
			modifiedVars = append(modifiedVars, name)
		}
	}

	if len(modifiedVars) > 0 {
		slices.Sort(modifiedVars)

		plural := ""
		if len(modifiedVars) > 1 {
			plural = "s"
		}
		fmt.Printf("%s: Warning: it seems like you tried to change the value of variable%s '%s'", fn, plural, modifiedVars[0])
		for _, name := range modifiedVars[1:] {
			fmt.Printf(", '%s'", name)
		}
		fmt.Printf(" before %s. Please note that any variable modification in the debugger will not be visible from the original Starlark program\n", operation)
	}
}

// getFrameOffset returns the highest stack frame (i.e., most recent call)
// where callable is the relevant callable.
func getFrameOffset(thread *starlark.Thread, callable starlark.Callable) int {
	startFrame := 0
	for ; startFrame < thread.CallStackDepth(); startFrame++ {
		if thread.DebugFrame(startFrame).Callable() == callable {
			startFrame++
			break
		}
	}
	return startFrame
}

// safeEq compares the equality of a and b without ever panicking.
// In case of doubt, safeEq returns true.
func safeEq(ar, br reflect.Value) bool {
	if ar.Type() != br.Type() {
		return false
	}
	if ar.Comparable() {
		return ar.Equal(br)
	} else if br.Comparable() {
		return br.Equal(ar)
	}

	// map and slice are basically completely comparable, except Go made them
	// artificially uncomparable to prevent people from doing shallow compares
	// when they really meant deep compares.
	//
	// func's UnsafePointer() returns a pointer to the function's code,
	// which _can_ say two functions are equal even when they are not,
	// but can't do the opposite, which is appropriate here given
	// safeEq's bias towards true.
	if ar.Kind() == reflect.Map || ar.Kind() == reflect.Func {
		return ar.UnsafePointer() == br.UnsafePointer()
	}
	if ar.Kind() == reflect.Slice {
		return ar.UnsafePointer() == br.UnsafePointer() && ar.Len() == br.Len() && ar.Cap() == br.Cap()
	}

	// The array/struct itself is not comparable, but it might have comparable parts, like
	// struct{A int; B func()}. If the comparable parts are not equal, the whole array/struct is not equal.
	if ar.Kind() == reflect.Array {
		for i := range ar.Len() {
			ai, bi := ar.Index(i), br.Index(i)
			if !safeEq(ai, bi) {
				return false
			}
		}
		return true
	}
	if ar.Kind() == reflect.Struct {
		for i := range ar.NumField() {
			ai, bi := ar.Field(i), br.Field(i)
			if !safeEq(ai, bi) {
				return false
			}
		}
		return true
	}

	return true
}

func starlarkPrint(thread *starlark.Thread, format string, a ...any) {
	if thread.Print != nil {
		thread.Print(thread, fmt.Sprintf(format, a...))
	} else {
		// If thread.Print is nil, Starlark defaults to fmt.Fprintln(os.Stderr, msg).
		fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
	}
}
