package debugger

import (
	"fmt"

	"go.starlark.net/starlark"
)

func (s *debugSession) switchFrame(fn string, newDepth int) error {
	endFrame := s.ParentThread.CallStackDepth() - 1

	frameIdx := s.startFrame + newDepth
	if frameIdx < s.startFrame || frameIdx > endFrame {
		return fmt.Errorf("provided frame index out of range: %v", newDepth)
	}

	// Check for modifications
	s.checkForModifications(fn, "switching frames")

	// Set frame
	s.currentDepth = newDepth
	s.refreshGlobals()

	// Print message
	frame := s.currentFrame()
	fmt.Printf("breakpoint: Stopped at %v: in %s\n", frame.Position(), frame.Callable().Name())
	return nil
}

func (s *debugSession) makeFrame() *starlark.Builtin {
	return starlark.NewBuiltin("frame", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		idxArg := starlark.MakeInt(1)
		err := starlark.UnpackArgs(fn.Name(), args, kwargs, "idx", &idxArg)
		if err != nil {
			return nil, err
		}

		// Check range
		idx, ok := idxArg.Uint64()
		if !ok || int(idx) < 0 || uint64(int(idx)) != idx {
			return nil, fmt.Errorf("frame: provided index out of range: %v", idxArg)
		}

		err = s.switchFrame(fn.Name(), int(idx))
		if err != nil {
			return nil, fmt.Errorf("frame: %w", err)
		}

		return starlark.None, nil
	})
}

func (s *debugSession) makeUp() *starlark.Builtin {
	return starlark.NewBuiltin("up", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		offsetArg := starlark.MakeInt(1)
		err := starlark.UnpackArgs(fn.Name(), args, kwargs, "idx?", &offsetArg)
		if err != nil {
			return nil, err
		}

		// Check range
		offset, ok := offsetArg.Uint64()
		if !ok || int(offset) < 0 || uint64(int(offset)) != offset {
			return nil, fmt.Errorf("up: provided offset out of range: %v", offsetArg)
		}
		newDepth := s.currentDepth + int(offset)
		if newDepth < 0 {
			return nil, fmt.Errorf("down: provided frame index out of range: %d", newDepth)
		}

		err = s.switchFrame(fn.Name(), newDepth)
		if err != nil {
			return nil, fmt.Errorf("up: %w", err)
		}

		return starlark.None, nil
	})
}

func (s *debugSession) makeDown() *starlark.Builtin {
	return starlark.NewBuiltin("down", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		offsetArg := starlark.MakeInt(1)
		err := starlark.UnpackArgs(fn.Name(), args, kwargs, "idx?", &offsetArg)
		if err != nil {
			return nil, err
		}

		// Check range
		offset, ok := offsetArg.Uint64()
		if !ok || int(offset) < 0 || uint64(int(offset)) != offset {
			return nil, fmt.Errorf("down: provided offset out of range: %v", offsetArg)
		}
		newDepth := s.currentDepth - int(offset)
		if newDepth < 0 {
			return nil, fmt.Errorf("down: provided frame index out of range: %d", newDepth)
		}

		err = s.switchFrame(fn.Name(), newDepth)
		if err != nil {
			return nil, fmt.Errorf("down: %w", err)
		}

		return starlark.None, nil
	})
}
