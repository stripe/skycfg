package skycfg

import (
	"fmt"
	"testing"

	"github.com/google/skylark"
)

type UrlTestCase struct {
	name      string
	skyExpr   string
	expErr    bool
	expOutput string
}

func TestEncodeQuery(t *testing.T) {
	thread := new(skylark.Thread)
	env := skylark.StringDict{
		"url": urlModule(),
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
		v, err := skylark.Eval(
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
			exp := skylark.String(testCase.expOutput)
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
