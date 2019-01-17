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

package skycfg_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"go.starlark.net/starlark"

	"github.com/stripe/skycfg"
	pb "github.com/stripe/skycfg/test_proto"
)

var testFiles = map[string]string{
	"test1.sky": `
load("test2.sky", "helper2")

test_proto = proto.package("skycfg.test_proto")

def helper1():
	s = struct(x = 12345)
	return s.x

def test_helper1(t):
	x = helper1()
	t.assert(x == 12345)

def main(ctx):
	msg = test_proto.MessageV2()
	msg.f_int64 = helper1()
	msg.f_string = json.marshal(helper2(ctx))

	return [msg]
`,
	"test2.sky": `
load("test3.sky", "helper3")

def helper2(ctx):
	result = helper3(ctx)

	result["key4"] = {
		"key5": "value5",
		"var_key": ctx.vars["var_key"],
	}

	return result

def test_helper2(t):
	ctx = struct(vars = {
		"var_key": "var_value",
	})
	result = helper2(ctx)
	t.assert(result["key4"]["key5"] == "value5")

def test_helper2_fails(t):
	ctx = struct(vars = {
		"var_key": "var_value",
	})
	result = helper2(ctx)
	t.assert(result["key4"]["key5"] == "value6")

def test_helper2_errors(t):
	t.someundefinedfunc()
`,
	"test3.sky": `
def helper3(ctx):
	return {
		"key1": "value1",
		"key2": url.encode_query({"key3": "value3"}),
	}
`,
	"test4.sky": `
# Bad load
load("non_existent_file.sky", "test_func")

def main(ctx):
	return []
`,
	"test5.sky": `
# Syntax error detected on load
print(non_existent_var)

def main(ctx):
	return []
`,
	"test6.sky": `
# Main does not return protos
def main(ctx):
	return ["str1", "str2"]
`,
}

// testLoader is a simple loader that loads files from the testFiles map.
type testLoader struct{}

func (loader *testLoader) Resolve(ctx context.Context, name, fromPath string) (string, error) {
	return name, nil
}

func (loader *testLoader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if source, ok := testFiles[path]; ok {
		return []byte(source), nil
	}
	return nil, fmt.Errorf("File %s not found", path)
}

type endToEndTestCase struct {
	caseName   string
	fileToLoad string
	vars       starlark.StringDict
	expLoadErr bool
	expExecErr bool
	expProtos  []proto.Message
}

func TestSkycfgEndToEnd(t *testing.T) {
	loader := &testLoader{}
	ctx := context.Background()

	testCases := []endToEndTestCase{
		endToEndTestCase{
			caseName:   "all good",
			fileToLoad: "test1.sky",
			vars: starlark.StringDict{
				"var_key": starlark.String("var_value"),
			},
			expProtos: []proto.Message{
				&pb.MessageV2{
					FInt64: proto.Int64(12345),
					FString: proto.String(
						`{"key1": "value1", "key2": "key3=value3", "key4": {"key5": "value5", "var_key": "var_value"}}`,
					),
				},
			},
		},
		endToEndTestCase{
			caseName:   "bad load target",
			fileToLoad: "test4.sky",
			expLoadErr: true,
		},
		endToEndTestCase{
			caseName:   "syntax error on load",
			fileToLoad: "test5.sky",
			expLoadErr: true,
		},
		endToEndTestCase{
			caseName:   "return non-protos",
			fileToLoad: "test6.sky",
			expExecErr: true,
		},
	}

	for _, testCase := range testCases {
		config, err := skycfg.Load(ctx, testCase.fileToLoad, skycfg.WithFileReader(loader))
		if testCase.expLoadErr {
			if err == nil {
				t.Error(
					"Bad err result from LoadConfig for case", testCase.caseName,
					"\nExpected non-nil",
					"\nGot", err,
				)
			}

			continue
		} else {
			if err != nil {
				t.Error(
					"Bad err result from LoadConfig for case", testCase.caseName,
					"\nExpected nil",
					"\nGot", err,
				)

				continue
			}
		}

		protos, err := config.Main(context.Background(), skycfg.WithVars(testCase.vars))

		if testCase.expExecErr {
			if err == nil {
				t.Error(
					"Bad err result from ExecMain for case", testCase.caseName,
					"\nExpected nil",
					"\nGot", err,
				)
			}

			continue
		} else {
			if err != nil {
				t.Error(
					"Bad err result from ExecMain for case", testCase.caseName,
					"\nExpected nil",
					"\nGot", err,
				)

				continue
			}
		}

		if !reflect.DeepEqual(protos, testCase.expProtos) {
			t.Error(
				"Wrong protos result from ExecMain for case", testCase.caseName,
				"\nExpected", testCase.expProtos,
				"\nGot", protos,
			)
		}
	}
}

// testTestCase is a test case for the testing functionality built into skycfg
type testTestCase struct {
	errors     bool
	passes     bool
	failureMsg string // we can't create a skycfg.AssertionError but we can check the message and type
}

func TestSkycfgTesting(t *testing.T) {
	loader := &testLoader{}
	ctx := context.Background()

	config, err := skycfg.Load(ctx, "test1.sky", skycfg.WithFileReader(loader))
	if err != nil {
		t.Error("Unexpected error loading test1.sky", err)
	}

	tests := config.Tests()
	if len(tests) != 4 {
		t.Error("Expected 4 tests but found", len(tests))
	}

	cases := map[string]testTestCase{
		"test_helper1": testTestCase{
			passes: true,
		},
		"test_helper2": testTestCase{
			passes: true,
		},
		"test_helper2_fails": testTestCase{
			passes:     false,
			failureMsg: "assertion failed",
		},
		"test_helper2_errors": testTestCase{
			errors: true,
		},
	}

	for _, test := range tests {
		result, err := test.Run(ctx)
		testCase, ok := cases[test.Name()]
		if !ok {
			t.Error("Could not find test case for test", test.Name())
			continue
		}

		if (err != nil) != testCase.errors {
			t.Errorf(
				"[%s] Execution result (error: %t) did not equal expected execution result (error: %t). err: %s",
				test.Name(),
				err != nil,
				testCase.errors,
				err,
			)
			continue
		}

		// if the execution errors, the result is nil and theres nothing else to check
		if err != nil {
			continue
		}

		if result.Name != test.Name() {
			t.Errorf("TestResult (%s) and Test (%s) should have the same name", result.Name, test.Name())
			continue
		}

		if (result.Failure == nil) != testCase.passes {
			t.Errorf(
				"[%s] Test result (pass: %t) did not equal expected test result (pass: %t)",
				test.Name(),
				result.Failure == nil,
				testCase.passes,
			)
			continue
		}

		if !testCase.passes {
			// check the error message
			if _, ok := result.Failure.(skycfg.AssertionError); !ok {
				t.Errorf(
					"[%s] Test failures should be of type skycfg.AssertionError, but found %#v",
					test.Name(),
					result.Failure,
				)
				continue
			}

			if !strings.Contains(result.Failure.Error(), testCase.failureMsg) {
				t.Errorf(
					"[%s] Expected %s to be in failure message, but instead found %s",
					test.Name(),
					testCase.failureMsg,
					result.Failure.Error(),
				)
				continue
			}
		}
	}
}
