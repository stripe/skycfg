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

type YamlTestCase struct {
	skyExpr   string
	expOutput string
}

func TestSkyToYaml(t *testing.T) {
	thread := new(starlark.Thread)
	env := starlark.StringDict{
		"yaml": YamlModule(),
	}

	testCases := []YamlTestCase{
		YamlTestCase{
			skyExpr: "123",
			expOutput: `123
`,
		},
		YamlTestCase{
			skyExpr: `{"a": 5, 13: 2, "k": {"k2": "v"}}`,
			expOutput: `13: 2
a: 5
k:
  k2: v
`,
		},
		YamlTestCase{
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

func TestYamlToSky(t *testing.T) {
	thread := new(starlark.Thread)
	env := starlark.StringDict{
		"yaml": YamlModule(),
	}
	v, err := starlark.Eval(
		thread,
		"<expr>",
		`yaml.unmarshal("testdata/test.yml")`,
		env,
	)
	if err != nil {
		t.Error("Error from eval", "\nExpected nil", "\nGot", err)
	}
	staryaml := v.(starlark.Mapping)
	for _, testCase := range []struct {
		name, key, want string
		expectedErr     error
	}{
		{
			name: "key mapped to String",
			key:  "strKey",
			want: `"value"`,
		},
		{
			name: "key mapped to Array",
			key:  "arrKey",
			want: `["a", "b", "c"]`,
		},
		{
			name: "key mapped to Map",
			key:  "mapKey",
			want: `{"subkey": "valuevalue"}`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, _, err := staryaml.Get(starlark.String(testCase.key))
			if err != nil {
				t.Errorf("error accessing key [%v] in staryaml: %v", testCase.key, err)
			}
			if testCase.want != got.String() {
				t.Error(
					"Bad return value from yaml.unmarshal",
					"\nExpected:",
					testCase.want,
					"\nGot:",
					got,
				)
			}
		})
	}
}
