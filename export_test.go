// Copyright 2025 The Skycfg Authors.
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

// This _test.go file exists in package skycfg to expose certain helpers to tests,
// which live in package skycfg_test.
// See https://github.com/golang/go/issues/56995 on how this works.

package skycfg

import "sort"

// KeysForTestOnly is a test helper that returns the keys in the [LoadCache].
func (cache *LoadCache) KeysForTestOnly() []string {
	var keys []string
	cache.cache.Range(func(key, _ any) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
