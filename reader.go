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

package skycfg

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// A FileReader controls how load() calls resolve and read other modules.
type FileReader interface {
	// Resolve parses the "name" part of load("name", "symbol") to a path. This
	// is not required to correspond to a true path on the filesystem, but should
	// be "absolute" within the semantics of this FileReader.
	//
	// fromPath will be empty when loading the root module passed to Load().
	Resolve(ctx context.Context, name, fromPath string) (path string, err error)

	// ReadFile reads the content of the file at the given path, which was
	// returned from Resolve().
	ReadFile(ctx context.Context, path string) ([]byte, error)
}

type localFileReader struct {
	root string
}

// LocalFileReader returns a [FileReader] that resolves and loads files from
// within a given filesystem directory.
// LocalFileReader expects paths in load() to always use '/' as the separator,
// regardless of the operating system's native path separator.
func LocalFileReader(root string) FileReader {
	if root == "" {
		panic("LocalFileReader: empty root path")
	}
	return &localFileReader{root}
}

func (r *localFileReader) Resolve(ctx context.Context, name, fromPath string) (string, error) {
	if fromPath == "" {
		return name, nil
	}
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return "", fmt.Errorf("load(%q): invalid character in module name", name)
	}
	resolved := filepath.Join(r.root, filepath.FromSlash(path.Clean("/"+name)))
	return resolved, nil
}

func (r *localFileReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

type fsFileReader struct {
	fsys fs.FS
}

// FSFileReader returns a [FileReader] that loads files from the given [fs.FS].
// Path resolution on a FSFileReader is just [path.Clean].
func FSFileReader(fsys fs.FS) FileReader {
	if fsys == nil {
		panic("FSFileReader: nil fsys")
	}
	return &fsFileReader{fsys: fsys}
}

func (r *fsFileReader) Resolve(ctx context.Context, name, fromPath string) (string, error) {
	return path.Clean(name), nil
}

func (r *fsFileReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return fs.ReadFile(r.fsys, path)
}
