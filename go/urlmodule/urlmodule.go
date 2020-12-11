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

// Package urlmodule defines a Starlark module of URL-related functions.
package urlmodule

import (
	"fmt"
	"net/url"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// NewModule returns a Starlark module of URL-related functions.
//
//  url = module(
//    encode_query,
//  )
//
// See `docs/modules.asciidoc` for details on the API of each function.
func NewModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "url",
		Members: starlark.StringDict{
			"encode_query": starlark.NewBuiltin("url.encode_query", encodeQuery),
		},
	}
}

func encodeQuery(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var d *starlark.Dict
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &d); err != nil {
		return nil, err
	}

	urlVals := url.Values{}

	for _, itemPair := range d.Items() {
		key := itemPair[0]
		value := itemPair[1]

		keyStr, keyIsStr := key.(starlark.String)
		if !keyIsStr {
			return nil, fmt.Errorf("Key is not string: %+v", key)
		}

		valStr, valIsStr := value.(starlark.String)
		if !valIsStr {
			return nil, fmt.Errorf("Value is not string: %+v", value)
		}

		urlVals.Add(string(keyStr), string(valStr))
	}

	return starlark.String(urlVals.Encode()), nil
}
