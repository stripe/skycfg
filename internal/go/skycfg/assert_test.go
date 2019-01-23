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
	"fmt"
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type assertTestCase interface {
	ExpFailure() bool
	ExpFailureMsg() string
	ExpError() bool
	ExpErrorMsg() string
}

type assertTestCaseImpl struct {
	expFailure    bool
	expFailureMsg string
	expError      bool
	expErrorMsg   string
}

func (a assertTestCaseImpl) ExpFailure() bool      { return a.expFailure }
func (a assertTestCaseImpl) ExpFailureMsg() string { return a.expFailureMsg }
func (a assertTestCaseImpl) ExpError() bool        { return a.expError }
func (a assertTestCaseImpl) ExpErrorMsg() string   { return a.expErrorMsg }

type assertBinaryTestCase struct {
	assertTestCaseImpl
	op      syntax.Token
	val1Str string
	val2Str string
}

type assertUnaryTestCase struct {
	assertTestCaseImpl
	val string
}

func TestUnaryAsserts(t *testing.T) {
	testCases := []assertUnaryTestCase{
		assertUnaryTestCase{
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure: false,
				expError:   false,
			},
			val: `1 == 1`,
		},
		assertUnaryTestCase{
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:    true,
				expFailureMsg: "assertion failed",
				expError:      false,
			},
			val: `2 == 1`,
		},
	}

	for _, testCase := range testCases {
		cmd := fmt.Sprintf(
			`t.assert.true(%s)`,
			testCase.val,
		)

		evalAndReportResults(t, cmd, testCase)
	}
}

func TestBinaryAsserts(t *testing.T) {
	testCases := []assertBinaryTestCase{
		assertBinaryTestCase{
			op:      syntax.EQL,
			val1Str: `"hello"`,
			val2Str: `"hello"`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure: false,
				expError:   false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.EQL,
			val1Str: `"hello"`,
			val2Str: `"nothello"`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:    true,
				expFailureMsg: `"hello" (type: string) == "nothello" (type: string)`,
				expError:      false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.NEQ,
			val1Str: `"hello"`,
			val2Str: `"hello"`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:    true,
				expFailureMsg: `"hello" (type: string) != "hello" (type: string)`,
				expError:      false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.NEQ,
			val1Str: `"6"`,
			val2Str: `6`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure: false,
				expError:   false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.LT,
			val1Str: "3",
			val2Str: "5",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure: false,
				expError:   false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.LT,
			val1Str: "5",
			val2Str: "5",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:    true,
				expFailureMsg: "5 (type: int) < 5 (type: int)",
				expError:      false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.LT,
			val1Str: "5",
			val2Str: `"5"`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:  false,
				expError:    true,
				expErrorMsg: "int < string not implemented",
			},
		},
		assertBinaryTestCase{
			op:      syntax.LE,
			val1Str: "5",
			val2Str: "5",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure: false,
				expError:   false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.LE,
			val1Str: "5",
			val2Str: "4",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:    true,
				expFailureMsg: "5 (type: int) <= 4 (type: int)",
				expError:      false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.LE,
			val1Str: "5",
			val2Str: `"5"`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:  false,
				expError:    true,
				expErrorMsg: "int <= string not implemented",
			},
		},
		assertBinaryTestCase{
			op:      syntax.GT,
			val1Str: "5",
			val2Str: "3",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure: false,
				expError:   false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.GT,
			val1Str: "5",
			val2Str: "5",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:    true,
				expFailureMsg: "5 (type: int) > 5 (type: int)",
				expError:      false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.GT,
			val1Str: "5",
			val2Str: `"5"`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:  false,
				expError:    true,
				expErrorMsg: "int > string not implemented",
			},
		},
		assertBinaryTestCase{
			op:      syntax.GE,
			val1Str: "5",
			val2Str: "5",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure: false,
				expError:   false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.GE,
			val1Str: "4",
			val2Str: "5",
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:    true,
				expFailureMsg: "4 (type: int) >= 5 (type: int)",
				expError:      false,
			},
		},
		assertBinaryTestCase{
			op:      syntax.GE,
			val1Str: "5",
			val2Str: `"5"`,
			assertTestCaseImpl: assertTestCaseImpl{
				expFailure:  false,
				expError:    true,
				expErrorMsg: "int >= string not implemented",
			},
		},
	}

	for _, testCase := range testCases {
		cmd := fmt.Sprintf(
			`t.assert.%s(%s, %s)`,
			tokenToString[testCase.op],
			testCase.val1Str,
			testCase.val2Str,
		)
		evalAndReportResults(t, cmd, testCase)
	}
}

func TestMultipleAssertionErrors(t *testing.T) {
	thread := new(starlark.Thread)
	failureCtx, assertModule := AssertModule()

	env := starlark.StringDict{
		"assert": assertModule,
	}

	_, err := starlark.Eval(
		thread,
		"<expr>",
		"assert.eql(1, 2)",
		env,
	)
	if err == nil {
		t.Error("Failing an assertion should return an error, but code completed successfully")
	}
	if len(failureCtx.Failures) != 1 {
		t.Errorf("Expected 1 assertion failure, but found %d", len(failureCtx.Failures))
	}
	_, err = starlark.Eval(
		thread,
		"<expr>",
		"assert.eql(1, 2)",
		env,
	)
	if err == nil {
		t.Error("Failing an assertion should return an error, but code completed successfully")
	}
	if len(failureCtx.Failures) != 2 {
		t.Errorf("Expected 2 assertion failures, but found %d", len(failureCtx.Failures))
	}
}

func evalAndReportResults(t *testing.T, cmd string, testCase assertTestCase) {
	thread := new(starlark.Thread)
	failureCtx, assertModule := AssertModule()

	// set it up like it would be used, off a param
	testCtx := &Module{
		Name: "skycfg_test_ctx",
		Attrs: starlark.StringDict(map[string]starlark.Value{
			"assert": assertModule,
		}),
	}
	env := starlark.StringDict{
		"t": testCtx,
	}

	_, err := starlark.Eval(
		thread,
		"<expr>",
		cmd,
		env,
	)
	// if it errors and the error was unexpected and the error is not a test failure
	// OR if it did not error and an error was expected
	if (err != nil && !testCase.ExpError() && !testCase.ExpFailure()) || (err == nil && testCase.ExpError()) {
		t.Errorf(
			"Eval execution result (error: %t) did not equal expected execution result (error: %t). cmd: `%s`. err: %s",
			err != nil,
			testCase.ExpError(),
			cmd,
			err,
		)
		return
	}

	if testCase.ExpError() && !strings.Contains(err.Error(), testCase.ExpErrorMsg()) {
		t.Errorf(
			"Expected '%s' to be included in the error message: '%s'",
			testCase.ExpErrorMsg(),
			err.Error(),
		)
	}

	// if the test case errors, theres nothing else to check
	if testCase.ExpError() {
		return
	}

	if (len(failureCtx.Failures) > 0) != testCase.ExpFailure() {
		t.Errorf(
			"Assertion result (%t) did not equal expected assertion result (%t) for `%s`",
			len(failureCtx.Failures) == 0,
			!testCase.ExpFailure(),
			cmd,
		)
		return
	}

	if testCase.ExpFailure() && !strings.Contains(failureCtx.Failures[0].Error(), testCase.ExpFailureMsg()) {
		t.Errorf(
			"Expected '%s' to be included in the failure message: '%s'",
			testCase.ExpFailureMsg(),
			failureCtx.Failures[0].Error(),
		)
	}
}
