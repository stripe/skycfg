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

// Package kube contains Kubernetes-specific modules and helpers.
package kube

import (
	"go.starlark.net/starlark"
	"k8s.io/apimachinery/pkg/api/resource"

	impl "github.com/stripe/skycfg/internal/go/skycfg"
)

// resourceQuantity returns a Starlark function that parses Kubernetes resource
// quantity string (e.g "1Gi") and returns it as *resource.Quantity wrapped into
// Skycfg proto message.
//
// Meant to be used directly in .sky configs for specifying resource
// requests/limits.
//
// Example:
//
//   core = proto.package('k8s.io.api.core.v1')
//   ...
//   container = core.Container(
//     resources = core.ResourceRequirements(
//       limits = {
//         'memory': kube.res_quantity('1Gi'),
//       }
//     ),
//   ...
func resourceQuantity() starlark.Callable {
	return starlark.NewBuiltin("kube.res_quantity", fnResourceQuantity)
}

// fnResourceQuantity takes Kubernetes resource quantity string and returns
// parsed *resource.Quantity object wrapped into *impl.skyProtoMessage.
func fnResourceQuantity(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var v string
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &v); err != nil {
		return nil, err
	}

	q, err := resource.ParseQuantity(v)
	if err != nil {
		return nil, err
	}

	return impl.NewSkyProtoMessage(&q), nil
}
