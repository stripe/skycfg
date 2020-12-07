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

package urlmodule

import (
	"fmt"
	"testing"

	"go.starlark.net/starlark"
)

type UrlTestCase struct {
	name      string
	skyExpr   string
	expErr    bool
	expOutput string
}

func TestEncodeQuery(t *testing.T) {
	thread := new(starlark.Thread)
	env := starlark.StringDict{
		"url": NewModule(),
	}

	testCases := []UrlTestCase{
		UrlTestCase{
			name:      "All good",
			skyExpr:   `{"a": "value1 value2", "b": "/test/path"}`,
			expOutput: "a=value1+value2&b=%2Ftest%2Fpath",
		},
		UrlTestCase{
			name:    "Called with a non-dict value",
			skyExpr: "abc",
			expErr:  true,
		},
		UrlTestCase{
			name:    "Key not a string",
			skyExpr: `{5: "a"}`,
			expErr:  true,
		},
		UrlTestCase{
			name:    "Value not a string",
			skyExpr: `{"a": 5}`,
			expErr:  true,
		},
	}

	for _, testCase := range testCases {
		v, err := starlark.Eval(
			thread,
			"<expr>",
			fmt.Sprintf("url.encode_query(%s)", testCase.skyExpr),
			env,
		)
		if testCase.expErr {
			if err == nil {
				t.Error(
					"Bad eval err result for case", testCase.name,
					"\nExpected error",
					"\nGot", err)
			}
		} else {
			if err != nil {
				t.Error(
					"Bad eval err result for case", testCase.name,
					"\nExpected nil",
					"\nGot", err)
			}
			exp := starlark.String(testCase.expOutput)
			if v != exp {
				t.Error(
					"Bad return value for case", testCase.name,
					"\nExpected", exp,
					"\nGot", v,
				)
			}
		}
	}
}
