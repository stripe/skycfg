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

package hashmodule

import (
	"fmt"
	"testing"

	"go.starlark.net/starlark"
)

type hashTestCase struct {
	input     string
	hashFunc  string
	expOutput string
}

func TestHashes(t *testing.T) {
	thread := new(starlark.Thread)
	env := starlark.StringDict{
		"hash": NewModule(),
	}

	testCases := []hashTestCase{
		hashTestCase{
			input:     "test md5 string",
			hashFunc:  "md5",
			expOutput: "e3c37791f9070bc459d677d015016f90",
		},
		hashTestCase{
			input:     "test sha1 string",
			hashFunc:  "sha1",
			expOutput: "e4d245f6e79cc13f5a4c0261dfb991438b86fed9",
		},
		hashTestCase{
			input:     "test sha256 string",
			hashFunc:  "sha256",
			expOutput: "a9c78816353b119a0ba2a1281675b147fd47abee11a8d41d5abb739dce8273b7",
		},
		hashTestCase{
			input:     "test murmur3 string",
			hashFunc:  "murmur3",
			expOutput: "91dd87efc613eb2c",
		},
	}

	for _, testCase := range testCases {
		v, err := starlark.Eval(
			thread,
			"<expr>",
			fmt.Sprintf(`hash.%s("%s")`, testCase.hashFunc, testCase.input),
			env,
		)
		if err != nil {
			t.Error("Error from eval", "\nExpected nil", "\nGot", err)
		} else if v != starlark.String(testCase.expOutput) {
			t.Error(
				"Bad result from func", testCase.hashFunc,
				"\nExpected", starlark.String(testCase.expOutput),
				"\nGot", v,
			)
		}
	}
}
