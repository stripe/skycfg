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
	"time"

	"go.starlark.net/starlark"
)

// TimeModule returns a new starlark.Value representing time Skycfg module.
func TimeModule() starlark.Value {
	return &Module{
		Name: "time",
		Attrs: starlark.StringDict{
			"duration": starlark.NewBuiltin("time.duration", fnDuration),
		},
	}
}

func fnDuration(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}

	return starlark.MakeInt64(int64(d)), nil
}
