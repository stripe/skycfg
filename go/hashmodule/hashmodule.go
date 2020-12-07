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

// Package hashmodule defines a Starlark module of common hash functions.
package hashmodule

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// NewModule returns a Starlark module of common hash functions.
//
//  hash = module(
//    md5,
//    sha1,
//    sha256,
//  )
func NewModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "hash",
		Members: starlark.StringDict{
			"md5":    starlarkMD5,
			"sha1":   starlarkSHA1,
			"sha256": starlarkSHA256,
		},
	}
}

var (
	starlarkMD5    = starlark.NewBuiltin("hash.md5", fnHash(md5.New))
	starlarkSHA1   = starlark.NewBuiltin("hash.sha1", fnHash(sha1.New))
	starlarkSHA256 = starlark.NewBuiltin("hash.sha256", fnHash(sha256.New))
)

// MD5 returns a Starlark function for calculating MD5 checksums.
//
//  >>> hash.md5("hello")
//  "5d41402abc4b2a76b9719d911017c592"
func MD5() starlark.Callable {
	return starlarkMD5
}

// SHA1 returns a Starlark function for calculating SHA1 checksums.
//
//  >>> hash.sha1("hello")
//  "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
func SHA1() starlark.Callable {
	return starlarkSHA1
}

// SHA256 returns a Starlark function for calculating SHA256 checksums.
//
//  >>> hash.sha256("hello")
//  "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
func SHA256() starlark.Callable {
	return starlarkSHA256
}

func fnHash(hash func() hash.Hash) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var s starlark.String
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &s); err != nil {
			return nil, err
		}

		h := hash()
		h.Write([]byte(string(s)))
		return starlark.String(fmt.Sprintf("%x", h.Sum(nil))), nil
	}
}
