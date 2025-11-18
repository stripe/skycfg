package parallelmodule

import (
	"context"
	"fmt"

	"go.starlark.net/starlark"
)

const (
	contextKey       = "context"
	threadCreatorKey = "parallel.thread_creator" // ThreadCreator

	parentThreadKey = "parallel.parent_thread" // *starlark.Thread
)

type ThreadCreator func(parent *starlark.Thread, parallelFunc string, idx int) *starlark.Thread

func GetThreadCreator(t *starlark.Thread) ThreadCreator {
	if tc, ok := t.Local(threadCreatorKey).(ThreadCreator); ok {
		return tc
	}
	return DefaultThreadCreator
}

func DefaultThreadCreator(parent *starlark.Thread, parallelFunc string, idx int) *starlark.Thread {
	nt := &starlark.Thread{
		Name:  fmt.Sprintf("%s > %s[%d]", parent.Name, parallelFunc, idx),
		Print: parent.Print,
	}
	if ctx, ok := parent.Local(contextKey).(context.Context); ok {
		nt.SetLocal(contextKey, ctx)
	}
	if tc, ok := parent.Local(threadCreatorKey).(ThreadCreator); ok {
		nt.SetLocal(threadCreatorKey, tc)
	}
	return nt
}

func newThread(parent *starlark.Thread, parallelFunc string, idx int) (*starlark.Thread, error) {
	tc := GetThreadCreator(parent)
	nt := tc(parent, parallelFunc, idx)
	nt.SetLocal(parentThreadKey, parent)
	return nt, nil
}
