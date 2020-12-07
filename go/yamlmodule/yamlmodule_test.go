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

package yamlmodule

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
		"yaml": NewModule(),
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
			fmt.Sprintf("yaml.encode(%s)", testCase.skyExpr),
			env,
		)
		if err != nil {
			t.Error("Error from eval", "\nExpected nil", "\nGot", err)
		}
		exp := starlark.String(testCase.expOutput)
		if v != exp {
			t.Error(
				"Bad return value from yaml.encode",
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
		"yaml": NewModule(),
	}

	skyExpr := `{
        "strKey": "val",
        "arrKey": ["a", "b"],
        "mapKey": {"subkey": "val"},
        "intKey": 2147483647,
        "int64Key": 2147483648,
        "nIntKey": -2147483648,
        "nInt64Key": -2147483649,
        "uintKey": 9223372036854775808,
        "overflowUintKey": 18446744073709551616,
        "floatKey": 1.234,
        "boolKey": False,
        "nullKey": None,
        2147483647: "intKey",
        2147483648: "int64Key",
        -2147483648: "nIntKey",
        -2147483649: "nInt64Key",
        9223372036854775808: "uintKey", 
        1.234: "floatKey",
        False: "boolKey",
        None: "nullKey",
    }`

	v, err := starlark.Eval(
		thread,
		"<expr>",
		fmt.Sprintf(`yaml.decode(yaml.encode(%s))`, skyExpr),
		env,
	)
	if err != nil {
		t.Error("Error from eval", "\nExpected nil", "\nGot", err)
	}
	staryaml := v.(starlark.Mapping)
	for _, testCase := range []struct {
		name        string
		key         starlark.Value
		want        string
		expectedErr error
	}{
		{
			name: "key mapped to String",
			key:  starlark.String("strKey"),
			want: `"val"`,
		},
		{
			name: "key mapped to Array",
			key:  starlark.String("arrKey"),
			want: `["a", "b"]`,
		},
		{
			name: "key mapped to Map",
			key:  starlark.String("mapKey"),
			want: `{"subkey": "val"}`,
		},
		{
			name: "key mapped to Uint",
			key:  starlark.String("uintKey"),
			want: `9223372036854775808`,
		},
		{
			name: "key mapped to negative Int64",
			key:  starlark.String("nInt64Key"),
			want: `-2147483649`,
		},
		{
			name: "key mapped to Int",
			key:  starlark.String("intKey"),
			want: `2147483647`,
		},
		{
			name: "key mapped to Int64",
			key:  starlark.String("int64Key"),
			want: `2147483648`,
		},
		{
			name: "key mapped to Float",
			key:  starlark.String("floatKey"),
			want: `1.234`,
		},
		{
			name: "key mapped to Overflow Uint64",
			key:  starlark.String("overflowUintKey"),
			want: `1.8446744073709552e+19`,
		},
		{
			name: "key mapped to Bool",
			key:  starlark.String("boolKey"),
			want: `False`,
		},
		{
			name: "key mapped to Null",
			key:  starlark.String("nullKey"),
			want: `None`,
		},
		{
			name: "int key mapped to String",
			key:  starlark.MakeInt(2147483647),
			want: `"intKey"`,
		},
		{
			name: "Int64 key mapped to String",
			key:  starlark.MakeInt64(2147483648),
			want: `"int64Key"`,
		},
		{
			name: "negative Int64 key mapped to String",
			key:  starlark.MakeInt64(-2147483649),
			want: `"nInt64Key"`,
		},
		{
			name: "Uint key mapped to String",
			key:  starlark.MakeUint(9223372036854775808),
			want: `"uintKey"`,
		},
		{
			name: "Float key mapped to String",
			key:  starlark.Float(1.234),
			want: `"floatKey"`,
		},
		{
			name: "Bool key mapped to String",
			key:  starlark.Bool(false),
			want: `"boolKey"`,
		},
		{
			name: "Null key mapped to String",
			key:  starlark.None,
			want: `"nullKey"`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, _, err := staryaml.Get(testCase.key)
			if err != nil {
				t.Errorf("error accessing key [%v] in staryaml: %v", testCase.key, err)
			}
			if testCase.want != got.String() {
				t.Error(
					"Bad return value from yaml.decode",
					"\nExpected:",
					testCase.want,
					"\nGot:",
					got,
				)
			}
		})
	}
}
