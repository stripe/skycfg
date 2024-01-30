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

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/proto"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/stripe/skycfg"
	pb "github.com/stripe/skycfg/internal/testdata/test_proto"
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

def test_main(ctx):
	msg = main(ctx)[0]
	ctx.assert(len(msg.r_string) == 1)
	ctx.assert(msg.r_string[0] == "var_value")

def main(ctx):
	msg = test_proto.MessageV2()
	msg.f_int64 = helper1()
	msg.f_string = json.encode(helper2(ctx))
	msg.r_string.append(ctx.vars["var_key"])

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
	"test7.sky": `
test_proto = proto.package("skycfg.test_proto")

# autoboxing of primitives into wrappers works 
def main(ctx):
	msg = test_proto.MessageV3()
	msg.f_BoolValue = True
	msg.f_StringValue = "something"
	msg.f_DoubleValue = 18
	msg.f_DoubleValue = 3110.4120
	msg.f_Int32Value = 110
	msg.f_Int64Value = 2148483647
	msg.f_BytesValue = "foo/bar/baz"
	msg.f_Uint32Value = 4294967295
	msg.f_Uint64Value = 8294967295
	msg.r_StringValue = ["s1","s2","s3"]
	return [msg]
`,
	"test8.sky": `
test_proto = proto.package("skycfg.test_proto")

# autoboxing but overflow error
def main(ctx):
	msg = test_proto.MessageV3()
	msg.f_Int32Value = 2147483648
	return [msg]
`,
	"test9.sky": `
test_proto = proto.package("skycfg.test_proto")

# autoboxing but not representable as int64 error
def main(ctx):
	msg = test_proto.MessageV3()
	msg.f_Int64Value = 999999999999999999999999999999
	return [msg]
`,
	"test10.sky": `
test_proto = proto.package("skycfg.test_proto")

# autoboxing but not representable as uint64 error
def main(ctx):
	msg = test_proto.MessageV3()
	msg.f_Uint64Value = -243789
	return [msg]
`,
	"test11.sky": `
test_proto = proto.package("skycfg.test_proto")

# autoboxing but not representable as uint32 error
def main(ctx):
	msg = test_proto.MessageV3()
	msg.f_Uint32Value = 4294967296
	return [msg]
`,
	"test12.sky": `
test_proto = proto.package("skycfg.test_proto")

def not_main(ctx):
	msg = test_proto.MessageV2()
	msg.f_int64 = 12345
	msg.f_string = "12345"

	return [msg]
`,
	"test13.sky": `
test_proto = proto.package("skycfg.test_proto")

# nested list
def main(ctx):
	msg = test_proto.MessageV2()
	msg.f_int64 = 11111
	msg.f_string = "11111"

	msg2 = test_proto.MessageV2()
	msg2.f_int64 = 22222
	msg2.f_string = "22222"

	msg3 = test_proto.MessageV2()
	msg3.f_int64 = 33333
	msg3.f_string = "33333"

	msgs23 = [msg2, msg3]
	return [msg, msgs23]
`,
	"test14.sky": `
test_proto = proto.package("skycfg.test_proto")

# nested list
def main(ctx):
	msg = test_proto.MessageV2()
	msg.f_int64 = 11111
	msg.f_string = "11111"

	msg2 = test_proto.MessageV2()
	msg2.f_int64 = 22222
	msg2.f_string = "22222"

	msg3 = test_proto.MessageV2()
	msg3.f_int64 = 33333
	msg3.f_string = "33333"

	msg4 = test_proto.MessageV2()
	msg4.f_int64 = 44444
	msg4.f_string = "44444"

	msgs12 = [msg, msg2]
	msgs34 = [msg3, msg4]
	return [msgs12, msgs34]
`,
	"test15.sky": `
test_proto = proto.package("skycfg.test_proto")

# nested list
def main(ctx):
	msg = test_proto.MessageV2()
	msg.f_int64 = 11111
	msg.f_string = "11111"

	msg2 = test_proto.MessageV2()
	msg2.f_int64 = 22222
	msg2.f_string = "22222"

	msg3 = test_proto.MessageV2()
	msg3.f_int64 = 33333
	msg3.f_string = "33333"

	msg4 = test_proto.MessageV2()
	msg4.f_int64 = 44444
	msg4.f_string = "44444"

	return [[msg, [msg2, msg3]], msg4]
`,
	"test16.sky": `
test_proto = proto.package("skycfg.test_proto")

# nested list
def main(ctx):
	msg = test_proto.MessageV2()
	msg.f_int64 = 11111
	msg.f_string = "11111"

	msg2 = test_proto.MessageV2()
	msg2.f_int64 = 22222
	msg2.f_string = "22222"

	msg3 = test_proto.MessageV2()
	msg3.f_int64 = 33333
	msg3.f_string = "33333"

	msg4 = test_proto.MessageV2()
	msg4.f_int64 = 44444
	msg4.f_string = "44444"

	return [[msg, [msg2, [msg3]]], msg4]
`,
	"test13_string.sky": `
# nested list
def main(ctx):
	msg = "11111"
	msg2 = "22222"
	msg3 = "33333"

	msgs23 = [msg2, msg3]
	return [msg, msgs23]
`,
	"test14_string.sky": `
# nested list
def main(ctx):
	msg = "11111"
	msg2 = "22222"
	msg3 = "33333"
	msg4 = "44444"

	msgs12 = [msg, msg2]
	msgs34 = [msg3, msg4]
	return [msgs12, msgs34]
`,
	"test15_string.sky": `
# nested list
def main(ctx):
	msg = "11111"
	msg2 = "22222"
	msg3 = "33333"
	msg4 = "44444"

	return [[msg, [msg2, msg3]], msg4]
`,
	"test16_string.sky": `
test_proto = proto.package("skycfg.test_proto")

# nested list
def main(ctx):
	msg = "11111"
	msg2 = "22222"
	msg3 = "33333"
	msg4 = "44444"

	return [[msg, [msg2, [msg3]]], msg4]
`,
	"print/on_load.sky": `
print("hello world")
`,
	"print/main_and_test.sky": `
def test_main(t):
	print("hello world in test")

def main(ctx):
	print("hello world in main")
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
	caseName    string
	fileToLoad  string
	vars        starlark.StringDict
	expLoadErr  bool
	expExecErr  bool
	expProtos   []proto.Message
	execOptions []skycfg.ExecOption
}

type ExecSkycfg func(config *skycfg.Config, testCase endToEndTestCase) ([]proto.Message, error)

func runTestCases(t *testing.T, testCases []endToEndTestCase, execSkycfg ExecSkycfg) {
	loader := &testLoader{}
	ctx := context.Background()

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

		protos, err := execSkycfg(config, testCase)

		if testCase.expExecErr {
			if err == nil {
				t.Error(
					"Bad err result from ExecMain for case", testCase.caseName,
					"\nExpected non-nil",
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

		if len(protos) != len(testCase.expProtos) {
			t.Fatalf("Expected %d protos, got %d", len(testCase.expProtos), len(protos))
		}

		for i := range testCase.expProtos {
			expected := testCase.expProtos[i]
			got := protos[i]
			if !proto.Equal(expected, got) {
				t.Errorf("Test %q: protos differed\nExpected: %s\nGot     : %s", testCase.caseName, expected, got)
			}
		}
	}
}

func TestSkycfgEndToEnd(t *testing.T) {
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
						`{"key1":"value1","key2":"key3=value3","key4":{"key5":"value5","var_key":"var_value"}}`,
					),
					RString: []string{"var_value"},
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
		endToEndTestCase{
			caseName:   "autoboxing primitives",
			fileToLoad: "test7.sky",
			expLoadErr: false,
			expExecErr: false,
			expProtos: []proto.Message{
				&pb.MessageV3{
					F_BoolValue:   &wrappers.BoolValue{Value: true},
					F_StringValue: &wrappers.StringValue{Value: "something"},
					F_DoubleValue: &wrappers.DoubleValue{Value: 3110.4120},
					F_Int32Value:  &wrappers.Int32Value{Value: 110},
					F_Int64Value:  &wrappers.Int64Value{Value: 2148483647},
					F_BytesValue:  &wrappers.BytesValue{Value: []byte("foo/bar/baz")},
					F_Uint32Value: &wrappers.UInt32Value{Value: 4294967295},
					F_Uint64Value: &wrappers.UInt64Value{Value: 8294967295},
					R_StringValue: []*wrappers.StringValue{
						&wrappers.StringValue{Value: "s1"},
						&wrappers.StringValue{Value: "s2"},
						&wrappers.StringValue{Value: "s3"},
					},
				},
			},
		},
		endToEndTestCase{
			caseName:   "flatten nested list",
			fileToLoad: "test13.sky",
			expProtos: []proto.Message{
				&pb.MessageV2{
					FInt64:  proto.Int64(11111),
					FString: proto.String("11111"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(22222),
					FString: proto.String("22222"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(33333),
					FString: proto.String("33333"),
				},
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
		endToEndTestCase{
			caseName:   "flatten nested list 2",
			fileToLoad: "test14.sky",
			expProtos: []proto.Message{
				&pb.MessageV2{
					FInt64:  proto.Int64(11111),
					FString: proto.String("11111"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(22222),
					FString: proto.String("22222"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(33333),
					FString: proto.String("33333"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(44444),
					FString: proto.String("44444"),
				},
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
		endToEndTestCase{
			caseName:   "flatten triple nested list",
			fileToLoad: "test15.sky",
			expProtos: []proto.Message{
				&pb.MessageV2{
					FInt64:  proto.Int64(11111),
					FString: proto.String("11111"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(22222),
					FString: proto.String("22222"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(33333),
					FString: proto.String("33333"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(44444),
					FString: proto.String("44444"),
				},
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
		endToEndTestCase{
			caseName:   "flatten triple nested list 2",
			fileToLoad: "test16.sky",
			expProtos: []proto.Message{
				&pb.MessageV2{
					FInt64:  proto.Int64(11111),
					FString: proto.String("11111"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(22222),
					FString: proto.String("22222"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(33333),
					FString: proto.String("33333"),
				},
				&pb.MessageV2{
					FInt64:  proto.Int64(44444),
					FString: proto.String("44444"),
				},
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
		endToEndTestCase{
			caseName:   "value err when attempting to autobox a too large integer into Int32Value",
			fileToLoad: "test8.sky",
			expExecErr: true,
		},
		endToEndTestCase{
			caseName:   "value err when attempting to autobox a too large integer into Int64Value",
			fileToLoad: "test9.sky",
			expExecErr: true,
		},
		endToEndTestCase{
			caseName:   "value err when attempting to autobox a negative int into UInt64Value",
			fileToLoad: "test10.sky",
			expExecErr: true,
		},
		endToEndTestCase{
			caseName:   "value err when attempting to autobox a too large int into UInt32Value",
			fileToLoad: "test11.sky",
			expExecErr: true,
		},
	}

	fnExecSkycfg := ExecSkycfg(func(config *skycfg.Config, testCase endToEndTestCase) ([]proto.Message, error) {
		execOptions := append(testCase.execOptions, skycfg.WithVars(testCase.vars))
		return config.Main(context.Background(), execOptions...)
	})
	runTestCases(t, testCases, fnExecSkycfg)
}

type endToEndTestCaseNonProtobuf struct {
	caseName    string
	fileToLoad  string
	vars        starlark.StringDict
	expLoadErr  bool
	expExecErr  bool
	expStrings  []string
	execOptions []skycfg.ExecOption
}
type ExecSkycfgNonProtobuf func(config *skycfg.Config, testCase endToEndTestCaseNonProtobuf) ([]string, error)

func runTestCasesNonProtobuf(t *testing.T, testCases []endToEndTestCaseNonProtobuf, execSkycfg ExecSkycfgNonProtobuf) {
	loader := &testLoader{}
	ctx := context.Background()

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

		strings, err := execSkycfg(config, testCase)

		if testCase.expExecErr {
			if err == nil {
				t.Error(
					"Bad err result from ExecMain for case", testCase.caseName,
					"\nExpected non-nil",
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

		if len(strings) != len(testCase.expStrings) {
			t.Fatalf("Expected %d strings, got %d", len(testCase.expStrings), len(strings))
		}

		for i := range testCase.expStrings {
			expected := testCase.expStrings[i]
			got := strings[i]
			if expected != got {
				t.Errorf("Test %q: strings differed\nExpected: %s\nGot     : %s", testCase.caseName, expected, got)
			}
		}
	}
}

func TestSkycfgNonProtobufEndToEnd(t *testing.T) {
	testCases := []endToEndTestCaseNonProtobuf{
		endToEndTestCaseNonProtobuf{
			caseName:   "flatten nested list",
			fileToLoad: "test13_string.sky",
			expStrings: []string{
				"11111",
				"22222",
				"33333",
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
		endToEndTestCaseNonProtobuf{
			caseName:   "flatten nested list 2",
			fileToLoad: "test14_string.sky",
			expStrings: []string{
				"11111",
				"22222",
				"33333",
				"44444",
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
		endToEndTestCaseNonProtobuf{
			caseName:   "flatten triple nested list",
			fileToLoad: "test15_string.sky",
			expStrings: []string{
				"11111",
				"22222",
				"33333",
				"44444",
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
		endToEndTestCaseNonProtobuf{
			caseName:   "flatten triple nested list 2",
			fileToLoad: "test16_string.sky",
			expStrings: []string{
				"11111",
				"22222",
				"33333",
				"44444",
			},
			execOptions: []skycfg.ExecOption{skycfg.WithFlattenLists()},
		},
	}

	fnExecSkycfg := ExecSkycfgNonProtobuf(func(config *skycfg.Config, testCase endToEndTestCaseNonProtobuf) ([]string, error) {
		execOptions := append(testCase.execOptions, skycfg.WithVars(testCase.vars))
		return config.MainNonProtobuf(context.Background(), execOptions...)
	})
	runTestCasesNonProtobuf(t, testCases, fnExecSkycfg)
}

func TestSkycfgLogOutput_Load(t *testing.T) {
	ctx := context.Background()
	loader := new(testLoader)
	var sb strings.Builder
	_, err := skycfg.Load(ctx, "print/on_load.sky",
		skycfg.WithFileReader(loader),
		skycfg.WithLogOutput(&sb),
	)
	if err != nil {
		t.Error(err)
	}
	out := sb.String()
	expected := "[print/on_load.sky:2:6] hello world\n"
	if out != expected {
		t.Errorf("incorrect output: found %q, expected %q", out, expected)
	}
}

func TestSkycfgLogOutput(t *testing.T) {
	ctx := context.Background()
	loader := new(testLoader)
	cfg, err := skycfg.Load(ctx, "print/main_and_test.sky", skycfg.WithFileReader(loader))
	if err != nil {
		t.Error("while loading:", err)
	}

	test := cfg.Tests()[0]

	t.Run("Exec", func(t *testing.T) {
		var sb strings.Builder
		_, err = cfg.Main(ctx, skycfg.WithLogOutput(&sb))
		if err != nil {
			t.Error("while running:", err)
		}
		out := sb.String()
		expected := "[print/main_and_test.sky:6:7] hello world in main\n"
		if out != expected {
			t.Errorf("incorrect output: found %q, expected %q", out, expected)
		}
	})

	t.Run("Test", func(t *testing.T) {
		var sb strings.Builder
		result, err := test.Run(ctx, skycfg.WithLogOutput(&sb))
		if err != nil {
			t.Error("while testing:", err)
		}
		if result.Failure != nil {
			t.Error("while testing:", result.Failure)
		}
		out := sb.String()
		expected := "[print/main_and_test.sky:3:7] hello world in test\n"
		if out != expected {
			t.Errorf("incorrect output: found %q, expected %q", out, expected)
		}
	})
}

func TestSkycfgWithEntryPoint(t *testing.T) {
	testCases := []endToEndTestCase{
		endToEndTestCase{
			caseName:   "all good",
			fileToLoad: "test12.sky",
			expProtos: []proto.Message{
				&pb.MessageV2{
					FInt64:  proto.Int64(12345),
					FString: proto.String("12345"),
				},
			},
		},
	}

	fnExecSkycfg := ExecSkycfg(func(config *skycfg.Config, testCase endToEndTestCase) ([]proto.Message, error) {
		return config.Main(context.Background(), skycfg.WithVars(testCase.vars), skycfg.WithEntryPoint("not_main"))
	})
	runTestCases(t, testCases, fnExecSkycfg)
}

// testTestCase is a test case for the testing functionality built into skycfg
type testTestCase struct {
	errors     bool
	passes     bool
	failureMsg string
}

func TestSkycfgTesting(t *testing.T) {
	loader := &testLoader{}
	ctx := context.Background()

	config, err := skycfg.Load(ctx, "test1.sky", skycfg.WithFileReader(loader))
	if err != nil {
		t.Error("Unexpected error loading test1.sky", err)
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
		"test_main": testTestCase{
			passes: true,
		},
	}

	tests := config.Tests()
	if len(tests) != len(cases) {
		t.Error("Expected %d tests but found %d", len(cases), len(tests))
	}

	for _, test := range tests {
		result, err := test.Run(ctx, skycfg.WithTestVars(starlark.StringDict{
			"var_key": starlark.String("var_value"),
		}))
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

		if result.TestName != test.Name() {
			t.Errorf("TestResult (%s) and Test (%s) should have the same name", result.TestName, test.Name())
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

type flattenStringTestCase struct {
	inputList      *starlark.List
	expectedOutput []string
}

func TestFlattenStringList(t *testing.T) {
	tests := []flattenStringTestCase{
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlark.String("a"),
					starlark.String("b"),
				},
			),
			expectedOutput: []string{"a", "b"},
		},
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlark.String("a"),
					starlark.NewList(
						[]starlark.Value{
							starlark.String("b"),
						},
					),
				},
			),
			expectedOutput: []string{"a", "b"},
		},
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlark.String("a"),
					starlark.NewList(
						[]starlark.Value{
							starlark.String("b"),
							starlark.NewList(
								[]starlark.Value{
									starlark.String("c"),
								},
							),
						},
					),
				},
			),
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlark.NewList(
						[]starlark.Value{
							starlark.String("a"),
							starlark.String("b"),
						},
					),
					starlark.NewList(
						[]starlark.Value{
							starlark.String("c"),
							starlark.String("d"),
						},
					),
				},
			),
			expectedOutput: []string{"a", "b", "c", "d"},
		},
	}
	for _, test := range tests {
		out, err := skycfg.FlattenStringList(test.inputList)
		if err != nil {
			t.Errorf("flattenStringList error: %s", err)
		}
		if !reflect.DeepEqual(out, test.expectedOutput) {
			t.Errorf("flattenStringList assertion failure: expected %v was %v", test.expectedOutput, out)
		}

	}
}

type flattenProtoTestCase struct {
	inputList      *starlark.List
	expectedOutput []proto.Message
}

func TestFlattenProtoList(t *testing.T) {
	protoVal := func(v string) proto.Message {
		return &pb.MessageV2{
			FString: proto.String(v),
		}
	}
	starlarkVal := func(v string) starlark.Value {
		val, err := skycfg.NewProtoMessage(
			protoVal(v),
		)
		if err != nil {
			t.Errorf("NewProtoMessage error: %s", err)
		}
		return val
	}

	tests := []flattenProtoTestCase{
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlarkVal("a"),
				},
			),
			expectedOutput: []proto.Message{
				protoVal("a"),
			},
		},
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlarkVal("a"),
					starlarkVal("b"),
				},
			),
			expectedOutput: []proto.Message{
				protoVal("a"),
				protoVal("b"),
			},
		},
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlarkVal("a"),
					starlark.NewList(
						[]starlark.Value{
							starlarkVal("b"),
						},
					),
				},
			),
			expectedOutput: []proto.Message{
				protoVal("a"),
				protoVal("b"),
			},
		},
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlarkVal("a"),
					starlark.NewList(
						[]starlark.Value{
							starlarkVal("b"),
							starlark.NewList(
								[]starlark.Value{
									starlarkVal("c"),
								},
							),
						},
					),
				},
			),
			expectedOutput: []proto.Message{
				protoVal("a"),
				protoVal("b"),
				protoVal("c"),
			},
		},
		{
			inputList: starlark.NewList(
				[]starlark.Value{
					starlark.NewList(
						[]starlark.Value{
							starlarkVal("a"),
							starlarkVal("b"),
						},
					),
					starlark.NewList(
						[]starlark.Value{
							starlarkVal("c"),
							starlarkVal("d"),
						},
					),
				},
			),
			expectedOutput: []proto.Message{
				protoVal("a"),
				protoVal("b"),
				protoVal("c"),
				protoVal("d"),
			},
		},
	}
	for _, test := range tests {
		out, err := skycfg.FlattenProtoList(test.inputList)
		if err != nil {
			t.Errorf("flattenProtoList error: %s", err)
		}
		if len(out) != len(test.expectedOutput) {
			t.Errorf("flattenProtoList assertion failure: expected %v was %v", test.expectedOutput, out)
		}
		for i, _ := range out {
			if fmt.Sprintf("%v", out[i]) != fmt.Sprintf("%v", test.expectedOutput[i]) {
				t.Errorf("flattenProtoList assertion failure at place %d: expected %q was %q", i, test.expectedOutput[i], out[i])
			}
		}
	}
}
