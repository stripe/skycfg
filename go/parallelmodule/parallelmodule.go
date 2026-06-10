// Package parallelmodule contains Starlark functions for running code in parallel.
// Unlike other Starlark modules in skycfg, this module is not loaded by skycfg by default.
package parallelmodule

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
	"golang.org/x/sync/errgroup"
)

// NewModule returns a Starlark module of parallel functions.
//
//	parallel = module(
//	    map,
//	)
//
// def map(function, iterable, limit=-1):
//
// Runs function on each element of iterable in parallel.
// Returns a list containing the result of function, in the same order as the
// original iterable.
// If any call to function fails, parallel.map fails as well.
// If multiple invocations fail, the error is chosen arbitrarily.
// If limit is provided and positive, at most limit goroutines run concurrently.
//
// To guarantee safety, the following measures are taken:
//   - The function and each element of iterable are frozen.
//   - map cannot be called during module initialization.
//     See https://github.com/google/starlark-go/issues/623.
//   - map detects and forbids recursion, even if Starlark is configured to
//     allow recursion.
var Module = &starlarkstruct.Module{
	Name: "parallel",
	Members: starlark.StringDict{
		"map": starlark.NewBuiltin("parallel.map", parMap),
	},
}

// makeTrampoline creates a function appropriate for [starlark.NewBuiltin] that
// just calls the provided callable with args and wargs.
func makeTrampoline(callable starlark.Callable) func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return func(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.Call(t, callable, args, kwargs)
	}
}

// callableWithPos is a callable that always has the given position in a stack trace.
type callableWithPos struct {
	starlark.Callable
	pos syntax.Position
}

func (f callableWithPos) Position() syntax.Position {
	return f.pos
}

// replicateStack creates a Starlark callable that, when called, adds the call
// frames of t to the call stack, and then calls fn.
func replicateStack(t *starlark.Thread, fn starlark.Callable) starlark.Callable {
	for depth := range t.CallStackDepth() {
		fr := t.CallFrame(depth)
		builtin := starlark.NewBuiltin(fr.Name, makeTrampoline(fn))
		fn = callableWithPos{builtin, fr.Pos}
	}
	return fn
}

func checkRecursionOrInitialization(t *starlark.Thread, fn *starlark.Builtin) error {
	seenPos := map[syntax.Position]struct{}{}
	for depth := range t.CallStackDepth() {
		fr := t.CallFrame(depth)

		// We cannot guarantee data race freedom during module initialization,
		// because freezing a function does not prevent a global from being
		// modified or reassigned (if -globalreassign).
		// See https://github.com/google/starlark-go/issues/623.
		//
		// Merely freezing the mapper's globals is **not** sufficient:
		// the iterable can also contain functions with access to globals.
		// See TestMap_mutate_global_during_toplevel2.
		//
		// On the other hand, if a module is fully initialized, race freedom is
		// guaranteed:
		//  1. Globals can never be reassigned by non-top-level Starlark functions.
		//  2. Global values must have been frozen already at the end of module
		//     initialization.
		if fr.Name == "<toplevel>" {
			return fmt.Errorf("function %s cannot be called during module top-level initialization", fn.Name())
		}

		// This is not entirely foolproof, since:
		//  1. we only check for cross-thread recursion if a parallel function
		//     is called;
		//  2. checkRecursion checks frame positions, not function code identity
		//     like Starlark itself.
		//
		// But in practice this is good enough to guarantee no infinite loop.
		// See TestMap_recurse_benign for a code sample that is allowed under this
		// check, but not by Starlark in single-threaded execution.
		if fr.Pos.Filename() == "<builtin>" {
			// built-in function; ignore
			continue
		}
		if _, ok := seenPos[fr.Pos]; ok {
			return fmt.Errorf("function %s called recursively", fr.Name)
		}
		seenPos[fr.Pos] = struct{}{}
	}
	return nil
}

// par implements part of the function:
//
//	def par(callable, iterable):
//
// It calls callable on each element of iterable, and returns a slice of result
// values.
func par(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) ([]starlark.Value, error) {
	fnName := fn.Name()

	var mapper starlark.Callable
	var iterable starlark.Iterable
	limit := -1
	if err := starlark.UnpackArgs(fnName, args, kwargs, "function", &mapper, "iterable", &iterable, "limit?", &limit); err != nil {
		return nil, err
	}

	if err := checkRecursionOrInitialization(t, fn); err != nil {
		return nil, err
	}

	// Important: freeze inputs to prevent data races.
	mapper.Freeze()
	inputs := toFrozenSlice(iterable)

	out := make([]starlark.Value, len(inputs))

	var eg errgroup.Group
	eg.SetLimit(limit)
	for i, val := range inputs {
		curThread, err := newThread(t, fn.Name(), i)
		if err != nil {
			return nil, err
		}

		var mapCaller starlark.Callable

		// Create a trampoline function with a descriptive name for nice stack traces.
		mapCaller = starlark.NewBuiltin(fmt.Sprintf("%s.fn[%d]", fnName, i), makeTrampoline(mapper))

		// Replicate the stack of the parent thread.
		// This is important for two reasons:
		//   1. It makes checkRecursion work across threads.
		//   2. In case of evaluation error, the captured stack trace will
		//      include the stack from the parent thread.
		mapCaller = replicateStack(t, mapCaller)

		eg.Go(func() error {
			mappedVal, err := starlark.Call(curThread, mapCaller, starlark.Tuple{val}, nil)
			if err != nil {
				return err
			}
			out[i] = mappedVal
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func parMap(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	vals, err := par(t, fn, args, kwargs)
	if err != nil {
		return nil, err
	}
	return starlark.NewList(vals), nil
}

// toFrozenSlice converts an iterable to a slice of frozen values.
func toFrozenSlice(seq starlark.Iterable) (out []starlark.Value) {
	iter := seq.Iterate()
	defer iter.Done()
	var elem starlark.Value
	for iter.Next(&elem) {
		elem.Freeze()
		out = append(out, elem)
	}
	return out
}
