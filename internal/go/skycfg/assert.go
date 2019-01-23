// Copyright 2018 The Skycfg Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package skycfg

import (
	"bytes"
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// AssertModule contains assertion functions.
// The *TestContext returned can be used to track assertion failures.
// assert.* functions from this module will mutate the *TestContext.
// After execution is complete, TestContext.Failures will be non-empty
// if any of the assertions failed, and also contain details about the failures.
func AssertModule() (*TestContext, starlark.Value) {
	ctx := &TestContext{}
	return ctx, &Module{
		Name: "assert",
		Attrs: starlark.StringDict{
			"true": starlark.NewBuiltin("assert.true", ctx.AssertUnaryImpl),

			// names match https://github.com/google/starlark-go/blob/7b3aad4436b8cbd25fda2bd658ed44b3dc2c6dcc/syntax/scan.go#L64
			"lt":  starlark.NewBuiltin("assert.lt", ctx.AssertBinaryImpl(syntax.LT)),
			"gt":  starlark.NewBuiltin("assert.gt", ctx.AssertBinaryImpl(syntax.GT)),
			"ge":  starlark.NewBuiltin("assert.ge", ctx.AssertBinaryImpl(syntax.GE)),
			"le":  starlark.NewBuiltin("assert.le", ctx.AssertBinaryImpl(syntax.LE)),
			"eql": starlark.NewBuiltin("assert.eql", ctx.AssertBinaryImpl(syntax.EQL)),
			"neq": starlark.NewBuiltin("assert.neq", ctx.AssertBinaryImpl(syntax.NEQ)),
		},
	}
}

// assertionError represents a failed assertion
type assertionError struct {
	op        *syntax.Token
	val1      starlark.Value
	val2      starlark.Value
	position  string
	backtrace string
}

func (err assertionError) Error() string {
	// straight boolean assertions like assert.true(false)
	if err.op == nil {
		return fmt.Sprintf("[%s] assertion failed\n%s", err.position, err.backtrace)
	}

	// binary assertions, like assert.eql(1, 2)
	return fmt.Sprintf(
		"[%s] assertion failed: %s (type: %s) %s %s (type: %s)\n%s",
		err.position,
		err.val1.String(),
		err.val1.Type(),
		err.op.String(),
		err.val2.String(),
		err.val2.Type(),
		err.backtrace,
	)
}

// TestContext is keeps track of whether there is a failure during a test execution
type TestContext struct {
	Failures []error
}

// AssertUnaryImpl is the implementation for assert(bool) in tests
func (t *TestContext) AssertUnaryImpl(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var val bool
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &val); err != nil {
		return nil, err
	}

	if !val {
		var buf bytes.Buffer
		thread.Caller().WriteBacktrace(&buf)
		err := assertionError{
			position:  thread.Caller().Position().String(),
			backtrace: buf.String(),
		}
		t.Failures = append(t.Failures, err)
		return nil, err
	}

	return starlark.None, nil
}

// AssertBinaryImpl returns a function that implements comparing binary values in an assertion (i.e. assert_eq(1, 2))
func (t *TestContext) AssertBinaryImpl(op syntax.Token) func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var val1 starlark.Value
		var val2 starlark.Value
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &val1, &val2); err != nil {
			return nil, err
		}

		passes, err := starlark.Compare(op, val1, val2)
		if err != nil {
			var buf bytes.Buffer
			thread.Caller().WriteBacktrace(&buf)
			return nil, err
		}

		if !passes {
			var buf bytes.Buffer
			thread.Caller().WriteBacktrace(&buf)
			err := assertionError{
				op:        &op,
				val1:      val1,
				val2:      val2,
				position:  thread.Caller().Position().String(),
				backtrace: buf.String(),
			}
			t.Failures = append(t.Failures, err)
			return nil, err
		}

		return starlark.None, nil
	}
}

var tokenToString = map[syntax.Token]string{
	syntax.LT:  "lt",
	syntax.GT:  "gt",
	syntax.GE:  "ge",
	syntax.LE:  "le",
	syntax.EQL: "eql",
	syntax.NEQ: "neq",
}
