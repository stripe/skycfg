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
