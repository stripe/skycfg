package skycfg

import (
	"fmt"
	"testing"

	"github.com/google/skylark"
)

type hashTestCase struct {
	input     string
	hashFunc  string
	expOutput string
}

func TestHashes(t *testing.T) {
	thread := new(skylark.Thread)
	env := skylark.StringDict{
		"hash": HashModule(),
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
	}

	for _, testCase := range testCases {
		v, err := skylark.Eval(
			thread,
			"<expr>",
			fmt.Sprintf(`hash.%s("%s")`, testCase.hashFunc, testCase.input),
			env,
		)
		if err != nil {
			t.Error("Error from eval", "\nExpected nil", "\nGot", err)
		} else if v != skylark.String(testCase.expOutput) {
			t.Error(
				"Bad result from func", testCase.hashFunc,
				"\nExpected", skylark.String(testCase.expOutput),
				"\nGot", v,
			)
		}
	}
}
