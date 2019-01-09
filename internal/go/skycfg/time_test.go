package skycfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.starlark.net/starlark"
)

func TestDuration(t *testing.T) {
	thread := new(starlark.Thread)
	env := starlark.StringDict{
		"time": TimeModule(),
	}

	for _, tc := range []struct {
		desc, expr string
		wantErr    string
		wantOut    starlark.Value
	}{
		{
			desc:    "Assign Duration",
			expr:    "time.duration('1s')",
			wantOut: starlark.MakeInt64(1000000000),
		},
		{
			desc:    "No Argument",
			expr:    "time.duration()",
			wantErr: "time.duration: got 0 arguments, want 1",
		},
		{
			desc:    "Empty String",
			expr:    "time.duration('')",
			wantErr: "time: invalid duration ",
		},
		{
			desc:    "Invalid String",
			expr:    "time.duration('1x')",
			wantErr: "time: unknown unit x in duration 1x",
		},
	} {
		gotOut, err := starlark.Eval(thread, "<expr>", tc.expr, env)

		if tc.wantErr != "" {
			assert.EqualError(t, err, tc.wantErr)
		} else {
			assert.Nil(t, err)
		}

		assert.Equal(t, tc.wantOut, gotOut)
	}
}
