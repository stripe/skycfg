package parallelmodule

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func TestCallableWithPos(t *testing.T) {
	t.Parallel()

	var callStack starlark.CallStack

	l := new(starlark.List)
	expectedArgs := starlark.Tuple{l}
	expectedKwargs := []starlark.Tuple{{starlark.String("foo"), l}}
	builtin := starlark.NewBuiltin("test", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		assert.Equal(t, expectedArgs, args)
		assert.Equal(t, expectedKwargs, kwargs)
		callStack = thread.CallStack()
		return starlark.None, nil
	})

	filename := "foobar"
	pos := syntax.MakePosition(&filename, 42, 1)
	callable := callableWithPos{builtin, pos}
	_, err := starlark.Call(new(starlark.Thread), callable, expectedArgs, expectedKwargs)
	if assert.NoError(t, err) {
		expectedCallStack := starlark.CallStack{
			{Name: "test", Pos: pos},
		}
		assert.Equal(t, expectedCallStack, callStack)
	}
}

func TestMap_basic_go(t *testing.T) {
	t.Parallel()

	elements := []starlark.Value{
		starlark.String("foo"),
		starlark.String("bar"),
		starlark.String("baz"),
	}

	s := starlark.NewSet(len(elements))
	for _, val := range elements {
		err := s.Insert(val)
		assert.NoError(t, err)
	}

	seen := make(chan starlark.Value, 10)
	seenThreads := make(chan *starlark.Thread, 10)

	done := make(chan struct{})
	go func() {
		defer close(done)

		mapper := starlark.NewBuiltin("mapper", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			seenThreads <- thread
			assert.Len(t, kwargs, 0)
			if assert.Len(t, args, 1) {
				seen <- args[0]
				return args[0], nil
			} else {
				return nil, errors.New("unexpected")
			}
		})
		thread := &starlark.Thread{Name: "root"}
		res, err := starlark.Call(thread, Module.Members["map"], starlark.Tuple{mapper, s}, nil)
		close(seen)
		close(seenThreads)

		if assert.NoError(t, err) {
			if assert.IsType(t, new(starlark.List), res) {
				assert.Equal(t, elements, toFrozenSlice(res.(*starlark.List)))
			}
		}
	}()

	<-done

	var seenSlice []starlark.Value
	for seenArg := range seen {
		seenSlice = append(seenSlice, seenArg)
	}
	assert.ElementsMatch(t, seenSlice, elements)

	seenThreadsSet := map[*starlark.Thread]struct{}{}
	var seenThreadNames []string
	for seenThread := range seenThreads {
		seenThreadNames = append(seenThreadNames, seenThread.Name)
		seenThreadsSet[seenThread] = struct{}{}
	}
	assert.Len(t, seenThreadsSet, len(elements))
	assert.ElementsMatch(t, seenThreadNames, []string{
		"root > parallel.map[0]",
		"root > parallel.map[1]",
		"root > parallel.map[2]",
	})
}

func TestMap_basic(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def mapper(arg):
	return str(arg) + " out"

def run():
	return parallel.map(mapper, range(6))
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_basic", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	out, err := starlark.Call(root, res["run"], nil, nil)
	if assert.NoError(t, err) {
		if assert.IsType(t, new(starlark.List), out) {
			expected := []starlark.Value{
				starlark.String("0 out"),
				starlark.String("1 out"),
				starlark.String("2 out"),
				starlark.String("3 out"),
				starlark.String("4 out"),
				starlark.String("5 out"),
			}
			assert.Equal(t, expected, toFrozenSlice(out.(*starlark.List)))
		}
	}
}

func TestMap_fail(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def mapper_sub(arg):
	if arg == 5:
		fail("oh noes")

def mapper(arg):
	mapper_sub(arg)
	return str(arg) + " out"

def run0():
	parallel.map(mapper, range(6))

def run1():
	run0()

def run2():
	run1()
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_fail", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run2"], nil, nil)
	assert.EqualError(t, err, "fail: oh noes")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := `Traceback (most recent call last):
  TestMap_fail:17:6: in run2
  TestMap_fail:14:6: in run1
  TestMap_fail:11:14: in run0
  <builtin>: in parallel.map
  <builtin>: in parallel.map.fn[5]
  TestMap_fail:7:12: in mapper
  TestMap_fail:4:7: in mapper_sub
Error in fail: fail: oh noes`
		assert.Equal(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

func TestMap_nested(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def mapper(arg):
	return parallel.map(lambda x: -x, [arg] * 4)

def run():
	out = parallel.map(mapper, range(6))
	expected = [
		[0, 0, 0, 0],
		[-1, -1, -1, -1],
		[-2, -2, -2, -2],
		[-3, -3, -3, -3],
		[-4, -4, -4, -4],
		[-5, -5, -5, -5],
	]
	if out != expected:
		fail("{} != {}".format(out, expected))
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_nested", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.NoError(t, err)
}

func TestMap_recurse(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def mapper(arg):
	return parallel.map(lambda x: run(), [arg] * 4)

def run():
	return parallel.map(mapper, range(6))
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_recurse", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.EqualError(t, err, "function run called recursively")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := strings.Join([]string{
			regexp.QuoteMeta(`Traceback (most recent call last):
  TestMap_recurse:6:21: in run
  <builtin>: in parallel.map
  <builtin>: in parallel.map.fn[`), `[0-5]`, regexp.QuoteMeta(`]
  TestMap_recurse:3:21: in mapper
  <builtin>: in parallel.map
  <builtin>: in parallel.map.fn[`), `[0-3]`, regexp.QuoteMeta(`]
  TestMap_recurse:3:35: in lambda
  TestMap_recurse:6:21: in run
Error in parallel.map: function run called recursively`),
		}, "")

		assert.Regexp(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

// This is an example of recursion check not being exhaustive (we allow rec to
// be called on a different thread). But in practice, it is benign: we can
// guarantee that no infinite loop can result from this type of recursion.
func TestMap_recurse_benign(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def rec(x):
	if x < 50:
		return [x]
	return parallel.map(rec, range(4))

def run():
	out = rec(100)
	expected = [
		[0],
		[1],
		[2],
		[3],
	]
	if out != expected:
		fail("{} != {}".format(out, expected))
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_recurse_benign", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.NoError(t, err)
}

func TestMap_mutate_global_during_toplevel(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
d = {}

def mapper(x):
	d[x] = x # this doesn't work

def run():
	return parallel.map(mapper, range(6))

run()
`
	_, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_mutate_global_during_toplevel", prog, starlark.StringDict{"parallel": Module})
	assert.EqualError(t, err, "function parallel.map cannot be called during module top-level initialization")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := `Traceback (most recent call last):
  TestMap_mutate_global_during_toplevel:10:4: in <toplevel>
  TestMap_mutate_global_during_toplevel:8:21: in run
Error in parallel.map: function parallel.map cannot be called during module top-level initialization`

		assert.Equal(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

func TestMap_mutate_global_during_toplevel2(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	runSelf := starlark.NewBuiltin("run_self", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var callable starlark.Callable
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &callable); err != nil {
			return nil, err
		}
		return starlark.Call(thread, callable, nil, nil)
	})
	prog := `
d = {}

def run():
	def fn1(): d[2] = 43
	def fn2(): d[2] = 44
	def fn3(): d[2] = 45

	d[1] = 42 # this works
	return parallel.map(run_self, [fn1, fn2, fn3])

run()
`
	globals := starlark.StringDict{"parallel": Module, "run_self": runSelf}
	_, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_mutate_global_during_toplevel2", prog, globals)
	assert.EqualError(t, err, "function parallel.map cannot be called during module top-level initialization")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := `Traceback (most recent call last):
  TestMap_mutate_global_during_toplevel2:12:4: in <toplevel>
  TestMap_mutate_global_during_toplevel2:10:21: in run
Error in parallel.map: function parallel.map cannot be called during module top-level initialization`

		assert.Equal(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

func TestMap_mutate_global_after_module_eval(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
d = {}

def mapper(x):
	d[x] = x # this doesn't work

def run():
	return parallel.map(mapper, range(6))
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_mutate_global_after_module_eval", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.EqualError(t, err, "cannot insert into frozen hash table")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := strings.Join([]string{
			regexp.QuoteMeta(`Traceback (most recent call last):
  TestMap_mutate_global_after_module_eval:8:21: in run
  <builtin>: in parallel.map
  <builtin>: in parallel.map.fn[`), `[0-5]`, regexp.QuoteMeta(`]
  TestMap_mutate_global_after_module_eval:5:3: in mapper
Error: cannot insert into frozen hash table`),
		}, "")

		assert.Regexp(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

func TestMap_mutate_freevar(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def run():
	d = {}
	def mapper(x):
		d[x] = x # this doesn't work
	d[1] = 42 # this works
	return parallel.map(mapper, range(6))
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_mutate_freevar", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.EqualError(t, err, "cannot insert into frozen hash table")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := strings.Join([]string{
			regexp.QuoteMeta(`Traceback (most recent call last):
  TestMap_mutate_freevar:7:21: in run
  <builtin>: in parallel.map
  <builtin>: in parallel.map.fn[`), `[0-5]`, regexp.QuoteMeta(`]
  TestMap_mutate_freevar:5:4: in mapper
Error: cannot insert into frozen hash table`),
		}, "")

		assert.Regexp(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

func TestMap_mutate_input(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def run():
	def mapper(x):
		x.append(1)
	return parallel.map(mapper, [[]] * 6)
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_mutate_input", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.EqualError(t, err, "append: cannot append to frozen list")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := strings.Join([]string{
			regexp.QuoteMeta(`Traceback (most recent call last):
  TestMap_mutate_input:5:21: in run
  <builtin>: in parallel.map
  <builtin>: in parallel.map.fn[`), `[0-5]`, regexp.QuoteMeta(`]
  TestMap_mutate_input:4:11: in mapper
Error in append: append: cannot append to frozen list`),
		}, "")

		assert.Regexp(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

func TestMap_mutate_input2(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def run():
	l = [[] for _ in range(6)]
	parallel.map(lambda x: None, l)
	l[3].append(1)
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_mutate_input2", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.EqualError(t, err, "append: cannot append to frozen list")
	if assert.IsType(t, new(starlark.EvalError), err) {
		expected := `Traceback (most recent call last):
  TestMap_mutate_input2:5:13: in run
Error in append: append: cannot append to frozen list`

		assert.Equal(t, expected, err.(*starlark.EvalError).Backtrace())
	}
}

// Only the input's elements are frozen, not the entire input iterable.
func TestMap_input_not_frozen(t *testing.T) {
	t.Parallel()

	fileOpts := &syntax.FileOptions{}
	root := &starlark.Thread{Name: "root"}
	prog := `
def assert_equal(x, y):
	if x != y:
		fail("{} != {}".format(x, y))

def run():
	l = [[]] * 6
	out = parallel.map(lambda x: None, l)
	l.append(42) # should work
	assert_equal(out, [None] * 6)
	assert_equal(l, [[]] * 6 + [42])

	d = {1: [], 2: []}
	out = parallel.map(lambda x: 2 * x, d)
	d[1].append(1) # should work
	d[3] = [3] # should work
	assert_equal(out, [2, 4])
	assert_equal(d, {1: [1], 2: [], 3: [3]})
`
	res, err := starlark.ExecFileOptions(fileOpts, root, "TestMap_input_not_frozen", prog, starlark.StringDict{"parallel": Module})
	require.NoError(t, err)

	_, err = starlark.Call(root, res["run"], nil, nil)
	assert.NoError(t, err)
}

// When we have go1.23, we can just do slices.Collect(l.Elements()).
func starlarkListToSlice(l *starlark.List) (out []starlark.Value) {
	iter := l.Iterate()
	defer iter.Done()
	var elem starlark.Value
	for iter.Next(&elem) {
		out = append(out, elem)
	}
	return out
}
