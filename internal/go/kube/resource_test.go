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

package kube

import (
	"testing"

	"go.starlark.net/starlark"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stripe/skycfg"
)

func TestResourceQuantity(t *testing.T) {
	globals := starlark.StringDict{
		"kube": KubeModule(),
	}
	for _, tc := range []struct {
		desc          string
		expr          string
		wantQ         *resource.Quantity
		wantErrString string
	}{
		{
			desc:  "5Gi of memory",
			expr:  "kube.res_quantity('5Gi')",
			wantQ: resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
		},
		{
			desc:  "5G of disk",
			expr:  "kube.res_quantity('5G')",
			wantQ: resource.NewQuantity(5*1000*1000*1000, resource.DecimalSI),
		},
		{
			desc:  "5.3 cpu",
			expr:  "kube.res_quantity('5300m')",
			wantQ: resource.NewMilliQuantity(5300, resource.DecimalSI),
		},
		{
			desc:          "invalid quantity regex",
			expr:          "kube.res_quantity('foobar')",
			wantErrString: resource.ErrFormatWrong.Error(),
		},
		{
			desc:          "empty quantity",
			expr:          "kube.res_quantity('')",
			wantErrString: resource.ErrFormatWrong.Error(),
		},
		{
			desc:          "no argument",
			expr:          "kube.res_quantity()",
			wantErrString: "kube.res_quantity: got 0 arguments, want 1",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			v, err := starlark.Eval(&starlark.Thread{}, "", tc.expr, globals)

			var gotErrString string
			if err != nil {
				gotErrString = err.(*starlark.EvalError).Msg
			}

			if tc.wantErrString != gotErrString {
				t.Fatalf("Unexpected error: (-want +got):\n\t-: %s\n\t+: %s", tc.wantErrString, gotErrString)
			}

			if tc.wantErrString == "" {
				gotQ, ok := skycfg.AsProtoMessage(v)
				if !ok {
					t.Fatalf("Invalid return value. Expected `*skycfg.skyProtoMessage', got: %v", v.Type())
				}

				if tc.wantQ.String() != gotQ.String() {
					t.Fatalf("Unexpected error: (-want +got):\n\t-: %v\n\t+: %v", tc.wantQ, gotQ)
				}
			}
		})
	}
}
