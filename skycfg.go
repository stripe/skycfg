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

// Package skycfg is an extension library for the Starlark language that adds support
// for constructing Protocol Buffer messages.
package skycfg

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkjson"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/stripe/skycfg/go/assertmodule"
	"github.com/stripe/skycfg/go/hashmodule"
	"github.com/stripe/skycfg/go/protomodule"
	"github.com/stripe/skycfg/go/urlmodule"
	"github.com/stripe/skycfg/go/yamlmodule"
)

// Starlark thread-local storage keys.
const (
	contextKey   = "context"   // has type context.Context
	logOutputKey = "logoutput" // has type io.Writer
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

// LocalFileReader returns a FileReader that resolves and loads files from
// within a given filesystem directory.
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
	return ioutil.ReadFile(path)
}

// NewProtoMessage returns a Starlark value representing the given Protobuf
// message. It can be returned back to a proto.Message() via AsProtoMessage().
func NewProtoMessage(msg proto.Message) (starlark.Value, error) {
	return protomodule.NewMessage(msg)
}

// AsProtoMessage returns a Protobuf message underlying the given Starlark
// value, which must have been created by NewProtoMessage(). Returns
// (_, false) if the value is not a valid message.
func AsProtoMessage(v starlark.Value) (proto.Message, bool) {
	return protomodule.AsProtoMessage(v)
}

// NewProtoPackage returns a Starlark value representing the given Protobuf
// package. It can be added to global symbols when loading a Skycfg config file.
func NewProtoPackage(r unstableProtoRegistryV2, name string) starlark.Value {
	return protomodule.NewProtoPackage(r.UnstableProtobufTypes(), protoreflect.FullName(name))
}

// A Config is a Skycfg config file that has been fully loaded and is ready
// for execution.
type Config struct {
	filename string
	globals  starlark.StringDict
	locals   starlark.StringDict
	tests    []*Test
}

type commonOptions struct {
	logOutput io.Writer
}

// A CommonOption is an option that can be applied to Load, Config.Main, and Test.Run.
type CommonOption interface {
	LoadOption
	ExecOption
	TestOption
}

type fnCommonOption func(options *commonOptions)

func (fn fnCommonOption) applyLoad(opts *loadOptions) {
	fn(&opts.commonOptions)
}

func (fn fnCommonOption) applyExec(opts *execOptions) {
	fn(&opts.commonOptions)
}

func (fn fnCommonOption) applyTest(opts *testOptions) {
	fn(&opts.commonOptions)
}

// WithLogOutput changes the destination of print() function calls in Starlark code.
// If nil, os.Stderr will be used.
func WithLogOutput(w io.Writer) CommonOption {
	return fnCommonOption(func(opts *commonOptions) {
		opts.logOutput = w
	})
}

// A LoadOption adjusts details of how Skycfg configs are loaded.
type LoadOption interface {
	applyLoad(*loadOptions)
}

type loadOptions struct {
	commonOptions
	globals       starlark.StringDict
	fileReader    FileReader
	protoRegistry unstableProtoRegistryV2
}

type fnLoadOption func(*loadOptions)

func (fn fnLoadOption) applyLoad(opts *loadOptions) { fn(opts) }

// WithGlobals adds additional global symbols to the Starlark environment
// when loading a Skycfg config.
func WithGlobals(globals starlark.StringDict) LoadOption {
	return fnLoadOption(func(opts *loadOptions) {
		for key, value := range globals {
			opts.globals[key] = value
		}
	})
}

// WithFileReader changes the implementation of load() when loading a
// Skycfg config.
func WithFileReader(r FileReader) LoadOption {
	if r == nil {
		panic("WithFileReader: nil reader")
	}
	return fnLoadOption(func(opts *loadOptions) {
		opts.fileReader = r
	})
}

type unstableProtoRegistryV2 interface {
	// UNSTABLE go-protobuf v2 type registry
	UnstableProtobufTypes() *protoregistry.Types
}

type protoRegistryWrapper struct {
	protoRegistry *protoregistry.Types
}

var _ (unstableProtoRegistryV2) = (*protoRegistryWrapper)(nil)

func (pr *protoRegistryWrapper) UnstableProtobufTypes() *protoregistry.Types {
	return pr.protoRegistry
}

func NewUnstableProtobufRegistryV2(r *protoregistry.Types) unstableProtoRegistryV2 {
	return &protoRegistryWrapper{r}
}

// WithProtoRegistry is an EXPERIMENTAL and UNSTABLE option to override
// how Protobuf message type names are mapped to Go types.
func WithProtoRegistry(r unstableProtoRegistryV2) LoadOption {
	if r == nil {
		panic("WithProtoRegistry: nil registry")
	}
	return fnLoadOption(func(opts *loadOptions) {
		opts.protoRegistry = r
	})
}

// UnstablePredeclaredModules returns a Starlark string dictionary with
// predeclared Skycfg modules which can be used in starlark.ExecFile.
//
// Takes in unstableProtoRegistry as param (if nil will use standard proto
// registry).
//
// Currently provides these modules (see REAMDE for more detailed description):
//   - fail   - interrupts execution and prints a stacktrace.
//   - hash   - supports md5, sha1 and sha245 functions.
//   - json   - marshals plain values (dicts, lists, etc) to JSON.
//   - proto  - package for constructing Protobuf messages.
//   - struct - experimental Starlark struct support.
//   - yaml   - same as "json" package but for YAML.
//   - url    - utility package for encoding URL query string.
func UnstablePredeclaredModules(r unstableProtoRegistryV2) starlark.StringDict {
	return starlark.StringDict{
		"fail":   assertmodule.Fail,
		"hash":   hashmodule.NewModule(),
		"json":   newJsonModule(),
		"proto":  UnstableProtoModule(r),
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
		"yaml":   newYamlModule(),
		"url":    urlmodule.NewModule(),
	}
}

func newJsonModule() starlark.Value {
	// Copy starlarkjson and add compatibility fields
	starlarjsonModule := starlarkjson.Module
	module := &starlarkstruct.Module{
		Name:    starlarjsonModule.Name,
		Members: make(starlark.StringDict),
	}
	for k, v := range starlarjsonModule.Members {
		module.Members[k] = v
	}

	// Aliases for compatibility with pre-v1.0 Skycfg API.
	module.Members["marshal"] = module.Members["encode"]
	module.Members["unmarshal"] = module.Members["decode"]

	return module
}

func newYamlModule() starlark.Value {
	module := yamlmodule.NewModule()

	// Aliases for compatibility with pre-v1.0 Skycfg API.
	module.Members["marshal"] = module.Members["encode"]
	module.Members["unmarshal"] = module.Members["decode"]

	return module
}

func UnstableProtoModule(r unstableProtoRegistryV2) starlark.Value {
	protoTypes := protoregistry.GlobalTypes
	if r != nil {
		protoTypes = r.UnstableProtobufTypes()
	}

	protoModule := protomodule.NewModule(protoTypes)

	// Compatibility aliases
	protoModule.Members["from_json"] = protoModule.Members["decode_json"]
	protoModule.Members["from_text"] = protoModule.Members["decode_text"]
	protoModule.Members["to_any"] = protoModule.Members["encode_any"]
	protoModule.Members["to_json"] = protoModule.Members["encode_json"]
	protoModule.Members["to_text"] = protoModule.Members["encode_text"]

	return protoModule
}

// Load reads a Skycfg config file from the filesystem.
func Load(ctx context.Context, filename string, opts ...LoadOption) (*Config, error) {
	parsedOpts := &loadOptions{
		globals:    starlark.StringDict{},
		fileReader: LocalFileReader(filepath.Dir(filename)),
	}
	for _, opt := range opts {
		opt.applyLoad(parsedOpts)
	}

	overriddenGlobals := parsedOpts.globals
	parsedOpts.globals = UnstablePredeclaredModules(parsedOpts.protoRegistry)
	for key, value := range overriddenGlobals {
		parsedOpts.globals[key] = value
	}
	configLocals, tests, err := loadImpl(ctx, parsedOpts, filename)
	if err != nil {
		return nil, err
	}
	return &Config{
		filename: filename,
		globals:  parsedOpts.globals,
		locals:   configLocals,
		tests:    tests,
	}, nil
}

func loadImpl(ctx context.Context, opts *loadOptions, filename string) (starlark.StringDict, []*Test, error) {
	reader := opts.fileReader

	type cacheEntry struct {
		globals starlark.StringDict
		err     error
	}
	cache := make(map[string]*cacheEntry)
	tests := []*Test{}

	load := func(thread *starlark.Thread, moduleName string) (starlark.StringDict, error) {
		var fromPath string
		if thread.CallStackDepth() > 0 {
			fromPath = thread.CallFrame(0).Pos.Filename()
		}
		modulePath, err := reader.Resolve(ctx, moduleName, fromPath)
		if err != nil {
			return nil, err
		}

		e, ok := cache[modulePath]
		if e != nil {
			return e.globals, e.err
		}
		if ok {
			return nil, fmt.Errorf("cycle in load graph")
		}
		moduleSource, err := reader.ReadFile(ctx, modulePath)
		if err != nil {
			cache[modulePath] = &cacheEntry{nil, err}
			return nil, err
		}

		cache[modulePath] = nil
		globals, err := starlark.ExecFile(thread, modulePath, moduleSource, opts.globals)
		cache[modulePath] = &cacheEntry{globals, err}

		for name, val := range globals {
			if !strings.HasPrefix(name, "test_") {
				continue
			}
			if fn, ok := val.(starlark.Callable); ok {
				tests = append(tests, &Test{
					callable: fn,
				})
			}
		}
		return globals, err
	}
	thread := &starlark.Thread{
		Print: skyPrint,
		Load:  load,
	}
	thread.SetLocal(logOutputKey, opts.logOutput)
	locals, err := load(thread, filename)
	return locals, tests, err
}

// Filename returns the original filename passed to Load().
func (c *Config) Filename() string {
	return c.filename
}

// Globals returns the set of variables in the Starlark global namespace,
// including any added to the config loader by WithGlobals().
func (c *Config) Globals() starlark.StringDict {
	return c.globals
}

// Locals returns the set of variables in the Starlark local namespace for
// the top-level module.
func (c *Config) Locals() starlark.StringDict {
	return c.locals
}

// An ExecOption adjusts details of how a Skycfg config's main function is
// executed.
type ExecOption interface {
	applyExec(*execOptions)
}

type execOptions struct {
	commonOptions
	vars         *starlark.Dict
	funcName     string
	flattenLists bool
}

type fnExecOption func(*execOptions)

func (fn fnExecOption) applyExec(opts *execOptions) { fn(opts) }

// WithVars adds key:value pairs to the ctx.vars dict passed to main().
func WithVars(vars starlark.StringDict) ExecOption {
	return fnExecOption(func(opts *execOptions) {
		for key, value := range vars {
			opts.vars.SetKey(starlark.String(key), value)
		}
	})
}

// WithEntryPoint changes the name of the Skycfg function to execute.
func WithEntryPoint(name string) ExecOption {
	return fnExecOption(func(opts *execOptions) {
		opts.funcName = name
	})
}

// WithFlattenLists flatten lists one layer deep ([1, [2,3]] -> [1, 2, 3])
func WithFlattenLists() ExecOption {
	return fnExecOption(func(opts *execOptions) {
		opts.flattenLists = true
	})
}

// Main executes main() or a custom entry point function from the top-level Skycfg config
// module, which is expected to return either None or a list of Protobuf messages.
func (c *Config) Main(ctx context.Context, opts ...ExecOption) ([]proto.Message, error) {
	parsedOpts := &execOptions{
		vars:     &starlark.Dict{},
		funcName: "main",
	}
	for _, opt := range opts {
		opt.applyExec(parsedOpts)
	}
	mainVal, ok := c.locals[parsedOpts.funcName]
	if !ok {
		return nil, fmt.Errorf("no %q function found in %q", parsedOpts.funcName, c.filename)
	}
	main, ok := mainVal.(starlark.Callable)
	if !ok {
		return nil, fmt.Errorf("%q must be a function (got a %s)", parsedOpts.funcName, mainVal.Type())
	}

	thread := &starlark.Thread{
		Print: skyPrint,
	}
	thread.SetLocal(contextKey, ctx)
	thread.SetLocal(logOutputKey, parsedOpts.logOutput)
	mainCtx := &starlarkstruct.Module{
		Name: "skycfg_ctx",
		Members: starlark.StringDict(map[string]starlark.Value{
			"vars": parsedOpts.vars,
		}),
	}
	args := starlark.Tuple([]starlark.Value{mainCtx})
	mainVal, err := starlark.Call(thread, main, args, nil)
	if err != nil {
		return nil, err
	}
	mainList, ok := mainVal.(*starlark.List)
	if !ok {
		if _, isNone := mainVal.(starlark.NoneType); isNone {
			return nil, nil
		}
		return nil, fmt.Errorf("%q didn't return a list (got a %s)", parsedOpts.funcName, mainVal.Type())
	}
	var msgs []proto.Message
	for ii := 0; ii < mainList.Len(); ii++ {
		maybeMsg := mainList.Index(ii)
		// Flatten lists recursively. [[1, 2], 3] => [1, 2, 3]
		if maybeMsgList, ok := maybeMsg.(*starlark.List); parsedOpts.flattenLists && ok {
			flattened, err := FlattenProtoList(maybeMsgList)
			if err != nil {
				return nil, fmt.Errorf("%q returned something that's not a protobuf within a nested list %w", parsedOpts.funcName, err)
			}
			msgs = append(msgs, flattened...)
		} else {
			msg, ok := AsProtoMessage(maybeMsg)
			if !ok {
				return nil, fmt.Errorf("%q returned something that's not a protobuf (a %s)", parsedOpts.funcName, maybeMsg.Type())
			}
			msgs = append(msgs, msg)
		}
	}
	return msgs, nil
}

func FlattenProtoList(list *starlark.List) ([]proto.Message, error) {
	var flattened []proto.Message
	for i := 0; i < list.Len(); i++ {
		v := list.Index(i)
		if l, ok := v.(*starlark.List); ok {
			recursiveFlattened, err := FlattenProtoList(l)
			if err != nil {
				return flattened, err
			}
			flattened = append(flattened, recursiveFlattened...)
			continue
		}
		if msg, ok := AsProtoMessage(v); ok {
			flattened = append(flattened, msg)
		} else {
			return flattened, fmt.Errorf("list contains object which is not a protobuf (got %s)", v.Type())
		}
	}
	return flattened, nil
}

// A TestResult is the result of a test run
type TestResult struct {
	TestName string
	Failure  error
	Duration time.Duration
}

// A Test is a test case, which is a skycfg function whose name starts with `test_`.
type Test struct {
	callable starlark.Callable
}

// Name returns the name of the test (the name of the function)
func (t *Test) Name() string {
	return t.callable.Name()
}

// An TestOption adjusts details of how a Skycfg config's test functions are
// executed.
type TestOption interface {
	applyTest(*testOptions)
}

type testOptions struct {
	commonOptions
	vars *starlark.Dict
}

type fnTestOption func(*testOptions)

func (fn fnTestOption) applyTest(opts *testOptions) { fn(opts) }

// WithTestVars adds key:value pairs to the ctx.vars dict passed to tests
func WithTestVars(vars starlark.StringDict) TestOption {
	return fnTestOption(func(opts *testOptions) {
		for key, value := range vars {
			opts.vars.SetKey(starlark.String(key), value)
		}
	})
}

// Run actually executes a test. It returns a TestResult if the test completes (even if it fails)
// The error return value will only be non-nil if the test execution itself errors.
func (t *Test) Run(ctx context.Context, opts ...TestOption) (*TestResult, error) {
	parsedOpts := &testOptions{
		vars: &starlark.Dict{},
	}
	for _, opt := range opts {
		opt.applyTest(parsedOpts)
	}

	thread := &starlark.Thread{
		Print: skyPrint,
	}
	thread.SetLocal(contextKey, ctx)
	thread.SetLocal(logOutputKey, parsedOpts.logOutput)

	assertModule := assertmodule.AssertModule()
	testCtx := &starlarkstruct.Module{
		Name: "skycfg_test_ctx",
		Members: starlark.StringDict(map[string]starlark.Value{
			"vars":   parsedOpts.vars,
			"assert": assertModule,
		}),
	}
	args := starlark.Tuple([]starlark.Value{testCtx})

	result := TestResult{
		TestName: t.Name(),
	}

	startTime := time.Now()
	_, err := starlark.Call(thread, t.callable, args, nil)
	result.Duration = time.Since(startTime)
	if err != nil {
		// if there is no assertion error, there was something wrong with the execution itself
		if len(assertModule.Failures) == 0 {
			return nil, err
		}

		// there should only be one failure, because each test run gets its own *TestContext
		// and each assertion failure halts execution.
		if len(assertModule.Failures) > 1 {
			panic("A test run should only have one assertion failure. Something went wrong with the test infrastructure.")
		}
		result.Failure = assertModule.Failures[0]
	}

	return &result, nil
}

// Tests returns all tests defined in the config
func (c *Config) Tests() []*Test {
	return c.tests
}

func skyPrint(t *starlark.Thread, msg string) {
	var out io.Writer = os.Stderr
	if w := t.Local(logOutputKey); w != nil {
		out = w.(io.Writer)
	}
	fmt.Fprintf(out, "[%v] %s\n", t.CallFrame(1).Pos, msg)
}

// MainNonProtobuf executes main() or a custom entry point function from the top-level Skycfg config
// module, which is expected to return either None or a list of strings, and NOT protobuf. If the rendered
// entry point returns nested lists, then they are flattened. This is expected to be used
// for Skycfg files which do not return protobufs (e.g. stringified YAML) which is then passed downstream
// to other systems which process the string output.
func (c *Config) MainNonProtobuf(ctx context.Context, opts ...ExecOption) ([]string, error) {
	parsedOpts := &execOptions{
		vars:     &starlark.Dict{},
		funcName: "main",
	}
	for _, opt := range opts {
		opt.applyExec(parsedOpts)
	}
	mainVal, ok := c.locals[parsedOpts.funcName]
	if !ok {
		return nil, fmt.Errorf("no %q function found in %q", parsedOpts.funcName, c.filename)
	}
	main, ok := mainVal.(starlark.Callable)
	if !ok {
		return nil, fmt.Errorf("%q must be a function (got a %s)", parsedOpts.funcName, mainVal.Type())
	}

	thread := &starlark.Thread{
		Print: skyPrint,
	}
	thread.SetLocal(contextKey, ctx)
	thread.SetLocal(logOutputKey, parsedOpts.logOutput)
	mainCtx := &starlarkstruct.Module{
		Name: "skycfg_ctx",
		Members: starlark.StringDict(map[string]starlark.Value{
			"vars": parsedOpts.vars,
		}),
	}
	args := starlark.Tuple([]starlark.Value{mainCtx})
	mainVal, err := starlark.Call(thread, main, args, nil)
	if err != nil {
		return nil, err
	}
	mainList, ok := mainVal.(*starlark.List)
	if !ok {
		if _, isNone := mainVal.(starlark.NoneType); isNone {
			return nil, nil
		}
		return nil, fmt.Errorf("%q didn't return a list (got a %s)", parsedOpts.funcName, mainVal.Type())
	}
	var msgs []string
	for ii := 0; ii < mainList.Len(); ii++ {
		so := mainList.Index(ii)
		if ss, ok := so.(starlark.String); ok {
			msgs = append(msgs, ss.GoString())
		} else {
			// Flatten lists recursively. [[1, 2], 3] => [1, 2, 3]
			if maybeMsgList, ok := so.(*starlark.List); parsedOpts.flattenLists && ok {
				flattened, err := FlattenStringList(maybeMsgList)
				if err != nil {
					return msgs, fmt.Errorf("%q returned something that's not of type string or list within a nested list %w", parsedOpts.funcName, err)
				}
				msgs = append(msgs, flattened...)
			} else {
				return msgs, fmt.Errorf("%q returned an object inside list not of type String (got %s)", parsedOpts.funcName, so.Type())
			}
		}
	}
	return msgs, nil
}

func FlattenStringList(list *starlark.List) ([]string, error) {
	var flattened []string
	for i := 0; i < list.Len(); i++ {
		v := list.Index(i)
		if l, ok := v.(*starlark.List); ok {
			recursiveFlattened, err := FlattenStringList(l)
			if err != nil {
				return flattened, err
			}
			flattened = append(flattened, recursiveFlattened...)
			continue
		} else if msg, ok := v.(starlark.String); ok {
			flattened = append(flattened, msg.GoString())
		} else {
			return flattened, fmt.Errorf("list contains object not of type string (got %s)", v.Type())
		}
	}
	return flattened, nil
}
