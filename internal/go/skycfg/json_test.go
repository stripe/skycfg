package skycfg

import (
	"fmt"
	"testing"

	"go.starlark.net/starlark"
)

type TestCase struct {
	skyExpr   string
	expOutput string
}

func TestSkyToJson(t *testing.T) {
	thread := new(starlark.Thread)
	env := starlark.StringDict{
		"json": JsonModule(),
	}

	testCases := []TestCase{
		TestCase{
			skyExpr:   "123",
			expOutput: "123",
		},
		TestCase{
			skyExpr:   `{"a": 5, 13: 2, "k": {"k2": "v"}}`,
			expOutput: `{"a": 5, 13: 2, "k": {"k2": "v"}}`,
		},
		TestCase{
			skyExpr:   `[1, 2, 3, "abc", None, 15, True, False, {"k": "v"}]`,
			expOutput: `[1, 2, 3, "abc", null, 15, true, false, {"k": "v"}]`,
		},
	}

	for _, testCase := range testCases {
		v, err := starlark.Eval(
			thread,
			"<expr>",
			fmt.Sprintf("json.marshal(%s)", testCase.skyExpr),
			env,
		)
		if err != nil {
			t.Error("Error from eval", "\nExpected nil", "\nGot", err)
		}
		exp := starlark.String(testCase.expOutput)
		if v != exp {
			t.Error(
				"Bad return value from json.marshal",
				"\nExpected",
				exp,
				"\nGot",
				v,
			)
		}
	}
}

func TestSkyToYaml(t *testing.T) {
	thread := new(starlark.Thread)
	env := starlark.StringDict{
		"yaml": YamlModule(),
	}

	testCases := []TestCase{
		TestCase{
			skyExpr: "123",
			expOutput: `123
`,
		},
		TestCase{
			skyExpr: `{"a": 5, 13: 2, "k": {"k2": "v"}}`,
			expOutput: `13: 2
a: 5
k:
  k2: v
`,
		},
		TestCase{
			skyExpr: `[1, 2, 3, "abc", None, 15, True, False, {"k": "v"}]`,
			expOutput: `- 1
- 2
- 3
- abc
- null
- 15
- true
- false
- k: v
`,
		},
	}

	for _, testCase := range testCases {
		v, err := starlark.Eval(
			thread,
			"<expr>",
			fmt.Sprintf("yaml.marshal(%s)", testCase.skyExpr),
			env,
		)
		if err != nil {
			t.Error("Error from eval", "\nExpected nil", "\nGot", err)
		}
		exp := starlark.String(testCase.expOutput)
		if v != exp {
			t.Error(
				"Bad return value from yaml.marshal",
				"\nExpected",
				exp,
				"\nGot",
				v,
			)
		}
	}
}
