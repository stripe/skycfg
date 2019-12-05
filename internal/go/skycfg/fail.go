package skycfg

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

var Fail = starlark.NewBuiltin("fail", failImpl)

type FailError struct {
	pos       syntax.Position
	msg       string
	callStack starlark.CallStack
}

func NewFailError(msg string, callStack starlark.CallStack) *FailError {
	return &FailError{
		pos:       callStack.At(0).Pos,
		msg:       msg,
		callStack: callStack,
	}
}

func (err *FailError) Error() string {
	return fmt.Sprintf("[%s] %s\n%s", err.pos, err.msg, err.callStack.String())
}

func failImpl(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg string
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &msg); err != nil {
		return nil, err
	}
	callStack := t.CallStack()
	callStack.Pop()
	return nil, NewFailError(msg, callStack)
}
