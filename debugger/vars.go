package debugger

import (
	"fmt"

	"go.starlark.net/starlark"
)

func (s *debugSession) makeGlobals() *starlark.Builtin {
	return starlark.NewBuiltin("globals", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// We have no args to unpack. Make sure no one tried to set any.
		err := starlark.UnpackArgs(fn.Name(), args, kwargs)
		if err != nil {
			return nil, err
		}

		frame := s.currentFrame()
		starFunc, ok := frame.Callable().(*starlark.Function)
		if !ok {
			fmt.Printf("globals: Warning: No globals visible in non-Starlark function (%s) %v\n", frame.Callable().Type(), frame.Callable())
			return new(starlark.Dict), nil
		}

		globals := starFunc.Globals()
		globalsDict := starlark.NewDict(len(starFunc.Globals()))
		for name, val := range globals {
			if val != nil {
				globalsDict.SetKey(starlark.String(name), val)
			}
		}
		return globalsDict, nil
	})
}

func (s *debugSession) makeLocals() *starlark.Builtin {
	return starlark.NewBuiltin("locals", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// We have no args to unpack. Make sure no one tried to set any.
		err := starlark.UnpackArgs(fn.Name(), args, kwargs)
		if err != nil {
			return nil, err
		}

		frame := s.currentFrame()
		if _, ok := frame.Callable().(*starlark.Function); !ok {
			fmt.Printf("locals: Warning: No locals visible in non-Starlark function (%s) %v\n", frame.Callable().Type(), frame.Callable())
			return new(starlark.Dict), nil
		}

		localsDict := starlark.NewDict(frame.NumLocals())
		for i := 0; i < frame.NumLocals(); i++ {
			binding, val := frame.Local(i)
			if val != nil {
				localsDict.SetKey(starlark.String(binding.Name), val)
			}
		}
		return localsDict, nil
	})
}

func (s *debugSession) makeVars() *starlark.Builtin {
	return starlark.NewBuiltin("vars", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// We have no args to unpack. Make sure no one tried to set any.
		err := starlark.UnpackArgs(fn.Name(), args, kwargs)
		if err != nil {
			return nil, err
		}

		frame := s.currentFrame()
		starFunc, ok := frame.Callable().(*starlark.Function)
		if !ok {
			fmt.Printf("No variables visible in non-Starlark function (%s) %v\n", frame.Callable().Type(), frame.Callable())
			return starlark.None, nil
		}

		fmt.Printf("All predeclared symbols in frame %d (%s):\n", s.currentDepth, starFunc.Name())
		for name, val := range getPredeclared(starFunc) {
			if val == nil {
				fmt.Printf("  %s = <uninitialized>\n", name)
			} else {
				fmt.Printf("  %s: %s = %s\n", name, val.Type(), shortString(val))
			}
		}
		fmt.Println()

		fmt.Printf("All globals in frame %d (%s):\n", s.currentDepth, starFunc.Name())
		for name, val := range starFunc.Globals() {
			if val == nil {
				fmt.Printf("  %s = <uninitialized>\n", name)
			} else {
				fmt.Printf("  %s: %s = %s\n", name, val.Type(), shortString(val))
			}
		}
		fmt.Println()

		fmt.Printf("All free vars in frame %d (%s):\n", s.currentDepth, starFunc.Name())
		for i := 0; i < starFunc.NumFreeVars(); i++ {
			binding, val := starFunc.FreeVar(i)
			if val == nil {
				fmt.Printf("  %s (%v) = <uninitialized>\n", binding.Name, binding.Pos)
			} else {
				fmt.Printf("  %s (%v): %s = %s\n", binding.Name, binding.Pos, val.Type(), shortString(val))
			}
		}
		fmt.Println()

		fmt.Printf("All locals in frame %d (%s):\n", s.currentDepth, starFunc.Name())
		for i := 0; i < frame.NumLocals(); i++ {
			binding, val := frame.Local(i)
			if val == nil {
				fmt.Printf("  %s (%v) = <uninitialized>\n", binding.Name, binding.Pos)
			} else {
				fmt.Printf("  %s (%v): %s = %s\n", binding.Name, binding.Pos, val.Type(), shortString(val))
			}
		}

		return starlark.None, nil
	})
}

func shortString(val starlark.Value) string {
	if val == nil {
		return "<uninitialized>"
	}
	s := val.String()
	if len(s) < 70 {
		return s
	}
	return s[:38] + "…" + s[len(s)-29:]
}
